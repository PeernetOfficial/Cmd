/*
File Name:  Command Line.go
Copyright:  2021 Peernet Foundation s.r.o.
Author:     Peter Kleissner
*/

package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func getUserOptionString(reader *bufio.Reader) (response string, valid bool) {
	responseA, err := reader.ReadString('\n')
	if err != nil {
		return "", false
	}

	responseA = strings.TrimSpace(responseA) // also removes the delimiter

	return responseA, true
}

func getUserOptionBool(reader *bufio.Reader) (response bool, valid bool) {
	responseA, err := reader.ReadString('\n')
	if err != nil {
		return false, false
	}

	responseA = strings.TrimSpace(responseA) // also removes the delimiter

	responseI, err := strconv.Atoi(responseA)
	if err != nil || (responseI != 0 && responseI != 1) {
		return false, false
	}

	return responseI == 1, true
}

func getUserOptionInt(reader *bufio.Reader) (response int, valid bool) {
	responseA, err := reader.ReadString('\n')
	if err != nil {
		return 0, false
	}

	responseA = strings.TrimSpace(responseA) // also removes the delimiter

	responseI, err := strconv.Atoi(responseA)
	if err != nil {
		return 0, false
	}

	return responseI, true
}

func showHelp() {
	fmt.Print("Please enter a command:\n")
	fmt.Print("help               Show this help\n" +
		"connect            Connect to Peernet\n" +
		"disconnect         Disconnect from Peernet\n" +
		"net list all       Lists all IPs\n" +
		"status             Get current status\n" +
		"chat all           Send text to all peers\n" +
		"peer list          List current peers\n" +
		"debug key create   Create Public-Private Key pair\n" +
		"debug key self     List current Public-Private Key pair\n" +
		"\n")
}

func userCommands() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Peernet Cmd " + version + "\n------------------------------\n")
	showHelp()

	for {
		command, valid := getUserOptionString(reader)
		if !valid {
			time.Sleep(time.Second)
			continue
		}

		switch command {
		case "help", "?":
			showHelp()

		case "net list all":
			fmt.Print(NetworkListOutput())

		case "debug key create":
			privateKey, publicKey, err := secp256k1NewPrivateKey()
			if err != nil {
				fmt.Printf("Error: %s\n", err.Error())
				return
			}

			fmt.Printf("Private Key: %s\n", hex.EncodeToString(privateKey.Serialize()))
			fmt.Printf("Public Key: %s\n", hex.EncodeToString(publicKey.SerializeCompressed()))

		case "debug key self":
			fmt.Printf("Private Key: %s\n", hex.EncodeToString(peerPrivateKey.Serialize()))
			fmt.Printf("Public Key: %s\n", hex.EncodeToString(peerPublicKey.SerializeCompressed()))

		case "peer list":
			for _, peer := range PeerlistGet() {
				fmt.Printf("* %s\n", peer.address.String())
				fmt.Printf("  Public Key: %s\n", hex.EncodeToString(peer.peerPublicKey.SerializeCompressed()))
				network := "(not set)"
				if peer.network != nil {
					network = peer.network.listen.String()
				}
				fmt.Printf("  Network adapter: %s\n", network)
				fmt.Printf("  Packets sent: %d\n", peer.statsPacketSent)
				fmt.Printf("  Packets received: %d\n", peer.statsPacketReceived)
			}

		case "chat all":
			if text, valid := getUserOptionString(reader); valid {
				sendChatAll(text)
			}

		}
	}
}
