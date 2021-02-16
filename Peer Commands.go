/*
File Name:  Peer Commands.go
Copyright:  2021 Peernet Foundation s.r.o.
Author:     Peter Kleissner
*/

package main

import (
	"net"
	"time"

	"github.com/btcsuite/btcd/btcec"
)

// packet2 is a high-level message between peers
type packet2 struct {
	SenderPublicKey *btcec.PublicKey // Sender Public Key, ECDSA (secp256k1) 257-bit
	packetRaw
	network *network     // network which received the packet
	address *net.UDPAddr // address of the sender or receiver
}

// receiveAnnouncement handles an incoming announcement
func receiveAnnouncement(msg *packet2) {
	for _, peer := range PeerlistGet() {
		if peer.peerPublicKey.IsEqual(msg.SenderPublicKey) {
			return
		}
	}

	PeerlistAdd(&PeerInfo{peerPublicKey: msg.SenderPublicKey, network: msg.network, address: msg.address})
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
