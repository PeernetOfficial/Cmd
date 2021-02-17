/*
File Name:  Network.go
Copyright:  2021 Peernet Foundation s.r.o.
Author:     Peter Kleissner
*/

package main

import (
	"log"
	"net"
	"strings"
	"sync/atomic"
	"time"
)

// network is a connection adapter through one network interface (adapter).
// Note that for each IP on the same adapter separate network entries are created.
type network struct {
	iface           *net.Interface // Network interface belonging to the IP. May not be set.
	ipnet           *net.IPNet     // IP network the listening address belongs to. May not be set.
	address         *net.UDPAddr   // IP:Port where the server listens
	socket          *net.UDPConn   // active socket for send/receive
	multicastIP     net.IP         // Multicast IP, IPv6 only.
	multicastSocket net.PacketConn // Multicast socket, IPv6 only.
	broadcastSocket net.PacketConn // Broadcast socket, IPv4 only.
}

// networks is a list of all connected networks
var networks4, networks6 []*network

// Default ports to use. This may be randomized in the future to prevent fingerprinting (and subsequent blocking) by corporate and ISP firewalls.
const defaultPort = 'p'         // 112
const startPortDefault2 = 12912 // 03270h = 12912 = 'p2'
const maxPorts = 100

// AutoAssignPort assigns a port for the given IP. Use port 0 for zero configuration.
func (network *network) AutoAssignPort(ip net.IP, port int) (err error) {
	networkA := "udp6"
	if IsIPv4(ip) {
		networkA = "udp4"
	}

	// Find an available port. Sometimes the network interface might not be ready when the service is started immediately after boot. Full error: "listen udp6 [REDACTED]:[REDACTED]: bind: The requested address is not valid in its context."
	// To fix this, try up to 4 times, wait 20 seconds in between each try.
	connectPortTry := func(port int) (address *net.UDPAddr, socket *net.UDPConn, err error) {
		for retryCount := 0; retryCount < 4; retryCount++ {
			address = &net.UDPAddr{IP: ip, Port: port}
			socket, err = net.ListenUDP(networkA, address)
			if err == nil {
				return address, socket, nil
			}
			if !strings.Contains(err.Error(), "address is not valid in its context") {
				break
			}
			time.Sleep(time.Second * 20)
		}
		return nil, nil, err
	}

	if port != 0 {
		network.address, network.socket, err = connectPortTry(port)
		return err
	}

	// try default main port, then alternative ones
	network.address, network.socket, err = connectPortTry(defaultPort)
	if err == nil {
		return nil
	}

	for port := startPortDefault2; port < startPortDefault2+maxPorts; port++ {
		network.address, network.socket, err = connectPortTry(port)
		if err == nil {
			return nil
		}
	}

	return err
}

// send sends a message
func (network *network) send(IP net.IP, port int, raw []byte) (err error) {
	_, err = network.socket.WriteTo(raw, &net.UDPAddr{IP: IP, Port: port})
	return err
}

// Currently packets are maxed at 4 KB. This is going to be refined.
const maxPacketSize = 4096

// Listen starts listening for incoming packets on the given UDP connection
func (network *network) Listen() {
	for {
		// Buffer: Must be created for each packet as it is passed as pointer.
		// If the buffer is too small, ReadFromUDP only reads until its length and returns this error: "wsarecvfrom: A message sent on a datagram socket was larger than the internal message buffer or some other network limit, or the buffer used to receive a datagram into was smaller than the datagram itself."
		buffer := make([]byte, maxPacketSize)
		length, sender, err := network.socket.ReadFromUDP(buffer)

		if err != nil {
			log.Printf("Listen Error receiving UDP message: %v\n", err) // Only log for debug purposes.
			time.Sleep(time.Millisecond * 50)                           // In case of endless errors, prevent ddos of CPU.
			continue
		}

		if length < packetLengthMin {
			// Discard packets that do not meet the minimum length.
			continue
		}

		// send the packet to a channel which is processed by multiple workers.
		rawPacketsIncoming <- networkWire{network: network, sender: sender, raw: buffer[:length], receiverPublicKey: peerPublicKey}
	}
}

// packetWorker handles incoming packets.
func packetWorker(packets <-chan networkWire) {
	for packet := range packets {
		decoded, senderPublicKey, err := packetDecrypt(packet.raw, packet.receiverPublicKey)
		if err != nil {
			log.Printf("packetWorker Error decrypting packet from '%s': %s\n", packet.sender.String(), err.Error())
			continue
		}

		// immediately discard message if sender = self
		if senderPublicKey.IsEqual(peerPublicKey) {
			continue
		}

		// supported protocol version
		if decoded.Protocol != 0 {
			continue
		}

		peer := PeerlistLookup(senderPublicKey)
		if peer != nil {
			atomic.AddUint64(&peer.statsPacketReceived, 1)
			peer.network = packet.network
			if peer.address == nil {
				peer.address = packet.sender
			} else if !peer.address.IP.Equal(packet.sender.IP) || peer.address.Port != packet.sender.Port {
				//log.Printf("DEBUG Mismatch previous connection %s new connection %s\n", peer.address.String(), packet.sender.String())
			}
		}

		// process the packet
		message := &packet2{SenderPublicKey: senderPublicKey, packetRaw: *decoded, network: packet.network, address: packet.sender}

		switch decoded.Command {
		case 0: // Announce
			peer.Announcement(message)

		case 10: // Chat [debug]
			peer.Chat(message)

		default: // Unknown command

		}

	}
}
