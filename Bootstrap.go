/*
File Name:  Bootstrap.go
Copyright:  2021 Peernet Foundation s.r.o.
Author:     Peter Kleissner
*/

package main

import "log"

// loadSeedList loads the seed list from the config and tries to connect to the peers
func loadSeedList() {
	for _, seed := range config.SeedList {
		for _, addressA := range seed.Address {
			peer, remote, err := PeerFromSeed(addressA, seed.PublicKey)
			if err != nil {
				log.Printf("loadSeedList Error address '%s' public key '%s': %v", addressA, seed.PublicKey, err.Error())
			} else {
				// TODO: Add flag to indicate root peer.
				//PeerlistAdd(peer)

				// TODO: This has to be done a few times in a Go routine until the peer becomes alive.
				peer.sendAllNetworks(&packetRaw{Command: 0}, remote)
			}

			break // currently only supporting 1 address per peer
		}
	}
}
