/*
File Name:  Network IPv6 Multicast.go
Copyright:  2021 Peernet Foundation s.r.o.
Author:     Peter Kleissner

IPv6 Multicast implementation to support discovery of peers within the same network (Link-local	and Site-local).
*/

package main

import (
	"log"
	"net"
	"time"

	"golang.org/x/net/ipv6"
)

// The IPv6 is in general "ff05::112" which is site-local, i.e. the entire network.
// However for tests we allow listen6 to be ::1 and use loopback local multicast of "ff01::112".
const ipv6MulticastGroup = "112"

// MulticastJoin joins the Multicast group
func (network *network) MulticastJoin() (err error) {
	group := net.ParseIP("ff05::" + ipv6MulticastGroup)
	network.multicastIP = group

	// specific interface or join all?
	if network.iface != nil {
		if network.listen.IP.IsLoopback() {
			network.multicastIP = net.ParseIP("ff01::" + ipv6MulticastGroup)
		}

		// join the multicast group to receive all messages there
		pc := ipv6.NewPacketConn(network.socket)
		if err := pc.JoinGroup(network.iface, &net.UDPAddr{IP: network.multicastIP}); err != nil {
			return err
		}

		go network.MulticastSend()

		return nil
	}

	// join all
	interfaceList, err := net.Interfaces()
	if err != nil {
		return err
	}

	for _, ifaceSingle := range interfaceList {
		// join the multicast group to receive all messages there
		pc := ipv6.NewPacketConn(network.socket)
		if err := pc.JoinGroup(&ifaceSingle, &net.UDPAddr{IP: group}); err != nil {
			//log.Printf("MulticastJoin error joining '%s': %s\n", ifaceSingle.Name, err.Error())
			continue
		}
	}

	// ??
	/*if loop, err := pc.MulticastLoopback(); err == nil {
		if !loop {
			if err := pc.SetMulticastLoopback(true); err != nil {
				log.Printf("Error setting multicast loopback status: %v\n", err)
			}
		}
	}*/

	// start the client that sends multicast messages
	go network.MulticastSend()

	return nil
}

// MulticastSend sends out multicast messages to discover peers at the same site
func (network *network) MulticastSend() {
	time.Sleep(1 * time.Second)

	for {
		raw, err := packetEncrypt(peerPrivateKey, peerPublicKey, &packetRaw{Protocol: 0, Command: 0})
		if err != nil {
			log.Printf("Error creating packet: %s\n", err.Error())
			time.Sleep(time.Second * 10)
			continue
		}

		// send out the wire
		err = network.send(network.multicastIP, defaultPort, raw)
		if err != nil {
			log.Printf("Error sending UDP packet: %v\n", err)
		}
		err = network.send(network.multicastIP, startPortDefault2, raw)
		if err != nil {
			log.Printf("Error sending UDP packet: %v\n", err)
		}

		time.Sleep(time.Second * 10)
	}
}
