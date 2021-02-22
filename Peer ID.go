/*
File Name:  Peer ID.go
Copyright:  2021 Peernet Foundation s.r.o.
Author:     Peter Kleissner
*/

package main

import (
	"encoding/hex"
	"errors"
	"log"
	"net"
	"os"
	"sync"
	"sync/atomic"

	"github.com/btcsuite/btcd/btcec"
)

// peerID is the current peers ID. It is a ECDSA (secp256k1) 257-bit public key.
var peerPrivateKey *btcec.PrivateKey
var peerPublicKey *btcec.PublicKey

func initPeerID() {
	peerList = make(map[[btcec.PubKeyBytesLenCompressed]byte]*PeerInfo)

	// load existing key from config, if available
	if len(config.PrivateKey) > 0 {
		configPK, err := hex.DecodeString(config.PrivateKey)
		if err == nil {
			peerPrivateKey, peerPublicKey = btcec.PrivKeyFromBytes(btcec.S256(), configPK)
			return
		}

		log.Printf("Private key in config is corrupted! Error: %s\n", err.Error())
		os.Exit(1)
	}

	// if the peer ID is empty, create a new user public-private key pair
	var err error
	peerPrivateKey, peerPublicKey, err = secp256k1NewPrivateKey()
	if err != nil {
		log.Printf("Error generating public-private key pairs: %s\n", err.Error())
		os.Exit(1)
	}

	// save the newly generated private key into the config
	config.PrivateKey = hex.EncodeToString(peerPublicKey.SerializeCompressed())

	saveConfig()
}

func secp256k1NewPrivateKey() (privateKey *btcec.PrivateKey, publicKey *btcec.PublicKey, err error) {
	key, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		return nil, nil, err
	}

	return key, (*btcec.PublicKey)(&key.PublicKey), nil
}

// connection is an established connection between a remote IP address and a local network adapter.
// New connections may only be created in case of successful INCOMING packets.
type connection struct {
	network *network     // network which received the packet
	address *net.UDPAddr // address of the sender or receiver
}

// PeerInfo stores information about a single remote peer
type PeerInfo struct {
	peerPublicKey *btcec.PublicKey // Public key
	connections   []connection     // list of established connections to the peer

	// statistics
	statsPacketSent     uint64 // Count of packets sent
	statsPacketReceived uint64 // Count of packets received
}

var peerList map[[btcec.PubKeyBytesLenCompressed]byte]*PeerInfo
var peerlistMutex sync.RWMutex

// PeerlistAdd adds a new peer to the peer list. It does not validate the peer info. If the peer is already added, it does nothing.
func PeerlistAdd(peer *PeerInfo) {
	peerlistMutex.Lock()
	defer peerlistMutex.Unlock()

	_, ok := peerList[publicKey2Compressed(peer.peerPublicKey)]
	if ok {
		return
	}

	peerList[publicKey2Compressed(peer.peerPublicKey)] = peer
}

// PeerlistRemove removes a peer from the peer list.
func PeerlistRemove(peer *PeerInfo) {
	peerlistMutex.Lock()
	defer peerlistMutex.Unlock()

	delete(peerList, publicKey2Compressed(peer.peerPublicKey))
}

// PeerlistGet returns the full peer list
func PeerlistGet() (peers []*PeerInfo) {
	peerlistMutex.RLock()
	defer peerlistMutex.RUnlock()

	for _, peer := range peerList {
		peers = append(peers, peer)
	}

	return peers
}

// PeerlistLookup returns the peer from the list with the public key
func PeerlistLookup(publicKey *btcec.PublicKey) (peer *PeerInfo) {
	peerlistMutex.RLock()
	defer peerlistMutex.RUnlock()

	peer, _ = peerList[publicKey2Compressed(publicKey)]
	return peer
}

func publicKey2Compressed(publicKey *btcec.PublicKey) [btcec.PubKeyBytesLenCompressed]byte {
	var key [btcec.PubKeyBytesLenCompressed]byte
	copy(key[:], publicKey.SerializeCompressed())
	return key
}

// send sends a raw packet to the peer
func (peer *PeerInfo) send(packet *packetRaw) (err error) {
	if len(peer.connections) == 0 {
		return errors.New("no valid connection to peer")
	}

	packet.Protocol = 0

	raw, err := packetEncrypt(peerPrivateKey, peer.peerPublicKey, packet)
	if err != nil {
		return err
	}

	atomic.AddUint64(&peer.statsPacketSent, 1)

	// send out the wire
	return peer.connections[0].network.send(peer.connections[0].address.IP, peer.connections[0].address.Port, raw)
}

// sendAllNetworks sends a raw packet via all networks
func sendAllNetworks(receiverPublicKey *btcec.PublicKey, packet *packetRaw, remote *net.UDPAddr) (err error) {
	packet.Protocol = 0
	raw, err := packetEncrypt(peerPrivateKey, receiverPublicKey, packet)
	if err != nil {
		return err
	}

	successCount := 0

	if IsIPv6(remote.IP.To16()) {
		for _, network := range networks6 {
			// Do not mix link-local unicast targets with non link-local networks (only when iface is known, i.e. not catch all local)
			if network.iface != nil && remote.IP.IsLinkLocalUnicast() != network.address.IP.IsLinkLocalUnicast() {
				continue
			}

			err = network.send(remote.IP, remote.Port, raw)
			if err == nil {
				successCount++
			}
		}
	} else {
		for _, network := range networks4 {
			// Do not mix link-local unicast targets with non link-local networks (only when iface is known, i.e. not catch all local)
			if network.iface != nil && remote.IP.IsLinkLocalUnicast() != network.address.IP.IsLinkLocalUnicast() {
				continue
			}

			err = network.send(remote.IP, remote.Port, raw)
			if err == nil {
				successCount++
			}
		}
	}

	if successCount == 0 {
		return errors.New("no successful send")
	}

	return nil
}
