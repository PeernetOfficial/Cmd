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
	"sort"
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

	responseA = strings.TrimSpace(responseA)

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
		command = strings.ToLower(command)

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
				fmt.Printf("%s", textPeerConnections(peer))
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

			fmt.Printf("\nPeer ID                                                             Sent      Received  IP                              \n")
			for _, peer := range core.PeerlistGet() {
				addressA := "N/A"
				if connectionsActive := peer.GetConnections(true); len(connectionsActive) > 0 {
					addressA = connectionsActive[0].Address.String()
				}
				fmt.Printf("%-66s  %-8d  %-8d  %-30s  \n", hex.EncodeToString(peer.PublicKey.SerializeCompressed()), peer.StatsPacketSent, peer.StatsPacketReceived, addressA)
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

const dateFormat = "2006-01-02 15:04:05"

func textPeerConnections(peer *core.PeerInfo) (text string) {
	connectionsActive := peer.GetConnections(true)
	connectionsInactive := peer.GetConnections(false)

	mapConnectionsA := make(map[string][]*core.Connection)
	mapConnectionsI := make(map[string][]*core.Connection)
	var listAdapters []string

	// for better human readability, sort all connections based on the network name
	for _, c := range connectionsActive {
		adapterName := c.Network.GetAdapterName()

		list, ok := mapConnectionsA[adapterName]
		if ok {
			mapConnectionsA[adapterName] = append(list, c)
		} else {
			mapConnectionsA[adapterName] = []*core.Connection{c}
			listAdapters = append(listAdapters, adapterName)
		}
	}

	for _, c := range connectionsInactive {
		adapterName := c.Network.GetAdapterName()

		_, ok1 := mapConnectionsA[adapterName]
		if !ok1 {
			if _, ok2 := mapConnectionsI[adapterName]; !ok2 {
				listAdapters = append(listAdapters, adapterName)
			}
		}

		list, ok := mapConnectionsI[adapterName]
		if ok {
			mapConnectionsI[adapterName] = append(list, c)
		} else {
			mapConnectionsI[adapterName] = []*core.Connection{c}
		}
	}

	sort.Strings(listAdapters)

	text += "  Status    Local                                               ->  Remote                                              Last Packet In       Last Packet Out      \n"

	for _, adapterName := range listAdapters {
		text += "  -- adapter '" + adapterName + "' --\n"

		list, _ := mapConnectionsA[adapterName]
		for _, c := range list {
			listenAddress, _, _ := c.Network.GetListen()
			text += fmt.Sprintf("  active    %-50s  ->  %-50s  %-19s  %-19s\n", listenAddress.String(), addressToA(c.Address), c.LastPacketIn.Format(dateFormat), c.LastPacketOut.Format(dateFormat))
		}

		list, _ = mapConnectionsI[adapterName]
		for _, c := range list {
			listenAddress, _, _ := c.Network.GetListen()
			text += fmt.Sprintf("  inactive  %-50s  ->  %-50s  %-19s  %-19s\n", listenAddress.String(), addressToA(c.Address), c.LastPacketIn.Format(dateFormat), c.LastPacketOut.Format(dateFormat))
		}
	}

	text += "  --\n"

	return text
}

// addressToA is UDPAddr.String without IPv6 zone
func addressToA(a *net.UDPAddr) (result string) {
	if a == nil || len(a.IP) == 0 {
		return "<nil>"
	}
	return net.JoinHostPort(a.IP.String(), strconv.Itoa(a.Port))
}
