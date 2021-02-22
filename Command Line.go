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
		"net list           Lists all network adapters and their IPs\n" +
		"status             Get current status\n" +
		"chat               Send text to all peers\n" +
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

		case "net list":
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
				fmt.Printf("* %s\n", hex.EncodeToString(peer.PublicKey.SerializeCompressed()))
				if len(peer.connections) > 0 {
					fmt.Printf("  Connections:\n")
					for _, connection := range peer.connections {
						fmt.Printf("  %-30s  on adapter %s\n", connection.address.String(), connection.network.address.String())
					}
				} else {
					fmt.Printf("  Connections: [none]\n")
				}
				fmt.Printf("  Packets sent:      %d\n", peer.statsPacketSent)
				fmt.Printf("  Packets received:  %d\n", peer.statsPacketReceived)
			}

		case "chat all", "chat":
			if text, valid := getUserOptionString(reader); valid {
				sendChatAll(text)
			}

		case "status":
			fmt.Printf("----------------\nPublic Key: %s\n\n", hex.EncodeToString(peerPublicKey.SerializeCompressed()))

			fmt.Printf("Listen Address                  Multicast IP out\n")

			for _, network := range networks4 {
				broadcastIPA := ""
				if network.multicastIP != nil {
					broadcastIPA = network.multicastIP.String()
				}
				fmt.Printf("%-30s  %-30s\n", network.address.String(), broadcastIPA)
			}
			for _, network := range networks6 {
				fmt.Printf("%-30s  %-30s\n", network.address.String(), network.multicastIP.String())
			}

			fmt.Printf("\nPeer ID                                                             IP                              Sent      Received\n")
			for _, peer := range PeerlistGet() {
				fmt.Printf("%-66s  %-30s  %-8d  %-8d\n", hex.EncodeToString(peer.PublicKey.SerializeCompressed()), peer.connections[0].address.String(), peer.statsPacketSent, peer.statsPacketReceived)
			}

			fmt.Printf("\n")
		}
	}
}
