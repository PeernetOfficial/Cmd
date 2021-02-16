/*
File Name:  Peer ID.go
Copyright:  2021 Peernet Foundation s.r.o.
Author:     Peter Kleissner
*/

package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

// peerID is the current peers ID. It is a ECDSA (secp256k1) 257-bit public key.
var peerIDcompressed [32]byte
var peerPrivateKey *btcec.PrivateKey
var peerPublicKey *btcec.PublicKey

func initPeerID() {
	// load existing key from config, if available
	if len(config.PrivateKey) > 0 {
		configPK, err := hex.DecodeString(config.PrivateKey)
		if err == nil {
			peerPrivateKey, peerPublicKey = btcec.PrivKeyFromBytes(btcec.S256(), configPK)
			copy(peerIDcompressed[:], peerPublicKey.SerializeCompressed()[1:])
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

	copy(peerIDcompressed[:], peerPublicKey.SerializeCompressed()[1:])

	// save the newly generated private key into the config
	config.PrivateKey = hex.EncodeToString(peerPublicKey.SerializeCompressed())

	saveConfig()
}

// TODO: All below private key algorithms (except generating the private key, possibly) are not needed.

func secp256k1NewPrivateKey() (privateKey *btcec.PrivateKey, publicKey *btcec.PublicKey, err error) {
	key, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		return nil, nil, err
	}

	//pk := (*btcec.PublicKey)(&key.PublicKey).SerializeCompressed()

	return key, (*btcec.PublicKey)(&key.PublicKey), nil
}

// secp256k1VerifySignature verifies the signature of the data using the public key
func secp256k1VerifySignature(publicKey *btcec.PublicKey, signature *btcec.Signature, data []byte) (valid bool, err error) {
	// TODO MOVE ME
	/*signature2, err := btcec.ParseSignature(signature, btcec.S256())
	if err != nil {
		fmt.Println(err)
		return false, err
	}*/

	// Verify the signature for the message using the public key.
	messageHash := chainhash.DoubleHashB(data) //TODO: Change doublehash to SHA3? or single SHA2. run speed tests.
	verified := signature.Verify(messageHash, publicKey)
	fmt.Println("Signature Verified?", verified)

	return verified, nil
}

func secp256k1SignData(privateKey *btcec.PrivateKey, data []byte) (signature []byte, err error) {
	messageHash := chainhash.DoubleHashB(data)
	signatureB, err := privateKey.Sign(messageHash)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return signatureB.Serialize(), nil
}

// TODO: SHA3 is not suspectible to the length attack, and might be faster than double hash.

//https://pkg.go.dev/github.com/btcsuite/btcd/btcec#NewPrivateKey

/*
https://github.com/btccom/secp256k1-go/blob/master/secp256k1/secp256k1.go

// EcdsaSignatureParseCompact parses an ECDSA signature in compact (64
// bytes) format. The return code is 1 when the signature could be
// parsed, 0 otherwise. The signature must consist of a 32-byte big
// endian R value, followed by a 32-byte big endian S value. If R or S fall
// outside of [0..order-1], the encoding is invalid. R and S with value 0
// are allowed in the encoding. After the call, sig will always be
// initialized. If parsing failed or R or S are zero, the resulting sig
// value is guaranteed to fail validation for any message and public key.
func EcdsaSignatureParseCompact(ctx *Context, signature []byte) (int, *EcdsaSignature, error) {
	if len(signature) != LenCompactSig {
		return 0, nil, errors.New(ErrorCompactSigSize)
	}

	sig := newEcdsaSignature()
	result := int(C.secp256k1_ecdsa_signature_parse_compact(ctx.ctx, sig.sig,
		(*C.uchar)(unsafe.Pointer(&signature[0])),
	))
	if result != 1 {
		return result, nil, errors.New(ErrorCompactSigParse)
	}
	return result, sig, nil
}

// Serialize an ECDSA signature in compact (64 byte) format. Return code is
// always 1. See EcdsaSignatureParseCompact for details about the encoding.
func EcdsaSignatureSerializeCompact(ctx *Context, sig *EcdsaSignature) (int, []byte, error) {
	output := make([]C.uchar, LenCompactSig)
	result := int(C.secp256k1_ecdsa_signature_serialize_compact(ctx.ctx, &output[0], sig.sig))
	return result, goBytes(output, C.int(LenCompactSig)), nil
}

int secp256k1_ecdsa_signature_serialize_compact(const secp256k1_context* ctx, unsigned char *output64, const secp256k1_ecdsa_signature* sig) {
    secp256k1_scalar r, s;

    VERIFY_CHECK(ctx != NULL);
    ARG_CHECK(output64 != NULL);
    ARG_CHECK(sig != NULL);

    secp256k1_ecdsa_signature_load(ctx, &r, &s, sig);
    secp256k1_scalar_get_b32(&output64[0], &r);
    secp256k1_scalar_get_b32(&output64[32], &s);
    return 1;
}

secp256k1.c
*/

// packet2 is a high-level message between peers
type packet2 struct {
	SenderPublicKey *btcec.PublicKey // Sender Public Key, ECDSA (secp256k1) 257-bit
	packetRaw
	network *network6    // network which received the packet
	sender  *net.UDPAddr // sender of the packet
}

// peer keeps information about a single remote peer
type peerInfo struct {
	peerIDcompressed [32]byte
	peerPublicKey    *btcec.PublicKey
	network          *network6    // network which received the packet
	sender           *net.UDPAddr // sender of the packet
}

var peerList []peerInfo

func (peer *peerInfo) send(packet *packetRaw) {
	raw, err := packetEncrypt(peerPrivateKey, &peer.peerIDcompressed, packet)
	if err != nil {
		log.Printf("Error creating packet: %s\n", err.Error())
		return
	}

	// send out the wire
	err = peer.network.send(peer.sender.IP, peer.sender.Port, raw)
	if err != nil {
		log.Printf("Error sending UDP packet: %v\n", err)
	}
}
