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
			fmt.Printf("Not found via DHT :(\n")
			return
		}

		fmt.Printf("Successfully discovered via DHT.\n")
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

// debug output of monitored keys searched in the DHT

var monitorKeys map[string]struct{}
var monitorKeysMutex sync.RWMutex

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
	monitorKeysMutex.Lock()
	_, ok := monitorKeys[string(client.Key)]
	monitorKeysMutex.Unlock()
	if !ok {
		return
	}

	keyA := client.Key
	if len(keyA) > 8 {
		keyA = keyA[:8]
	}

	intend := " ->"
	if function == "sendInfoRequest" {
		intend = "    >"
	}

	fmt.Printf(intend+" ["+function+" "+hex.EncodeToString(keyA)+"] "+format, v...)
}
