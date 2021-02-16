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
	"strconv"
	"sync"

	"github.com/btcsuite/btcd/btcec"
)

// peerID is the current peers ID. It is a ECDSA (secp256k1) 257-bit public key.
var peerPrivateKey *btcec.PrivateKey
var peerPublicKey *btcec.PublicKey

func initPeerID() {
	peerList = make(map[btcec.PublicKey]*PeerInfo)

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

// PeerInfo stores information about a single remote peer
type PeerInfo struct {
	peerPublicKey *btcec.PublicKey // Public key
	network       *network         // Network interface used for communicating with the peer. If empty, try all.
	address       *net.UDPAddr     // Remote address of the peer (IP:Port)
}

var peerList map[btcec.PublicKey]*PeerInfo
var peerlistMutex sync.RWMutex

// PeerlistAdd adds a new peer to the peer list. It does not validate the peer info. If the peer is already added, it does nothing.
func PeerlistAdd(peer *PeerInfo) {
	peerlistMutex.Lock()
	defer peerlistMutex.Unlock()

	_, ok := peerList[*peer.peerPublicKey]
	if ok {
		return
	}

	peerList[*peer.peerPublicKey] = peer
}

// PeerlistRemove removes a peer from the peer list.
func PeerlistRemove(peer *PeerInfo) {
	peerlistMutex.Lock()
	defer peerlistMutex.Unlock()

	delete(peerList, *peer.peerPublicKey)
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

// send sends a raw packet to the peer
func (peer *PeerInfo) send(packet *packetRaw) (err error) {
	raw, err := packetEncrypt(peerPrivateKey, peer.peerPublicKey, packet)
	if err != nil {
		return err
	}

	// send out the wire
	if peer.network != nil {
		return peer.network.send(peer.address.IP, peer.address.Port, raw)
	}

	successCount := 0

	if IsIPv6(peer.address.IP.To16()) {
		for _, network := range networks6 {
			err = network.send(peer.address.IP, peer.address.Port, raw)
			if err == nil {
				successCount++
			}
		}
	} else {
		for _, network := range networks4 {
			err = network.send(peer.address.IP, peer.address.Port, raw)
			if err == nil {
				successCount++
			}
		}
	}

	if successCount == 0 {
		err = errors.New("Unsuccessful send")
	}

	return nil
}

// PeerFromSeed attempts to create a peer structure based on the input seed info.
// Address must be in "IP:Port" form. The Public Key must be hex encoded.
func PeerFromSeed(Address, PublicKey string) (peer *PeerInfo, err error) {
	host, portA, err := net.SplitHostPort(Address)
	if err != nil {
		return nil, err
	}

	portI, err := strconv.Atoi(portA)
	if err != nil {
		return nil, err
	} else if portI <= 0 || portI > 65535 {
		return nil, errors.New("Invalid port number")
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return nil, errors.New("Invalid input IP")
	}

	publicKeyB, err := hex.DecodeString(PublicKey)
	if err != nil {
		return nil, err
	}

	publicKeyP, err := btcec.ParsePubKey(publicKeyB, btcec.S256())
	if err != nil {
		return nil, err
	}

	peer = &PeerInfo{peerPublicKey: publicKeyP, address: &net.UDPAddr{IP: ip, Port: portI}}

	// Network is intentionally kept empty. It will try sending it from all available networks.

	return peer, nil
}
