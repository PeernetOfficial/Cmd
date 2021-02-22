/*
File Name:  Peer Commands.go
Copyright:  2021 Peernet Foundation s.r.o.
Author:     Peter Kleissner
*/

package main

import (
	"fmt"
	"net"
	"time"

	"github.com/btcsuite/btcd/btcec"
)

// packet2 is a high-level message between peers
type packet2 struct {
	packetRaw
	SenderPublicKey *btcec.PublicKey // Sender Public Key, ECDSA (secp256k1) 257-bit
	network         *network         // network which received the packet
	address         *net.UDPAddr     // address of the sender or receiver
}

// Announcement handles an incoming announcement
func (peer *PeerInfo) Announcement(msg *packet2) {
	if peer == nil {
		PeerlistAdd(&PeerInfo{peerPublicKey: msg.SenderPublicKey, connections: []connection{connection{network: msg.network, address: msg.address}}})
		fmt.Printf("Incoming initial announcement from %s\n", msg.address.String())
	}
}

// Chat handles a chat message [debug]
func (peer *PeerInfo) Chat(msg *packet2) {
	fmt.Printf("Chat from '%s': %s\n", msg.address.String(), string(msg.packetRaw.Payload))
}

// autoAnnounce sends out the announcement message every 10 seconds to its peer list
func autoAnnounce() {
	for {
		time.Sleep(time.Second * 10)

		for _, peer := range PeerlistGet() {
			peer.send(&packetRaw{Command: 0})
		}
	}
}

func sendChatAll(text string) {
	for _, peer := range PeerlistGet() {
		peer.send(&packetRaw{Command: 10, Payload: []byte(text)})
	}
}
