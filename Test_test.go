// Random tests for debugging. Not actual functionality tests.
package main

import (
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/PeernetOfficial/core"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"lukechampine.com/blake3"
)

func TestSignCompact(t *testing.T) {
	privateKey, publicKey, err := core.Secp256k1NewPrivateKey()
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
	privateKey, publicKey, err := core.Secp256k1NewPrivateKey()
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		return
	}

	packet1 := core.PacketRaw{Protocol: 0, Command: 1, Payload: []byte{1, 2, 3}}

	raw1, err := core.PacketEncrypt(privateKey, publicKey, &packet1)
	if err != nil {
		return
	}

	packet1d, _, err := core.PacketDecrypt(raw1, publicKey)
	if err != nil {
		fmt.Printf("Error %s\n", err.Error())
		return
	}

	fmt.Printf("%d\n", packet1d.Command)
}

func BenchmarkHash(b *testing.B) {
	length := 20000

	b.Run("blake3", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(length))
		buf := make([]byte, length)
		rand.Read(buf)
		for i := 0; i < b.N; i++ {
			blake3.Sum256(buf)
		}
	})
	b.Run("SHA256 double", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(length))
		buf := make([]byte, length)
		rand.Read(buf)
		for i := 0; i < b.N; i++ {
			chainhash.DoubleHashB(buf)
		}
	})
}

func TestEncryption2(t *testing.T) {
	privateKey, publicKey, err := core.Secp256k1NewPrivateKey()
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		return
	}

	packet1 := core.PacketRaw{Protocol: 0, Command: 0}

	for n := 0; n < 1000; n++ {
		raw1, err := core.PacketEncrypt(privateKey, publicKey, &packet1)
		if err != nil {
			return
		}
		packet1d, _, err := core.PacketDecrypt(raw1, publicKey)
		if err != nil {
			fmt.Printf("Error %s\n", err.Error())
			return
		}
		fmt.Printf("Command %d payload %v\n", packet1d.Command, packet1d.Payload)
	}

}
