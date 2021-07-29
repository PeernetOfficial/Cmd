/*
File Name:  Command Debug.go
Copyright:  2021 Peernet Foundation s.r.o.
Author:     Peter Kleissner
*/

package main

import (
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/PeernetOfficial/core"
	"github.com/PeernetOfficial/core/dht"
)

// debugCmdConnect connects to the node ID
func debugCmdConnect(nodeID []byte) {
	fmt.Printf("---------------- Connect to node %s ----------------\n", hex.EncodeToString(nodeID))
	defer fmt.Printf("---------------- done node %s ----------------\n", hex.EncodeToString(nodeID))

	// in local DHT list?
	_, peer := core.IsNodeContact(nodeID)
	if peer != nil {
		fmt.Printf("* In local routing table: Yes.\n")
	} else {
		fmt.Printf("* In local routing table: No. Lookup via DHT. Timeout = 10 seconds.\n")

		addKeyMonitor(nodeID)
		defer removeKeyMonitor(nodeID)

		// Discovery via DHT.
		_, peer, _ = core.FindNode(nodeID, time.Second*10)
		if peer == nil {
			fmt.Printf("* Not found via DHT :(\n")
			return
		}

		fmt.Printf("* Successfully discovered via DHT.\n")
	}

	// virtual peer?
	if peer.IsVirtual() {
		fmt.Printf("* Peer is virtual and was not contacted before. It will show no active connections until contacted.\n")
	}

	fmt.Printf("* Peer details:\n")
	fmt.Printf("%s", textPeerConnections(peer))

	// ping via all connections TODO
	//fmt.Printf("* Sending ping:\n")
}

// ---- filter for outgoing DHT searches ----

// debug output of monitored keys searched in the DHT

var monitorKeys map[string]struct{}
var monitorKeysMutex sync.RWMutex
var enableMonitorAll = false // Enables output for all searches. Otherwise it only monitors searches stored in monitorKeys.

func addKeyMonitor(key []byte) {
	monitorKeysMutex.Lock()
	monitorKeys[string(key)] = struct{}{}
	monitorKeysMutex.Unlock()
}

func removeKeyMonitor(key []byte) {
	monitorKeysMutex.Lock()
	delete(monitorKeys, string(key))
	monitorKeysMutex.Unlock()
}

func filterSearchStatus(client *dht.SearchClient, function, format string, v ...interface{}) {
	// check if the search client is actively being monitored
	if !enableMonitorAll {
		monitorKeysMutex.Lock()
		_, ok := monitorKeys[string(client.Key)]
		monitorKeysMutex.Unlock()
		if !ok {
			return
		}
	}

	keyA := client.Key
	if len(keyA) > 8 {
		keyA = keyA[:8]
	}

	intend := " ->"

	switch function {
	case "search.sendInfoRequest":
		intend = "    >"
	case "dht.FindNode", "dht.Get", "dht.Store":
		intend = " -"
	case "search.startSearch":
		intend = "  >"
	}

	fmt.Printf(intend+" "+function+" ["+hex.EncodeToString(keyA)+"] "+format, v...)
}

// ---- filter for incoming information requests ----

var enableWatchIncomingAll = false

func filterIncomingRequest(peer *core.PeerInfo, Action int, Key []byte, Info interface{}) {
	if !enableWatchIncomingAll {
		return
	}

	requestType := "UNKNOWN"
	switch Action {
	case core.ActionFindSelf:
		requestType = "FIND_SELF"
	case core.ActionFindPeer:
		requestType = "FIND_PEER"
	case core.ActionFindValue:
		requestType = "FIND_VALUE"
	case core.ActionInfoStore:
		requestType = "INFO_STORE"
	}

	fmt.Printf("Incoming info request %s from %s for key %s\n", requestType, hex.EncodeToString(peer.NodeID), hex.EncodeToString(Key))
}
