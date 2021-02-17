/*
File Name:  Network Init.go
Copyright:  2021 Peernet Foundation s.r.o.
Author:     Peter Kleissner

Magic 🪄 to start the network configuration with 0 manual input. Users may specify the list of IPs (and optional ports) to listen; otherwise it listens on all.
IPv6 is always preferred.
*/

package main

import (
	"errors"
	"log"
	"math/rand"
	"net"
	"strconv"
	"time"
)

// networkWire is an incoming packet
type networkWire struct {
	network *network     // network which received the packet
	sender  *net.UDPAddr // sender of the packet
	raw     []byte       // buffer
}

var (
	rawPacketsIncoming chan networkWire
)

// initNetwork sets up the network configuration and starts listening.
func initNetwork() {
	rawPacketsIncoming = make(chan networkWire, 1000) // buffer up to 1000 UDP packets before they get buffered by the OS network stack and eventually dropped
	rand.Seed(time.Now().UnixNano())                  // we are not using "crypto/rand" for speed tradeoff

	if config.ListenWorkers == 0 {
		config.ListenWorkers = 2
	}
	for n := 0; n < config.ListenWorkers; n++ {
		go packetWorker(rawPacketsIncoming)
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
}

// networkPrepareListen prepares to listen on the given IP address. If port is 0, one is chosen automatically.
func networkPrepareListen(ipA string, port int) (err error) {
	ip := net.ParseIP(ipA)
	if ip == nil {
		return errors.New("Invalid input IP")
	}

	network := new(network)

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
	if IsIPv4(ip) {
		networks4 = append(networks4, network)
	} else {
		networks6 = append(networks6, network)

		network.MulticastJoin()
	}

	go network.Listen()

	log.Printf("Listen on UDP %s\n", network.listen.String())

	return nil
}
