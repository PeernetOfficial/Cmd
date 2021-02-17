/*
File Name:  Network IPv4 Broadcast.go
Copyright:  2021 Peernet Foundation s.r.o.
Author:     Peter Kleissner

IPv4 Multicast just sucks (can't use socket bound to 0.0.0.0:PortMain and send to 224.0.0.1:PortMulticast), so we rely on Broadcast instead.
*/

package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/libp2p/go-reuseport"
)

const ipv4BroadcastPort = 12911

// special Public-Private Key pair for local discovery
var ipv4BroadcastPrivateKey *btcec.PrivateKey
var ipv4BroadcastPublicKey *btcec.PublicKey

const ipv4BroadcastPrivateKeyH = "5e27ecc8e54a24e71dca9ba84a9bf465400e27b8c46a977d34962d3d88558c8e"

func initBroadcastIPv4() {
	if configPK, err := hex.DecodeString(ipv4BroadcastPrivateKeyH); err == nil {
		ipv4BroadcastPrivateKey, ipv4BroadcastPublicKey = btcec.PrivKeyFromBytes(btcec.S256(), configPK)
	}
}

// BroadcastIPv4Join prepares sending Broadcasts
func (network *network) BroadcastIPv4() (err error) {
	if ipv4BroadcastPrivateKey == nil || ipv4BroadcastPublicKey == nil {
		return
	}

	// listen on a special socket
	network.broadcastSocket, err = reuseport.ListenPacket("udp4", net.JoinHostPort(network.address.IP.String(), strconv.Itoa(ipv4BroadcastPort)))
	if err != nil {
		return err
	}

	go network.BroadcastIPv4Listen()
	go network.BroadcastIPv4Send()

	if network.ipnet == nil {
		go network.BroadcastIPv4SendAllInterfaces()
	}

	return nil
}

// BroadcastIPv4Listen listens for incoming broadcast packets
// Fork from network.Listen! Keep any changes synced.
func (network *network) BroadcastIPv4Listen() {
	for {
		// Buffer: Must be created for each packet as it is passed as pointer.
		// If the buffer is too small, ReadFromUDP only reads until its length and returns this error: "wsarecvfrom: A message sent on a datagram socket was larger than the internal message buffer or some other network limit, or the buffer used to receive a datagram into was smaller than the datagram itself."
		buffer := make([]byte, maxPacketSize)
		length, sender, err := network.broadcastSocket.ReadFrom(buffer)

		if err != nil {
			log.Printf("Listen Error receiving UDP message: %v\n", err) // Only log for debug purposes.
			time.Sleep(time.Millisecond * 50)                           // In case of endless errors, prevent ddos of CPU.
			continue
		}

		fmt.Printf("Broadcast from %s\n", sender.String())

		if length < packetLengthMin {
			// Discard packets that do not meet the minimum length.
			continue
		}

		// send the packet to a channel which is processed by multiple workers.
		rawPacketsIncoming <- networkWire{network: network, sender: sender.(*net.UDPAddr), raw: buffer[:length], receiverPublicKey: ipv4BroadcastPublicKey}
	}
}

// BroadcastIPv4Send sends out broadcast messages to discover peers at the same site
func (network *network) BroadcastIPv4Send() {
	time.Sleep(1 * time.Second)
	var broadcastIPs []net.IP

	broadcastIPs = append(broadcastIPs, net.IPv4bcast)

	// auto detect the broadcast IPs
	if network.ipnet != nil {
		if ip2 := ipv4DirectedBroadcast(network.ipnet); ip2 != nil {
			broadcastIPs = append(broadcastIPs, ip2)
		}
	}

	for {
		raw, err := packetEncrypt(peerPrivateKey, ipv4BroadcastPublicKey, &packetRaw{Protocol: 0, Command: 0})
		if err != nil {
			log.Printf("Error creating packet: %s\n", err.Error())
			time.Sleep(time.Second * 10)
			continue
		}

		// send out the wire
		for _, ip := range broadcastIPs {
			err = network.send(ip, ipv4BroadcastPort, raw)
			if err != nil {
				log.Printf("Error sending UDP packet: %v\n", err)
			}
		}

		time.Sleep(time.Second * 10)
	}
}

// BroadcastIPv4SendAllInterfaces sends out broadcast messages to all network interfaces
func (network *network) BroadcastIPv4SendAllInterfaces() {
	time.Sleep(1 * time.Second)
	var broadcastNets []*net.IPNet

	interfaceList, err := net.Interfaces()
	if err != nil {
		return
	}

	for _, iface := range interfaceList {
		addresses, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, address := range addresses {
			net1 := address.(*net.IPNet)

			if !IsIPv4(net1.IP) || rfc3927Net.Contains(net1.IP) {
				continue
			}

			broadcastNets = append(broadcastNets, net1)
		}
	}

	for {
		raw, err := packetEncrypt(peerPrivateKey, ipv4BroadcastPublicKey, &packetRaw{Protocol: 0, Command: 0})
		if err != nil {
			log.Printf("Error creating packet: %s\n", err.Error())
			time.Sleep(time.Second * 10)
			continue
		}

		// send out the wire
		for _, net1 := range broadcastNets {
			ipBroadcast := ipv4DirectedBroadcast(net1)
			if ipBroadcast == nil {
				continue
			}

			err = network.send(ipBroadcast, ipv4BroadcastPort, raw)
			if err != nil {
				log.Printf("Error sending UDP packet: %v\n", err)
			}
		}

		time.Sleep(time.Second * 10)
	}
}

func ipv4DirectedBroadcast(n *net.IPNet) net.IP {
	ip4 := n.IP.To4()
	if ip4 == nil {
		return nil
	}
	last := make(net.IP, len(ip4))
	copy(last, ip4)
	for i := range ip4 {
		last[i] |= ^n.Mask[i]
	}
	return last
}

var (
	// rfc3927Net specifies the IPv4 auto configuration address block as
	// defined by RFC3927 (169.254.0.0/16).
	rfc3927Net = ipNet("169.254.0.0", 16, 32)
)

// ipNet returns a net.IPNet struct given the passed IP address string, number
// of one bits to include at the start of the mask, and the total number of bits
// for the mask.
func ipNet(ip string, ones, bits int) net.IPNet {
	return net.IPNet{IP: net.ParseIP(ip), Mask: net.CIDRMask(ones, bits)}
}
