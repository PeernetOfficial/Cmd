/*
File Name:  Packet Encoding.go
Copyright:  2021 Peernet Foundation s.r.o.
Author:     Peter Kleissner

Basic packet structure of ALL packets:
Offset  Size   Info
0       4      Nonce
4       1      Protocol version = 0
5       1      Command
6       2      Size of payload data
8       ?      Payload
        ?      Randomized garbage
?		65     Signature, ECDSA secp256k1 512-bit + 1 header byte

The peer ID of the sender, which is a ECDSA (secp256k1) 257-bit public key, can be extracted from the ECDSA signature.
The signature is applied on the entire packet, which guarantees that the signature becomes invalid should someone try to forge the receiver (i.e. forward the packet).
Because the signature could be a possible fingerpint, it is encrypted itself.
*/

package main

import (
	"encoding/binary"
	"errors"
	"sync/atomic"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"golang.org/x/crypto/salsa20"
)

// packetRaw is a decrypted P2P message
type packetRaw struct {
	SenderPublicKey *btcec.PublicKey // Sender Public Key, ECDSA (secp256k1) 257-bit
	Protocol        uint8            // Protocol version = 0
	Command         uint8            // 0 = Announcement
	Payload         []byte           // Payload
}

// The minimum packet size is 8 bytes (minimum header size) + 65 bytes (signature)
const packetLengthMin = 8 + signatureSize
const signatureSize = 65

// packetDecrypt decrypts the packet, verifies its signature and returns a high-level version of the packet.
// receiverID is the compressed version of the receiver's public key.
func packetDecrypt(raw []byte, receiverID *[32]byte) (packet *packetRaw, err error) {
	// Packet is assumed to be already checked for minimum length.

	// Prepare Salsa20 nonce and key. Nonce = 2x first 4 bytes. For size reasons, only 4 bytes (instead of 8 bytes) is supplied in the packet.
	// This could be a risk, but considering we only use the PUBLIC key as decryption key, it is negligible.
	nonce := make([]byte, 8)
	copy(nonce[0:4], raw[0:4])
	copy(nonce[4:8], raw[0:4])

	// Verify the signature and extract the public key from it.
	var signature [signatureSize]byte
	copy(signature[:], raw[len(raw)-signatureSize:])
	salsa20.XORKeyStream(signature[:], signature[:], nonce, receiverID)

	senderPublicKey, _, err := btcec.RecoverCompact(btcec.S256(), signature[:], chainhash.DoubleHashB(raw[:len(raw)-signatureSize]))
	if err != nil {
		return nil, err
	}

	// Decrypt the packet using Salsa20.
	bufferDecrypted := make([]byte, len(raw)-signatureSize-4) // full length -signature -nonce
	salsa20.XORKeyStream(bufferDecrypted[:], raw[4:len(raw)-signatureSize], nonce, receiverID)

	// copy all fields
	packet = &packetRaw{SenderPublicKey: senderPublicKey, Protocol: bufferDecrypted[0], Command: bufferDecrypted[1]}

	sizePayload := binary.LittleEndian.Uint16(bufferDecrypted[2:4])
	if int(sizePayload) > len(bufferDecrypted)-4 { // invalid length?
		return nil, errors.New("invalid length field")
	}
	if sizePayload > 0 {
		packet.Payload = make([]byte, int(sizePayload))
		copy(packet.Payload, bufferDecrypted[4:4+int(sizePayload)])
	}

	return packet, nil
}

// packetEncrypt encrypts a packet using the provided senders private key and receivers compressed public key.
func packetEncrypt(senderPrivateKey *btcec.PrivateKey, receiverID *[32]byte, packet *packetRaw) (raw []byte, err error) {
	raw = make([]byte, packetLengthMin+len(packet.Payload))

	// Nonce. TODO: Translate into a pseudo random number
	nonce := make([]byte, 8)
	nonceC := atomic.AddUint64(&nonce1, 1)
	binary.LittleEndian.PutUint32(nonce[0:4], uint32(nonceC))
	binary.LittleEndian.PutUint32(nonce[4:8], uint32(nonceC))
	copy(raw[0:4], nonce[0:4])

	raw[4] = packet.Protocol
	raw[5] = packet.Command

	binary.LittleEndian.PutUint16(raw[6:8], uint16(len(packet.Payload)))
	copy(raw[8:], packet.Payload)

	// TODO: Add garbage.

	// encrypt it using Salsa20
	salsa20.XORKeyStream(raw[4:8+len(packet.Payload)], raw[4:8+len(packet.Payload)], nonce, receiverID)

	// add signature
	signature, err := btcec.SignCompact(btcec.S256(), senderPrivateKey, chainhash.DoubleHashB(raw[:len(raw)-signatureSize]), true)
	if err != nil {
		return nil, err
	}

	salsa20.XORKeyStream(signature[:], signature[:], nonce, receiverID)
	copy(raw[len(raw)-signatureSize:], signature)

	return raw, nil
}
