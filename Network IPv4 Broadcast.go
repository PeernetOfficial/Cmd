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
	//network.broadcastSocket, err = net.DialUDP("udp4", nil, &net.UDPAddr{IP: network.address.IP, Port: ipv4MulticastPort})
	network.broadcastSocket, err = reuseport.ListenPacket("udp4", net.JoinHostPort(network.address.IP.String(), strconv.Itoa(ipv4BroadcastPort)))
	if err != nil {
		return err
	}

	go network.BroadcastIPv4Listen()
	go network.BroadcastIPv4Send()

	return nil
}

// BroadcastIPv4Listen listens for incoming multicast packets
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

// BroadcastIPv4Send sends out multicast messages to discover peers at the same site
func (network *network) BroadcastIPv4Send() {
	time.Sleep(1 * time.Second)
	var broadcastIPs []net.IP

	broadcastIPs = append(broadcastIPs, net.ParseIP("255.255.255.255"))

	// auto detect the braodcast IPs
	//if network.iface != nil {
	//} else {
	//	interfaceList, err := net.Interfaces()
	//}

	for {
		raw, err := packetEncrypt(peerPrivateKey, ipv4BroadcastPublicKey, &packetRaw{Protocol: 0, Command: 0})
		if err != nil {
			log.Printf("Error creating packet: %s\n", err.Error())
			time.Sleep(time.Second * 10)
			continue
		}

		// send out the wire
		for _, ip := range broadcastIPs {
			err = network.send(ip, ipv6MulticastPort, raw)
			if err != nil {
				log.Printf("Error sending UDP packet: %v\n", err)
			}
		}

		time.Sleep(time.Second * 10)
	}
}

/*
// MulticastIPv4Send sends out multicast messages to discover peers at the same site
func (network *network) MulticastIPv4Send() {
	time.Sleep(1 * time.Second)

	// prepare sockets for sending out the multicast messages
	var sockets []*net.UDPConn

	interfaceMulticastSocketOpen := func(iface *net.Interface) (err error) {
		addresses, err := iface.Addrs()
		if err != nil {
			return err
		}

		// iterate through all IPs of the interfaces
		for _, address := range addresses {
			addressIP := address.(*net.IPNet).IP
			fmt.Printf("Interface %s IP %s\n", iface.Name, addressIP.String())
			if !IsIPv4(addressIP) || rfc3927Net.Contains(addressIP) {
				continue
			}
			socket, err := net.DialUDP("udp", &net.UDPAddr{IP: addressIP, Port: ipv4MulticastPort}, &net.UDPAddr{IP: network.multicastIP, Port: ipv4MulticastPort})
			if err != nil {
				log.Printf("MulticastIPv4Send error dialing: %v\n", err)
				continue
			}
			sockets = append(sockets, socket)
		}

		return nil
	}

	if network.iface != nil {
		if err := interfaceMulticastSocketOpen(network.iface); err != nil {
			return
		}
	} else {
		interfaceList, err := net.Interfaces()
		if err != nil {
			return
		}
		for _, ifaceSingle := range interfaceList {
			interfaceMulticastSocketOpen(&ifaceSingle)
		}
	}

	for {
		raw, err := packetEncrypt(peerPrivateKey, ipv4MulticastPublicKey, &packetRaw{Protocol: 0, Command: 0})
		if err != nil {
			log.Printf("Error creating packet: %s\n", err.Error())
			time.Sleep(time.Second * 10)
			continue
		}

		// send it out to all sockets
		for _, socket := range sockets {
			_, err = socket.Write(raw)
			if err != nil {
				log.Printf("Error sending UDP packet: %v\n", err)
			}
		}

		// DEBUG
		//laddr, err := net.ResolveUDPAddr("udp", "10.0.0.98:0")
		//laddr, err := net.ResolveUDPAddr("udp", "0.0.0.0:0")
		//socket2, err := net.DialUDP("udp", laddr, &net.UDPAddr{IP: network.multicastIP, Port: ipv4MulticastPort})
		//if err != nil {
		//	log.Printf("Error sending UDP packet: %v\n", err)
		//}

		// send out the wire
		//_, err = network.multicastSocket.WriteTo(raw, &net.UDPAddr{IP: network.multicastIP, Port: ipv4MulticastPort})
		time.Sleep(time.Second * 10)
	}
}
*/
