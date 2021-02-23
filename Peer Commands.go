/*
File Name:  Peer Commands.go
Copyright:  2021 Peernet Foundation s.r.o.
Author:     Peter Kleissner
*/

package main

import (
	"fmt"
	"net"

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
		peer, added := PeerlistAdd(msg.SenderPublicKey, &connection{network: msg.network, address: msg.address})
		fmt.Printf("Incoming initial announcement from %s\n", msg.address.String())

		// send the Response
		if added {
			peer.send(&packetRaw{Command: 1})
		}

		return
	}
	fmt.Printf("Incoming secondary announcement from %s\n", msg.address.String())

	// Announcement from existing peer means the peer most likely restarted
	peer.send(&packetRaw{Command: 1})
}

// Response handles the response to the announcement
func (peer *PeerInfo) Response(msg *packet2) {
	if peer == nil {
		peer, _ = PeerlistAdd(msg.SenderPublicKey, &connection{network: msg.network, address: msg.address})
		fmt.Printf("Incoming initial response from %s\n", msg.address.String())

		return
	}

	fmt.Printf("Incoming response from %s on %s\n", msg.address.String(), msg.network.address.String())
}

// Chat handles a chat message [debug]
func (peer *PeerInfo) Chat(msg *packet2) {
	fmt.Printf("Chat from '%s': %s\n", msg.address.String(), string(msg.packetRaw.Payload))
}

// autoAnnounce sends out the announcement message every 10 seconds to its peer list
func autoAnnounce() {
	/*for {
		time.Sleep(time.Second * 10)

		for _, peer := range PeerlistGet() {
			peer.send(&packetRaw{Command: 0})
		}
	}*/
}

// autoPingAll automatically pings all peers. This has multiple important reasons:
// * Keeping the UDP port open at NAT, if applicable
// * Making sure to recognize invalid connections and drop them.
func autoPingAll() {

	// TODO: If peer was previously unresponsive for X times, start pinging on all available ports.
}

func sendChatAll(text string) {
	for _, peer := range PeerlistGet() {
		peer.send(&packetRaw{Command: 10, Payload: []byte(text)})
	}
}
