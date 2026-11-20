/*
File Name:  Command Debug.go
Copyright:  2021 Kleissner Ventures Ltd
Author:     Peter Kleissner
*/

package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/PeernetOfficial/core"
	"github.com/PeernetOfficial/core/dht"
	"github.com/PeernetOfficial/core/protocol"
)

// debugCmdConnect connects to the peer
func debugCmdConnect(backend *core.Backend, publicKey ed25519.PublicKey, output io.Writer) {
	fmt.Fprintf(output, "---------------- Connect to peer %s ----------------\n", hex.EncodeToString(publicKey))
	defer fmt.Fprintf(output, "---------------- done peer %s ----------------\n", hex.EncodeToString(publicKey))

	// in local DHT list?
	_, peer := backend.IsNodeContact(publicKey)
	if peer != nil {
		fmt.Fprintf(output, "* In local routing table: Yes.\n")
	} else {
		fmt.Fprintf(output, "* In local routing table: No. Lookup via DHT. Timeout = 10 seconds.\n")

		hashMonitorControl(publicKey, 0, output)
		defer hashMonitorControl(publicKey, 1, nil)

		// Discovery via DHT.
		_, peer, _ = backend.FindNode(publicKey, time.Second*10)
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

// debugCmdFindBlockchain looks up the peers that store the blockchain and lists each one as a peer reports it
func debugCmdFindBlockchain(backend *core.Backend, publicKey ed25519.PublicKey, output io.Writer) {
	fmt.Fprintf(output, "---------------- Find blockchain %s ----------------\n", hex.EncodeToString(publicKey))
	defer fmt.Fprintf(output, "---------------- done blockchain %s ----------------\n", hex.EncodeToString(publicKey))

	hashMonitorControl(publicKey, 0, output)
	defer hashMonitorControl(publicKey, 1, nil)

	timeStart := time.Now()

	search, err := backend.FindBlockchain(publicKey)
	if err != nil {
		fmt.Fprintf(output, "* Error starting the lookup: %s\n", err.Error())
		return
	}

	count := 0

	for {
		select {
		case result := <-search.Storing:
			storing := result.StoringNode.Info.(*core.BlockchainStoring)
			count++

			fmt.Fprintf(output, "* Peer %s stores the blockchain: version %d, height %d, time %s UTC. Reported by %s.\n", hex.EncodeToString(storing.Peer.PublicKey), storing.Version, storing.Height, storing.Time.UTC().Format(dateFormat), hex.EncodeToString(result.SenderID))

		case <-search.TerminateSignal:
			fmt.Fprintf(output, "* Lookup ended after %s. Storing peers reported: %d\n", time.Since(timeStart).Round(time.Millisecond).String(), count)
			return
		}
	}
}

// ---- filter for outgoing DHT searches ----

// debug output of monitored keys searched in the DHT

var monitorKeys map[string]io.Writer = make(map[string]io.Writer)
var monitorKeysMutex sync.RWMutex

// hashMonitorControl adds (0), removes (1), or inverts (2) a hash on the list
func hashMonitorControl(key []byte, action int, output io.Writer) (added bool) {
	monitorKeysMutex.Lock()
	defer monitorKeysMutex.Unlock()

	switch action {
	case 0:
		monitorKeys[string(key)] = output
		added = true
	case 1:
		delete(monitorKeys, string(key))
	case 2:
		if _, ok := monitorKeys[string(key)]; !ok {
			monitorKeys[string(key)] = output
			added = true
		} else {
			delete(monitorKeys, string(key))
		}
	}

	return
}

func hashIsMonitored(keys ...[]byte) (monitored bool, output io.Writer) {
	monitorKeysMutex.Lock()
	defer monitorKeysMutex.Unlock()

	for _, key := range keys {
		if output, monitored = monitorKeys[string(key)]; monitored {
			return true, output
		}
	}

	return false, nil
}

const keyMonitorAllSearches = "all searches" // special key to monitor all searches

func filterSearchStatus(client *dht.SearchClient, function, format string, v ...interface{}) {
	monitored, output := hashIsMonitored(client.Key, []byte(keyMonitorAllSearches))
	if !monitored {
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
	case "dht.FindNode":
		intend = " -"
	case "search.startSearch":
		intend = "  >"
	}

	fmt.Fprintf(output, intend+" "+function+" ["+hex.EncodeToString(keyA)+"] "+format, v...)
}

// ---- filter for incoming information requests ----

const keyMonitorAllRequests = "all requests" // special key to monitor all info requests

// requestTypeNames are the request record types as named in the whitepaper.
var requestTypeNames = map[int]string{
	protocol.RequestAFindSelf:        "A_FIND_SELF",
	protocol.RequestAFindPeer:        "A_FIND_PEER",
	protocol.RequestBFindBlockchain:  "B_FIND_BLOCKCHAIN",
	protocol.RequestBStoreBlockchain: "B_STORE_BLOCKCHAIN",
}

// responseTypeNames are the response record types as named in the whitepaper.
var responseTypeNames = map[uint8]string{
	protocol.ResponseASelf:       "A_RESPONSE_SELF",
	protocol.ResponseAPeer:       "A_RESPONSE_PEER",
	protocol.ResponseBBlockchain: "B_RESPONSE_BLOCKCHAIN",
}

func filterIncomingRequest(peer *core.PeerInfo, RequestType int, Key []byte, Info interface{}) {
	monitored, output := hashIsMonitored(peer.PublicKey, []byte(keyMonitorAllRequests))
	if !monitored {
		return
	}

	requestType, ok := requestTypeNames[RequestType]
	if !ok {
		requestType = "UNKNOWN"
	}

	if RequestType == protocol.RequestAFindSelf {
		fmt.Fprintf(output, "Request from %s %s\n", hex.EncodeToString(peer.PublicKey), requestType)
	} else {
		fmt.Fprintf(output, "Request from %s %s for key %s\n", hex.EncodeToString(peer.PublicKey), requestType, hex.EncodeToString(Key))
	}
}

// ---- filter for incoming and outgoing packets ----

func filterMessageIn(peer *core.PeerInfo, raw *protocol.MessageRaw, message interface{}) {
	monitored, output := hashIsMonitored(peer.PublicKey)
	if !monitored {
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
	case protocol.CommandRelay:
		commandA = "Relay"
	case protocol.CommandChat:
		commandA = "Chat"
	}

	text := fmt.Sprintf("-------- Peer %s Incoming %s --------\n", hex.EncodeToString(peer.PublicKey), commandA)
	text += fmt.Sprintf("Sender Peer ID: %s\n", hex.EncodeToString(peer.PublicKey))

	if !raw.SenderPublicKey.Equal(peer.PublicKey) {
		text += fmt.Sprintf("WARNING: Mismatch of public keys, sender %s and packet indicates %s\n", hex.EncodeToString(peer.PublicKey), hex.EncodeToString(raw.SenderPublicKey))
	}

	if message == nil {
		text += "(no message decoded)\n"
	} else if announce, ok := message.(*protocol.MessageAnnouncement); ok {
		text += fmt.Sprintf("Fields:\n  Protocol supported    %d\n", announce.Protocol)
		text += fmt.Sprintf("  Feature bits          %d\n", announce.Features)
		text += fmt.Sprintf("  Blockchain Version    %d\n", announce.BlockchainVersion)
		text += fmt.Sprintf("  Blockchain Height     %d\n", announce.BlockchainHeight)
		text += fmt.Sprintf("  Port Internal         %d\n", announce.PortInternal)
		text += fmt.Sprintf("  Port External         %d\n", announce.PortExternal)
		text += fmt.Sprintf("  User Agent            %s\n", announce.UserAgent)

		text += outputRequests(&announce.Requests)
	} else if response, ok := message.(*protocol.MessageResponse); ok {
		text += fmt.Sprintf("Fields:\n  Protocol supported    %d\n", response.Protocol)
		text += fmt.Sprintf("  Feature bits          %d\n", response.Features)
		text += fmt.Sprintf("  Sequence last         %t\n", response.SequenceLast)
		text += fmt.Sprintf("  Blockchain Version    %d\n", response.BlockchainVersion)
		text += fmt.Sprintf("  Blockchain Height     %d\n", response.BlockchainHeight)
		text += fmt.Sprintf("  Port Internal         %d\n", response.PortInternal)
		text += fmt.Sprintf("  Port External         %d\n", response.PortExternal)
		text += fmt.Sprintf("  User Agent            %s\n", response.UserAgent)

		text += outputResponseRecords(response.Records)
	} else if traverse, ok := message.(*protocol.MessageTraverse); ok {
		text += fmt.Sprintf("Fields:\n  Target Peer                     %s\n", hex.EncodeToString(traverse.TargetPeer))

		// The embedded packet is encrypted for the target peer. The relay reads only its sender.
		// The target peer decrypts and verifies it, with the Peer ID of the relay that sent this message as additional authenticated data.
		if !traverse.TargetPeer.Equal(peer.Backend.SelfPublicKey()) {
			if embeddedSender, err := protocol.PacketEmbeddedSender(traverse.EmbeddedPacketRaw, traverse.TargetPeer); err == nil {
				text += fmt.Sprintf("  Original Sender                 %s (as read by the relay)\n", hex.EncodeToString(embeddedSender))
			} else {
				text += fmt.Sprintf("  Original Sender                 (embedded packet invalid: %s)\n", err.Error())
			}
		} else {
			privateKey, _ := peer.Backend.ExportPrivateKey()
			var embeddedSender ed25519.PublicKey
			keyPair, err := protocol.NewKeyPair(privateKey)
			if err == nil {
				_, embeddedSender, err = protocol.PacketDecryptEmbedded(traverse.EmbeddedPacketRaw, keyPair, peer.PublicKey)
			}
			if err == nil {
				text += fmt.Sprintf("  Original Sender                 %s (embedded packet valid)\n", hex.EncodeToString(embeddedSender))
			} else {
				text += fmt.Sprintf("  Original Sender                 (embedded packet invalid: %s)\n", err.Error())
			}
		}
		text += fmt.Sprintf("  IPv4                            %s\n", traverse.IPv4.String())
		text += fmt.Sprintf("  Port IPv4                       %d\n", traverse.PortIPv4)
		text += fmt.Sprintf("  Port IPv4 Reported External     %d\n", traverse.PortIPv4ReportedExternal)
		text += fmt.Sprintf("  IPv6                            %s\n", traverse.IPv6.String())
		text += fmt.Sprintf("  Port IPv6                       %d\n", traverse.PortIPv6)
		text += fmt.Sprintf("  Port IPv6 Reported External     %d\n", traverse.PortIPv6ReportedExternal)
	} else if relay, ok := message.(*protocol.MessageRelay); ok {
		text += fmt.Sprintf("Fields:\n  Peer                            %s\n", hex.EncodeToString(relay.Peer))
		text += fmt.Sprintf("  FORWARDED                       %t\n", relay.Forwarded)

		// With the FORWARDED flag this peer is the end receiver: the embedded packet is a packet from the named peer to it, with the Peer ID of the relay that sent this message
		// as additional authenticated data. Without the flag this peer is the relay, which cannot read the embedded packet.
		if relay.Forwarded {
			privateKey, _ := peer.Backend.ExportPrivateKey()
			var embedded *protocol.PacketRaw
			keyPair, err := protocol.NewKeyPair(privateKey)
			if err == nil {
				embedded, err = protocol.PacketDecryptRelay(relay.EmbeddedPacketRaw, keyPair, relay.Peer, peer.PublicKey, nil)
			}
			if err == nil {
				text += fmt.Sprintf("  Embedded Packet                 command %d sequence %d from the named peer (valid)\n", embedded.Command, embedded.Sequence)
			} else {
				text += fmt.Sprintf("  Embedded Packet                 (invalid: %s)\n", err.Error())
			}
		} else {
			text += fmt.Sprintf("  Embedded Packet                 %d bytes for the named peer\n", len(relay.EmbeddedPacketRaw))
		}
	}

	text += "--------\n"

	output.Write([]byte(text))
}

// outputRequests formats the request records of an announcement
func outputRequests(requests *protocol.Requests) (text string) {
	if requests.FindSelf {
		text += "    - A_FIND_SELF\n"
	}
	for _, key := range requests.FindPeer {
		text += fmt.Sprintf("    - A_FIND_PEER %s\n", hex.EncodeToString(key))
	}
	for _, key := range requests.FindBlockchain {
		text += fmt.Sprintf("    - B_FIND_BLOCKCHAIN %s\n", hex.EncodeToString(key))
	}
	for _, store := range requests.StoreBlockchain {
		text += fmt.Sprintf("    - B_STORE_BLOCKCHAIN %s, version %d, height %d, time %s UTC\n", hex.EncodeToString(store.PublicKey), store.Version, store.Height, time.Unix(int64(store.Time), 0).UTC().Format(dateFormat))
	}
	return text
}

// outputResponseRecords formats the records of a response
func outputResponseRecords(records []protocol.ResponseRecord) (text string) {
	for _, record := range records {
		name, ok := responseTypeNames[record.Type]
		if !ok {
			name = fmt.Sprintf("UNKNOWN (%d)", record.Type)
		}

		switch record.Type {
		case protocol.ResponseASelf:
			text += fmt.Sprintf("    - %s", name)
		default:
			text += fmt.Sprintf("    - %s %s", name, hex.EncodeToString(record.Key))
		}
		if record.IsLast {
			text += " [last result for the key]"
		}
		text += "\n"

		for n := range record.Storing {
			text += fmt.Sprintf("      Peer stores:\n%s\n", outputPeerRecord(&record.Storing[n].PeerRecord))
			if record.Type == protocol.ResponseBBlockchain {
				text += fmt.Sprintf("        Blockchain version %d, height %d, time %s UTC\n", record.Storing[n].BlockchainVersion, record.Storing[n].BlockchainHeight, time.Unix(int64(record.Storing[n].Time), 0).UTC().Format(dateFormat))
			}
		}
		for n := range record.Closest {
			text += fmt.Sprintf("      Close peer:\n%s\n", outputPeerRecord(&record.Closest[n]))
		}
	}
	return text
}

func outputPeerRecord(record *protocol.PeerRecord) (output string) {
	output += fmt.Sprintf("      * Peer ID                         %s", hex.EncodeToString(record.PublicKey))
	for _, address := range record.IPv4 {
		output += fmt.Sprintf("\n        IPv4                            %s", address.IP.String())
		output += fmt.Sprintf("\n        Port IPv4                       %d", address.Port)
		output += fmt.Sprintf("\n        Port IPv4 Reported Internal     %d", address.PortReportedInternal)
		output += fmt.Sprintf("\n        Port IPv4 Reported External     %d", address.PortReportedExternal)
	}
	for _, address := range record.IPv6 {
		output += fmt.Sprintf("\n        IPv6                            %s", address.IP.String())
		output += fmt.Sprintf("\n        Port IPv6                       %d", address.Port)
		output += fmt.Sprintf("\n        Port IPv6 Reported Internal     %d", address.PortReportedInternal)
		output += fmt.Sprintf("\n        Port IPv6 Reported External     %d", address.PortReportedExternal)
	}

	return
}

func outputOutgoingMessage(peer *core.PeerInfo, packet *protocol.PacketRaw) {
	monitored, output := hashIsMonitored(peer.PublicKey)
	if !monitored {
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
	case protocol.CommandRelay:
		commandA = "Relay"
	case protocol.CommandChat:
		commandA = "Chat"
	}

	text := fmt.Sprintf("-------- Peer %s Outgoing %s --------\n", hex.EncodeToString(peer.PublicKey), commandA)
	text += fmt.Sprintf("Receiver Peer ID: %s\n", hex.EncodeToString(peer.PublicKey))

	// TODO: Decoding of payload data (done by caller of this function)

	text += "--------\n"

	output.Write([]byte(text))
}

func filterMessageOutAnnouncement(receiverPublicKey ed25519.PublicKey, peer *core.PeerInfo, packet *protocol.PacketRaw, requests *protocol.Requests) {
	if peer == nil {
		peer = &core.PeerInfo{PublicKey: receiverPublicKey}
	}

	outputOutgoingMessage(peer, packet)
}

func filterMessageOutResponse(peer *core.PeerInfo, packet *protocol.PacketRaw, records []protocol.ResponseRecord) {
	outputOutgoingMessage(peer, packet)
}

func filterMessageOutTraverse(peer *core.PeerInfo, packet *protocol.PacketRaw, embeddedPacket *protocol.PacketRaw, receiverEnd ed25519.PublicKey) {
	outputOutgoingMessage(peer, packet)
}

func filterMessageOutPing(peer *core.PeerInfo, packet *protocol.PacketRaw, connection *core.Connection) {
	outputOutgoingMessage(peer, packet)
}

func filterMessageOutPong(peer *core.PeerInfo, packet *protocol.PacketRaw) {
	outputOutgoingMessage(peer, packet)
}
