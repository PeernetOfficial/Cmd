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
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PeernetOfficial/core"
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

	fmt.Print("Peernet Cmd " + core.Version + "\n------------------------------\n")
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
			privateKey, publicKey, err := core.Secp256k1NewPrivateKey()
			if err != nil {
				fmt.Printf("Error: %s\n", err.Error())
				return
			}

			fmt.Printf("Private Key: %s\n", hex.EncodeToString(privateKey.Serialize()))
			fmt.Printf("Public Key: %s\n", hex.EncodeToString(publicKey.SerializeCompressed()))

		case "debug key self":
			privateKey, publicKey := core.ExportPrivateKey()
			fmt.Printf("Private Key: %s\n", hex.EncodeToString(privateKey.Serialize()))
			fmt.Printf("Public Key: %s\n", hex.EncodeToString(publicKey.SerializeCompressed()))

		case "peer list":
			for _, peer := range core.PeerlistGet() {
				fmt.Printf("* %s\n", hex.EncodeToString(peer.PublicKey.SerializeCompressed()))
				if len(peer.Connections) > 0 {
					fmt.Printf("  Connections:\n")
					for _, connection := range peer.Connections {
						address, _, _ := connection.Network.GetListen()
						fmt.Printf("  %-30s  on adapter %s\n", connection.Address.String(), address.String())
					}
				} else {
					fmt.Printf("  Connections: [none]\n")
				}
				fmt.Printf("  Packets sent:      %d\n", peer.StatsPacketSent)
				fmt.Printf("  Packets received:  %d\n", peer.StatsPacketReceived)
			}

		case "chat all", "chat":
			if text, valid := getUserOptionString(reader); valid {
				core.SendChatAll(text)
			}

		case "status":
			_, publicKey := core.ExportPrivateKey()
			fmt.Printf("----------------\nPublic Key: %s\n\n", hex.EncodeToString(publicKey.SerializeCompressed()))

			fmt.Printf("Listen Address                  Multicast IP out\n")

			for _, network := range core.GetNetworks(4) {
				address, _, broadcastIPv4 := network.GetListen()
				fmt.Printf("%-30s\n", address.String())

				for _, broadcastIP := range broadcastIPv4 {
					fmt.Printf("  %-30s\n", broadcastIP.String())
				}
			}
			for _, network := range core.GetNetworks(6) {
				address, multicastIP, _ := network.GetListen()
				fmt.Printf("%-30s  %-30s\n", address.String(), multicastIP.String())
			}

			fmt.Printf("\nPeer ID                                                             IP                              Sent      Received\n")
			for _, peer := range core.PeerlistGet() {
				fmt.Printf("%-66s  %-30s  %-8d  %-8d\n", hex.EncodeToString(peer.PublicKey.SerializeCompressed()), peer.Connections[0].Address.String(), peer.StatsPacketSent, peer.StatsPacketReceived)
			}

			fmt.Printf("\n")
		}
	}
}

// NetworkListOutput provides a user friendly output
func NetworkListOutput() (text string) {

	interfaceList, err := net.Interfaces()
	if err != nil {
		return "Error " + err.Error()
	}

	// iterate through all interfaces
	for _, ifaceSingle := range interfaceList {
		text += "Interface " + ifaceSingle.Name + ":\n"
		//text += "  MAC:        " + ifaceSingle.HardwareAddr.String() + "\n"

		addresses, err := ifaceSingle.Addrs()
		if err != nil {
			text += "  Error getting addresses: " + err.Error() + "\n\n"
			continue
		}

		// iterate through all IPs of the interfaces
		for _, address := range addresses {
			text += "  IP:         " + address.(*net.IPNet).IP.String() + "\n"
		}

		// Subscribed Multicast IPs of adapters are not really newsworthy
		//addresses2, err := ifaceSingle.MulticastAddrs()
		//if err != nil {
		//	text += "  Error getting multicast addresses: " + err.Error() + "\n\n"
		//	continue
		//}

		//for _, address := range addresses2 {
		//	text += "  Multicast:  " + address.(*net.IPAddr).IP.String() + "\n"
		//}

		text += "\n"
	}

	return text
}
