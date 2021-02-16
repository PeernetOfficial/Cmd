package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

func TestSignCompact(t *testing.T) {
	privateKey, publicKey, err := secp256k1NewPrivateKey()
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		return
	}

	var compact1 []byte
	data := []byte{1, 2, 3}

	time1 := time.Now()

	compact1 = make([]byte, 0)
	compact1, err = btcec.SignCompact(btcec.S256(), privateKey, chainhash.DoubleHashB(data), true)

	timeT := time.Since(time1)
	fmt.Printf("length %d time %s\n", len(compact1), timeT.String())

	time2 := time.Now()

	key2, _, err := btcec.RecoverCompact(btcec.S256(), compact1, chainhash.DoubleHashB(data))

	timeT2 := time.Since(time2)
	fmt.Printf("VALID PUBLIC KEY: %t\n", key2.IsEqual(publicKey))
	fmt.Printf("time %s\n", timeT2.String())
}

func TestEncryption1(t *testing.T) {
	privateKey, publicKey, err := secp256k1NewPrivateKey()
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		return
	}

	packet1 := packetRaw{Protocol: 0, Command: 1, Payload: []byte{1, 2, 3}}

	raw1, err := packetEncrypt(privateKey, publicKey, &packet1)
	if err != nil {
		return
	}

	packet1d, _, err := packetDecrypt(raw1, publicKey)
	if err != nil {
		fmt.Printf("Error %s\n", err.Error())
		return
	}

	fmt.Printf("%d\n", packet1d.Command)
}
