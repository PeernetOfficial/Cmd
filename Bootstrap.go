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
		for _, address := range seed.Address {
			peer, err := PeerFromSeed(address, seed.PublicKey)
			if err != nil {
				log.Printf("loadSeedList Error address '%s' public key '%s': %v", address, seed.PublicKey, err.Error())
			} else {
				// TODO: Add flag to indicate root peer.
				PeerlistAdd(peer)
			}

			break // currently only supporting 1 address per peer
		}
	}
}
