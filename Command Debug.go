/*
File Name:  Command Debug.go
Copyright:  2021 Peernet Foundation s.r.o.
Author:     Peter Kleissner
*/

package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/PeernetOfficial/core"
	"github.com/PeernetOfficial/core/btcec"
	"github.com/PeernetOfficial/core/dht"
	"github.com/PeernetOfficial/core/protocol"
)

// debugCmdConnect connects to the node ID
func debugCmdConnect(backend *core.Backend, nodeID []byte, output io.Writer) {
	fmt.Fprintf(output, "---------------- Connect to node %s ----------------\n", hex.EncodeToString(nodeID))
	defer fmt.Fprintf(output, "---------------- done node %s ----------------\n", hex.EncodeToString(nodeID))

	// in local DHT list?
	_, peer := backend.IsNodeContact(nodeID)
	if peer != nil {
		fmt.Fprintf(output, "* In local routing table: Yes.\n")
	} else {
		fmt.Fprintf(output, "* In local routing table: No. Lookup via DHT. Timeout = 10 seconds.\n")

		hashMonitorControl(nodeID, 0)
		defer hashMonitorControl(nodeID, 1)

		// Discovery via DHT.
		_, peer, _ = backend.FindNode(nodeID, time.Second*10)
		if peer == nil {
			fmt.Fprintf(output, "* Not found via DHT :(\n")
			return
		}

		fmt.Fprintf(output, "* Successfully discovered via DHT.\n")
	}

	fmt.Fprintf(output, "* Peer details:\n")
	fmt.Fprintf(output, "  Uncontacted:      %t\n", peer.IsVirtual())
	fmt.Fprintf(output, "  Root peer:        %t\n", peer.IsRootPeer)
	fmt.Fprintf(output, "  User Agent:       %s\n", peer.UserAgent)
	fmt.Fprintf(output, "  Firewall:         %t\n", peer.IsFirewallReported())

	// virtual peer?
	if peer.IsVirtual() {
		fmt.Fprintf(output, "* Peer is virtual and was not contacted before. Sending out ping.\n")
		peer.Ping()
	} else {
		fmt.Fprintf(output, "* Connections:\n")
		fmt.Fprintf(output, "%s", textPeerConnections(peer))
	}

	// ping via all connections TODO
	//fmt.Fprintf(output, "* Sending ping:\n")
}

// ---- filter for outgoing DHT searches ----

// debug output of monitored keys searched in the DHT

var monitorKeys map[string]struct{}
var monitorKeysMutex sync.RWMutex
var enableMonitorAll = false // Enables output for all searches. Otherwise it only monitors searches stored in monitorKeys.

// hashMonitorControl adds (0), removes (1), or inverts (2) a hash on the list
func hashMonitorControl(key []byte, action int) (added bool) {
	monitorKeysMutex.Lock()
	defer monitorKeysMutex.Unlock()

	switch action {
	case 0:
		monitorKeys[string(key)] = struct{}{}
		added = true
	case 1:
		delete(monitorKeys, string(key))
	case 2:
		if _, ok := monitorKeys[string(key)]; !ok {
			monitorKeys[string(key)] = struct{}{}
			added = true
		} else {
			delete(monitorKeys, string(key))
		}
	}

	return
}

func hashIsMonitored(key []byte) (monitored bool) {
	monitorKeysMutex.Lock()
	_, monitored = monitorKeys[string(key)]
	monitorKeysMutex.Unlock()
	return
}

func filterSearchStatus(client *dht.SearchClient, function, format string, v ...interface{}) {
	if !enableMonitorAll && !hashIsMonitored(client.Key) {
		return
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
	if !enableWatchIncomingAll && !hashIsMonitored(peer.NodeID) {
		return
	}

	requestType := "UNKNOWN"
	switch Action {
	case protocol.ActionFindSelf:
		requestType = "FIND_SELF"
	case protocol.ActionFindPeer:
		requestType = "FIND_PEER"
	case protocol.ActionFindValue:
		requestType = "FIND_VALUE"
	case protocol.ActionInfoStore:
		requestType = "INFO_STORE"
	}

	if Action == protocol.ActionFindSelf && bytes.Equal(peer.NodeID, Key) {
		fmt.Printf("Info request from %s %s\n", hex.EncodeToString(peer.NodeID), requestType)
	} else {
		fmt.Printf("Info request from %s %s for key %s\n", hex.EncodeToString(peer.NodeID), requestType, hex.EncodeToString(Key))
	}
}

// ---- filter for incoming and outgoing packets ----

func filterMessageIn(peer *core.PeerInfo, raw *protocol.MessageRaw, message interface{}) {
	if !hashIsMonitored(peer.NodeID) {
		// TODO: For Announcement/Response also check data, Traverse the final target
		return
	}

	commandA := "Unknown"

	switch raw.Command {
	case protocol.CommandAnnouncement:
		commandA = "Announcement"
	case protocol.CommandResponse:
		commandA = "Response"
	case protocol.CommandPing:
		commandA = "Ping"
	case protocol.CommandPong:
		commandA = "Pong"
	case protocol.CommandLocalDiscovery:
		commandA = "Local Discovery"
	case protocol.CommandTraverse:
		commandA = "Traverse"
	case protocol.CommandChat:
		commandA = "Chat"
	}

	output := fmt.Sprintf("-------- Node %s Incoming %s --------\n", hex.EncodeToString(peer.NodeID), commandA)
	output += fmt.Sprintf("Sender Peer ID: %s\n", hex.EncodeToString(peer.PublicKey.SerializeCompressed()))

	if !raw.SenderPublicKey.IsEqual(peer.PublicKey) {
		output += fmt.Sprintf("WARNING: Mismatch of public keys, sender %s and packet indicates %s\n", hex.EncodeToString(peer.PublicKey.SerializeCompressed()), hex.EncodeToString(raw.SenderPublicKey.SerializeCompressed()))
	}

	if message == nil {
		output += "(no message decoded)\n"
	} else if announce, ok := message.(*protocol.MessageAnnouncement); ok {
		output += fmt.Sprintf("Fields:\n  Protocol supported    %d\n", announce.Protocol)
		output += fmt.Sprintf("  Feature bits          %d\n", announce.Features)
		output += fmt.Sprintf("  Action bits           %d\n", announce.Actions)
		output += fmt.Sprintf("  Blockchain Height     %d\n", announce.BlockchainHeight)
		output += fmt.Sprintf("  Blockchain Version    %d\n", announce.BlockchainVersion)
		output += fmt.Sprintf("  Port Internal         %d\n", announce.PortInternal)
		output += fmt.Sprintf("  Port External         %d\n", announce.PortExternal)
		output += fmt.Sprintf("  User Agent            %s\n", announce.UserAgent)

		if len(announce.FindPeerKeys) > 0 {
			output += fmt.Sprintf("FIND_PEER %d records:\n", len(announce.FindPeerKeys))
		}
		for _, find := range announce.FindPeerKeys {
			output += fmt.Sprintf("    - Find peer %s\n", hex.EncodeToString(find.Hash))
		}
		if len(announce.FindDataKeys) > 0 {
			output += fmt.Sprintf("FIND_VALUE %d records:\n", len(announce.FindDataKeys))
		}
		for _, find := range announce.FindDataKeys {
			output += fmt.Sprintf("    - Find data %s\n", hex.EncodeToString(find.Hash))
		}
		if len(announce.InfoStoreFiles) > 0 {
			output += fmt.Sprintf("INFO_STORE %d records:\n", len(announce.InfoStoreFiles))
		}
		for _, info := range announce.InfoStoreFiles {
			output += fmt.Sprintf("    - Info store %s, type %d, size %d\n", hex.EncodeToString(info.ID.Hash), info.Type, info.Size)
		}
	} else if response, ok := message.(*protocol.MessageResponse); ok {
		output += fmt.Sprintf("Fields:\n  Protocol supported    %d\n", response.Protocol)
		output += fmt.Sprintf("  Feature bits          %d\n", response.Features)
		output += fmt.Sprintf("  Action bits           %d\n", response.Actions)
		output += fmt.Sprintf("  Blockchain Height     %d\n", response.BlockchainHeight)
		output += fmt.Sprintf("  Blockchain Version    %d\n", response.BlockchainVersion)
		output += fmt.Sprintf("  Port Internal         %d\n", response.PortInternal)
		output += fmt.Sprintf("  Port External         %d\n", response.PortExternal)
		output += fmt.Sprintf("  User Agent            %s\n", response.UserAgent)

		for _, hash := range response.Hash2Peers {
			isLast := ""
			if hash.IsLast {
				isLast = " [last result in sequence]"
			}
			output += fmt.Sprintf("    - Peers known for the hash %s%s\n", hex.EncodeToString(hash.ID.Hash), isLast)
			for n := range hash.Closest {
				output += fmt.Sprintf("      Close peer:\n%s\n", outputPeerRecord(&hash.Closest[n]))
			}
			for n := range hash.Storing {
				output += fmt.Sprintf("      Peer stores:\n%s\n", outputPeerRecord(&hash.Storing[n]))
			}
		}
		for _, find := range response.FilesEmbed {
			output += fmt.Sprintf("    - File embedded %s (%d bytes)\n", hex.EncodeToString(find.ID.Hash), len(find.Data))
		}
		for _, hash := range response.HashesNotFound {
			output += fmt.Sprintf("    - Hash not found %s\n", hex.EncodeToString(hash))
		}
	} else if traverse, ok := message.(*protocol.MessageTraverse); ok {
		output += fmt.Sprintf("Fields:\n  Target Peer                     %s\n", hex.EncodeToString(traverse.TargetPeer.SerializeCompressed()))
		output += fmt.Sprintf("  Authorized Relay Peer           %s\n", hex.EncodeToString(traverse.AuthorizedRelayPeer.SerializeCompressed()))
		output += fmt.Sprintf("  Signer Public Key               %s\n", hex.EncodeToString(traverse.SignerPublicKey.SerializeCompressed()))
		output += fmt.Sprintf("  Expires                         %s\n", traverse.Expires.String())
		output += fmt.Sprintf("  IPv4                            %s\n", traverse.IPv4.String())
		output += fmt.Sprintf("  Port IPv4                       %d\n", traverse.PortIPv4)
		output += fmt.Sprintf("  Port IPv4 Reported External     %d\n", traverse.PortIPv4ReportedExternal)
		output += fmt.Sprintf("  IPv6                            %s\n", traverse.IPv6.String())
		output += fmt.Sprintf("  Port IPv6                       %d\n", traverse.PortIPv6)
		output += fmt.Sprintf("  Port IPv6 Reported External     %d\n", traverse.PortIPv6ReportedExternal)
	}

	output += "--------\n"

	fmt.Printf("%s", output)
}

func outputPeerRecord(record *protocol.PeerRecord) (output string) {
	output += fmt.Sprintf("      * Peer ID                         %s\n", hex.EncodeToString(record.PublicKey.SerializeCompressed()))
	output += fmt.Sprintf("        Node ID                         %s\n", hex.EncodeToString(record.NodeID))
	if record.IPv4 != nil && !record.IPv4.IsUnspecified() {
		output += fmt.Sprintf("        IPv4                            %s\n", record.IPv4.String())
		output += fmt.Sprintf("        Port IPv4                       %d\n", record.IPv4Port)
		output += fmt.Sprintf("        Port IPv4 Reported Internal     %d\n", record.IPv4PortReportedInternal)
		output += fmt.Sprintf("        Port IPv4 Reported External     %d\n", record.IPv4PortReportedExternal)
	}
	if record.IPv6 != nil && !record.IPv6.IsUnspecified() {
		output += fmt.Sprintf("        IPv6                            %s\n", record.IPv6.String())
		output += fmt.Sprintf("        Port IPv6                       %d\n", record.IPv6Port)
		output += fmt.Sprintf("        Port IPv6 Reported Internal     %d\n", record.IPv6PortReportedInternal)
		output += fmt.Sprintf("        Port IPv6 Reported External     %d\n", record.IPv6PortReportedExternal)
	}
	output += fmt.Sprintf("        Last Contact                    %s", record.LastContactT.Format(dateFormat))

	return
}

func outputOutgoingMessage(peer *core.PeerInfo, packet *protocol.PacketRaw) {
	if !hashIsMonitored(peer.NodeID) {
		// TODO: For Announcement/Response also check data, Traverse the final target
		return
	}

	commandA := "Unknown"

	switch packet.Command {
	case protocol.CommandAnnouncement:
		commandA = "Announcement"
	case protocol.CommandResponse:
		commandA = "Response"
	case protocol.CommandPing:
		commandA = "Ping"
	case protocol.CommandPong:
		commandA = "Pong"
	case protocol.CommandLocalDiscovery:
		commandA = "Local Discovery"
	case protocol.CommandTraverse:
		commandA = "Traverse"
	case protocol.CommandChat:
		commandA = "Chat"
	}

	output := fmt.Sprintf("-------- Node %s Outgoing %s --------\n", hex.EncodeToString(peer.NodeID), commandA)
	output += fmt.Sprintf("Receiver Peer ID: %s\n", hex.EncodeToString(peer.PublicKey.SerializeCompressed()))

	// TODO: Decoding of payload data (done by caller of this function)

	output += "--------\n"

	fmt.Printf("%s", output)
}

func filterMessageOutAnnouncement(receiverPublicKey *btcec.PublicKey, peer *core.PeerInfo, packet *protocol.PacketRaw, findSelf bool, findPeer []protocol.KeyHash, findValue []protocol.KeyHash, files []protocol.InfoStore) {
	if peer == nil {
		peer = &core.PeerInfo{PublicKey: receiverPublicKey, NodeID: protocol.PublicKey2NodeID(receiverPublicKey)}
	}

	outputOutgoingMessage(peer, packet)
}

func filterMessageOutResponse(peer *core.PeerInfo, packet *protocol.PacketRaw, hash2Peers []protocol.Hash2Peer, filesEmbed []protocol.EmbeddedFileData, hashesNotFound [][]byte) {
	outputOutgoingMessage(peer, packet)
}

func filterMessageOutTraverse(peer *core.PeerInfo, packet *protocol.PacketRaw, embeddedPacket *protocol.PacketRaw, receiverEnd *btcec.PublicKey) {
	outputOutgoingMessage(peer, packet)
}

func filterMessageOutPing(peer *core.PeerInfo, packet *protocol.PacketRaw, connection *core.Connection) {
	outputOutgoingMessage(peer, packet)
}

func filterMessageOutPong(peer *core.PeerInfo, packet *protocol.PacketRaw) {
	outputOutgoingMessage(peer, packet)
}
