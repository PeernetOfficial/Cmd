/*
File Name:  Network Init.go
Copyright:  2021 Peernet Foundation s.r.o.
Author:     Peter Kleissner

Magic 🪄 to start the network configuration with 0 manual input. Users may specify the list of IPs (and optional ports) to listen; otherwise it listens on all.
IPv6 is always preferred.
*/

package main

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"net"
	"strconv"
)

type networkWire struct {
	network *network6    // network which received the packet
	sender  *net.UDPAddr // sender of the packet
	raw     []byte       // buffer
}

var (
	rawPacketsIncoming chan networkWire
	nonce1             uint64 // Input for nonce generation. Should be set to random value at at start and increased with every sent packet.
)

// initNetwork sets up the network configuration and starts listening.
func initNetwork() {
	rawPacketsIncoming = make(chan networkWire, 1000) // buffer up to 1000 UDP packets before they get buffered by the OS network stack and eventually dropped
	_, nonce1, _ = GenerateNonce1()

	if config.ListenWorkers == 0 {
		config.ListenWorkers = 2
	}
	for n := 0; n < config.ListenWorkers; n++ {
		go handlePackets(rawPacketsIncoming)
	}

	// check if user specified where to listen
	if len(config.Listen) > 0 {
		for _, listenA := range config.Listen {
			host, portA, err := net.SplitHostPort(listenA)
			if err != nil && err.Error() == "missing port in address" { // port is optional
				host = listenA
				portA = "0"
			} else if err != nil {
				log.Printf("initNetwork Error invalid input listen address '%s': %s\n", listenA, err.Error())
				continue
			}

			portI, _ := strconv.Atoi(portA)

			if err := networkPrepareListen(host, portI); err != nil {
				log.Printf("initNetwork Error listen on '%s': %s\n", listenA, err.Error())
				continue
			}
		}

		return
	}

	// Listen on all IPv4 and IPv6 addresses
	if err := networkPrepareListen("0.0.0.0", 0); err != nil {
		log.Printf("initNetwork Error listen on all IPv4 addresses (0.0.0.0): %s\n", err.Error())
	}
	if err := networkPrepareListen("::", 0); err != nil {
		log.Printf("initNetwork Error listen on all IPv6 addresses (::): %s\n", err.Error())
	}

	// Note: While it would be easy to simply listen on 0.0.0.0 and ::, it makes more sense to enumerate all IPs and listen on each specifically.
	// This has several advantages down the road:
	// 1. We can use IPv6 Multicast for local peer discovery. This is way more reliable than IPv4 MDNS.
	// 2. Support for complex multi-adapter configurations. Since we will bind outgoing traffic to the same adapter as inbound one, this allows proper routing
	//    where the OS would fail to auto-pick. This is a real scenario where competing adapter Metrics fuck up the default route.
	// 3. Overall better granularity and control.

	// TODO: can it all listen on one? https://pkg.go.dev/golang.org/x/net/ipv6

}

// networkPrepareListen prepares to listen on the given IP address. If port is 0, one is chosen automatically.
func networkPrepareListen(ipA string, port int) (err error) {
	ip := net.ParseIP(ipA)
	if ip == nil {
		return errors.New("Invalid input IP")
	} else if IsIPv4(ip) {
		return errors.New("IPv4 is not supported")
	}

	network := new(network6)

	// get the network interface that belongs to the IP
	if ipA != "0.0.0.0" && ipA != "::" {
		network.iface, network.ipnet = FindInterfaceByIP(ip)
		if network.iface == nil {
			return errors.New("Error finding the network interface belonging to IP")
		}
	}

	// open up the port
	if err = network.AutoAssignPort(ip, port); err != nil {
		return err
	}

	// Success - port is open. Add to the list and start accepting incoming messages.
	networks6 = append(networks6, network)

	// TODO
	//initMessage(newNetwork.apiSocketPort, newNetwork.apiSocket)

	return nil
}

// GenerateNonce1 generates the initial input for the nonce
func GenerateNonce1() (nonce1 [8]byte, nonce1u uint64, err error) {
	_, err = io.ReadFull(rand.Reader, nonce1[:])
	return nonce1, binary.LittleEndian.Uint64(nonce1[:]), err
}
