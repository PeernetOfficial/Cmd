/*
File Name:  Bootstrap.go
Copyright:  2021 Peernet Foundation s.r.o.
Author:     Peter Kleissner

Strategy for sending our IPv6 Multicast and IPv4 Broadcast messages:
* During bootstrap: Immediately at the beginning, then every 10 seconds until there is at least 1 peer.
* Every 10 minutes during regular operation.
* Each time a network adapter / IP change is detected. [FUTURE]

*/

package main

import (
	"encoding/hex"
	"errors"
	"log"
	"net"
	"strconv"

	"github.com/btcsuite/btcd/btcec"
)

// rootPeer is a single root peer info
type rootPeer struct {
	peer      *PeerInfo        // loaded PeerInfo
	publicKey *btcec.PublicKey // Public key
	addresses []*net.UDPAddr   // IP:Port addresses
}

var rootPeers map[[btcec.PubKeyBytesLenCompressed]byte]*rootPeer

// initSeedList loads the seed list from the config
func initSeedList() {
loopSeedList:
	for _, seed := range config.SeedList {
		peer := rootPeer{}

		// parse the Public Key
		publicKeyB, err := hex.DecodeString(seed.PublicKey)
		if err != nil {
			log.Printf("initSeedList error public key '%s': %v", seed.PublicKey, err.Error())
			continue
		}

		if peer.publicKey, err = btcec.ParsePubKey(publicKeyB, btcec.S256()); err != nil {
			log.Printf("initSeedList error public key '%s': %v", seed.PublicKey, err.Error())
			continue
		}

		// parse all IP addresses
		for _, addressA := range seed.Address {
			address, err := parseAddress(addressA)
			if err != nil {
				log.Printf("initSeedList error public key '%s' address '%s': %v", seed.PublicKey, addressA, err.Error())
				continue loopSeedList
			}

			peer.addresses = append(peer.addresses, address)
		}
	}
}

// parseAddress parses an input peer address in the form "IP:Port".
func parseAddress(Address string) (remote *net.UDPAddr, err error) {
	host, portA, err := net.SplitHostPort(Address)
	if err != nil {
		return nil, err
	}

	portI, err := strconv.Atoi(portA)
	if err != nil {
		return nil, err
	} else if portI <= 0 || portI > 65535 {
		return nil, errors.New("invalid port number")
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return nil, errors.New("invalid input IP")
	}

	return &net.UDPAddr{IP: ip, Port: portI}, err
}

// bootstrap connects to the initial set of peers. It will also start the routine for ongoing sending of multicast/broadcast messages.
func bootstrap() {
	// initial contact to peer
	for _, peer := range rootPeers {
		for _, address := range peer.addresses {
			sendAllNetworks(peer.publicKey, &packetRaw{Command: 0}, address)
		}
	}

	// TODO: Continue trying to contact every X seconds until at least 1 peer is found

}

func autoMulticastBroadcast() {
	for _, network := range networks6 {
		if err := network.MulticastIPv6Send(); err != nil {
			log.Printf("bootstrap error multicast from network address '%s': %v", network.address.IP.String(), err.Error())
		}
	}

	for _, network := range networks4 {
		if err := network.BroadcastIPv4Send(); err != nil {
			log.Printf("bootstrap error broadcast from network address '%s': %v", network.address.IP.String(), err.Error())
		}
	}

	// TODO: Every X minutes
}
