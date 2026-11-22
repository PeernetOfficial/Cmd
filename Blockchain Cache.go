/*
File Name:  Blockchain Cache.go
Copyright:  2026 Kleissner Ventures Ltd
Author:     Peter Kleissner

Which blockchains of other peers are kept in the blockchain cache. A blockchain is kept while one of the reasons below is set for it.
Each reason is switched on or off in the config.
*/

package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/PeernetOfficial/core"
	"github.com/PeernetOfficial/core/webapi"
)

// Reasons for which a blockchain is kept in the blockchain cache.
const (
	cacheReasonRequested = 1 // The user requested the blockchain.
	cacheReasonNeighbour = 2 // This peer is among the closest peers it knows to the owner of the blockchain.
	cacheReasonMet       = 3 // The owner of the blockchain reported it in a message.
)

// cacheRequestedTime is the time a blockchain is kept after the user requested it.
const cacheRequestedTime = 30 * 24 * time.Hour

// cacheMetTime is the time a blockchain is kept after its owner last reported it.
const cacheMetTime = 24 * time.Hour

// cacheNeighbourCount is the count of closest peers to the owner of a blockchain that keep it as neighbours.
const cacheNeighbourCount = 20

// The blockchain of a neighbour is kept for 1 day after its owner last reported it. For each day that it has been kept as neighbour, it is kept a day longer, up to 7 days.
const cacheNeighbourDay = 24 * time.Hour
const cacheNeighbourDaysMax = 7

// filterBlockchainReported is called when a peer reports the version and height of its blockchain. It sets the reasons for which the blockchain is kept.
func filterBlockchainReported(peer *core.PeerInfo) {
	backend := peer.Backend
	now := time.Now()

	if config.CacheNeighbour && backend.IsAmongClosest(peer.PublicKey, cacheNeighbourCount) {
		days := 1
		if since, _, found := backend.GlobalBlockchainCache.Reason(peer.PublicKey, cacheReasonNeighbour); found {
			days += int(now.Sub(since) / cacheNeighbourDay)
		}
		if days > cacheNeighbourDaysMax {
			days = cacheNeighbourDaysMax
		}

		if err := backend.GlobalBlockchainCache.Keep(peer.PublicKey, cacheReasonNeighbour, now.Add(time.Duration(days)*cacheNeighbourDay)); err != nil {
			backend.LogError("filterBlockchainReported", "keeping blockchain %s as neighbour: %s\n", hex.EncodeToString(peer.PublicKey), err.Error())
		}
	}

	if config.CacheMet {
		if err := backend.GlobalBlockchainCache.Keep(peer.PublicKey, cacheReasonMet, now.Add(cacheMetTime)); err != nil {
			backend.LogError("filterBlockchainReported", "keeping blockchain %s as met: %s\n", hex.EncodeToString(peer.PublicKey), err.Error())
		}
	}
}

// debugCmdCacheKeep keeps the blockchain in the blockchain cache as requested by the user, and connects to its owner so that it is downloaded.
func debugCmdCacheKeep(backend *core.Backend, publicKey ed25519.PublicKey, output io.Writer) {
	if backend.GlobalBlockchainCache == nil {
		fmt.Fprintf(output, "The blockchain cache is disabled.\n")
		return
	} else if !config.CacheRequested {
		fmt.Fprintf(output, "Keeping requested blockchains is disabled in the config (CacheRequested).\n")
		return
	}

	until := time.Now().Add(cacheRequestedTime)

	if err := backend.GlobalBlockchainCache.Keep(publicKey, cacheReasonRequested, until); err != nil {
		fmt.Fprintf(output, "Error keeping the blockchain: %s\n", err.Error())
		return
	}

	fmt.Fprintf(output, "Blockchain %s is kept in the blockchain cache until %s UTC.\n", hex.EncodeToString(publicKey), until.UTC().Format(dateFormat))

	if _, err := webapi.PeerConnectPublicKey(backend, publicKey, time.Second*10); err != nil {
		fmt.Fprintf(output, "Could not connect to its owner: %s\nThe blockchain is downloaded when its owner is connected.\n", err.Error())
	}
}

// debugCmdCacheDrop ends keeping the blockchain in the blockchain cache as requested by the user.
func debugCmdCacheDrop(backend *core.Backend, publicKey ed25519.PublicKey, output io.Writer) {
	if backend.GlobalBlockchainCache == nil {
		fmt.Fprintf(output, "The blockchain cache is disabled.\n")
		return
	}

	if err := backend.GlobalBlockchainCache.Drop(publicKey, cacheReasonRequested); err != nil {
		fmt.Fprintf(output, "Error: %s\n", err.Error())
		return
	}

	fmt.Fprintf(output, "Blockchain %s is no longer kept in the blockchain cache as requested.\n", hex.EncodeToString(publicKey))
}
