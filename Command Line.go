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

func getUserOptionHash(reader *bufio.Reader) (hash []byte, valid bool) {
	responseA, err := reader.ReadString('\n')
	if err != nil {
		return nil, false
	}

	responseA = strings.TrimSpace(responseA)

	hash, err = hex.DecodeString(responseA)
	if err != nil || len(hash) != 256/8 {
		return nil, false
	}

	return hash, true
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
		"hash               Create blake3 hash of input\n" +
		"warehouse get      Get data from local warehouse by hash\n" +
		"warehouse store    Store data into local warehouse\n" +
		"dht get            Get data via DHT by hash\n" +
		"dht store          Store data into DHT\n" +
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
			fmt.Printf("Public Key:  %s\n", hex.EncodeToString(publicKey.SerializeCompressed()))

		case "debug key self":
			privateKey, publicKey := core.ExportPrivateKey()
			fmt.Printf("Private Key: %s\n", hex.EncodeToString(privateKey.Serialize()))
			fmt.Printf("Public Key:  %s\n", hex.EncodeToString(publicKey.SerializeCompressed()))

		case "peer list":
			for _, peer := range GetPeerlistSorted() {
				info := ""
				if peer.IsRootPeer {
					info = " [root peer]"
				}
				if peer.IsBehindNAT() {
					info += " [NAT]"
				}
				userAgent := strings.ToValidUTF8(peer.UserAgent, "?")

				fmt.Printf("* %s%s\n  User Agent: %s\n\n%s  Packets sent:      %d\n  Packets received:  %d\n\n", hex.EncodeToString(peer.PublicKey.SerializeCompressed()), info, userAgent, textPeerConnections(peer), peer.StatsPacketSent, peer.StatsPacketReceived)
			}

		case "chat all", "chat":
			if text, valid := getUserOptionString(reader); valid {
				core.SendChatAll(text)
			}

		case "status":
			_, publicKey := core.ExportPrivateKey()
			nodeID := core.SelfNodeID()
			fmt.Printf("----------------\nPublic Key: %s\nNode ID:    %s\n\n", hex.EncodeToString(publicKey.SerializeCompressed()), hex.EncodeToString(nodeID))

			features := ""
			featureSupport := core.FeatureSupport()
			if featureSupport&(1<<core.FeatureIPv4Listen) > 0 {
				features = "IPv4_LISTEN"
			}
			if featureSupport&(1<<core.FeatureIPv6Listen) > 0 {
				if len(features) > 0 {
					features += ", "
				}
				features += "IPv6_LISTEN"
			}

			fmt.Printf("User Agent: %s\nFeatures:   %s\n\n", core.UserAgent, features)

			fmt.Printf("Listen Address                                  Multicast IP out                  External Address\n")

			for _, network := range core.GetNetworks(4) {
				address, _, broadcastIPv4, ipExternal, externalPort := network.GetListen()

				broadcastIPsA := ""
				for n, broadcastIP := range broadcastIPv4 {
					if n > 0 {
						broadcastIPsA += ", "
					}
					broadcastIPsA += broadcastIP.String()
				}

				externalAddress := ""

				if ipExternal != nil && !ipExternal.IsUnspecified() || externalPort > 0 {
					externalIPA := "[unknown]"
					externalPortA := ""
					if ipExternal != nil && !ipExternal.IsUnspecified() {
						externalIPA = ipExternal.String()
					}
					if externalPort > 0 {
						externalPortA = strconv.Itoa(int(externalPort))
					}

					externalAddress = net.JoinHostPort(externalIPA, externalPortA)
				}

				fmt.Printf("%-46s  %-32s  %s\n", address.String(), broadcastIPsA, externalAddress)
			}
			for _, network := range core.GetNetworks(6) {
				address, multicastIP, _, _, externalPort := network.GetListen()

				externalPortA := ""
				if externalPort > 0 {
					externalPortA = strconv.Itoa(int(externalPort))
				}

				fmt.Printf("%-46s  %-31s  %s\n", address.String(), multicastIP.String(), externalPortA)
			}

			fmt.Printf("\nPeer ID                                                             Sent      Received  IP                                   Flags   RTT     \n")
			for _, peer := range GetPeerlistSorted() {
				addressA := "N/A"
				rttA := "N/A"
				if connectionsActive := peer.GetConnections(true); len(connectionsActive) > 0 {
					addressA = addressToA(connectionsActive[0].Address)
				}
				if rtt := peer.GetRTT(); rtt > 0 {
					rttA = rtt.Round(time.Millisecond).String()
				}
				flagsA := ""
				if peer.IsRootPeer {
					flagsA = "R"
				}
				if peer.IsBehindNAT() {
					flagsA += "N"
				}
				fmt.Printf("%-66s  %-8d  %-8d  %-35s  %-6s  %-6s\n", hex.EncodeToString(peer.PublicKey.SerializeCompressed()), peer.StatsPacketSent, peer.StatsPacketReceived, addressA, flagsA, rttA)
			}

			fmt.Printf("\n")

		case "hash":
			if text, valid := getUserOptionString(reader); valid {
				hash := core.Data2Hash([]byte(text))
				fmt.Printf("blake3 hash: %s\n", hex.EncodeToString(hash))
			}

		case "warehouse get":
			if hash, valid := getUserOptionHash(reader); valid {
				data, found := core.GetDataLocal(hash)
				if !found {
					fmt.Printf("Not found.\n")
				} else {
					fmt.Printf("Data hex:    %s\n", hex.EncodeToString(data))
					fmt.Printf("Data string: %s\n", string(data))
				}
			} else {
				fmt.Printf("Invalid hash. Hex-encoded blake3 hash as input is required.\n")
			}

		case "warehouse store":
			if text, valid := getUserOptionString(reader); valid {
				if err := core.StoreDataLocal([]byte(text)); err != nil {
					fmt.Printf("Error storing data: %s\n", err.Error())
					break
				}
				fmt.Printf("Stored via hash: %s\n", hex.EncodeToString(core.Data2Hash([]byte(text))))
			}

		case "dht store":
			if text, valid := getUserOptionString(reader); valid {
				if err := core.StoreDataDHT([]byte(text)); err != nil {
					fmt.Printf("Error storing data: %s\n", err.Error())
					break
				}
				fmt.Printf("Stored via hash: %s\n", hex.EncodeToString(core.Data2Hash([]byte(text))))
			}

		case "dht get":
			if hash, valid := getUserOptionHash(reader); valid {
				data, sender, found := core.GetDataDHT(hash)
				if !found {
					fmt.Printf("Not found.\n")
				} else {
					fmt.Printf("\nSender:      %s\n", hex.EncodeToString(sender))
					fmt.Printf("Data hex:    %s\n", hex.EncodeToString(data))
					fmt.Printf("Data string: %s\n", string(data))
				}
			} else {
				fmt.Printf("Invalid hash. Hex-encoded blake3 hash as input is required.\n")
			}

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

	text += "  Status     Local                                               ->  Remote                                              Last Packet In       Last Packet Out      RTT     Ports I/E  \n"

	for _, adapterName := range listAdapters {
		text += "  -- adapter '" + adapterName + "' --\n"

		list, _ := mapConnectionsA[adapterName]
		for _, c := range list {
			listenAddress, _, _, _, _ := c.Network.GetListen()
			rttA := "N/A"
			if c.RoundTripTime > 0 {
				rttA = c.RoundTripTime.Round(time.Millisecond).String()
			}

			portEA := strconv.Itoa(int(c.PortInternal))
			if c.PortExternal > 0 {
				portEA += " / " + strconv.Itoa(int(c.PortExternal))
			}

			text += fmt.Sprintf("  %-9s  %-50s  ->  %-50s  %-19s  %-19s  %-6s  %-9s  \n", connectionStatusToA(c.Status), listenAddress.String(), addressToA(c.Address), c.LastPacketIn.Format(dateFormat), c.LastPacketOut.Format(dateFormat), rttA, portEA)
		}

		list, _ = mapConnectionsI[adapterName]
		for _, c := range list {
			listenAddress, _, _, _, _ := c.Network.GetListen()
			rttA := "N/A"
			if c.RoundTripTime > 0 {
				rttA = c.RoundTripTime.Round(time.Millisecond).String()
			}

			portEA := strconv.Itoa(int(c.PortInternal))
			if c.PortExternal > 0 {
				portEA += " / " + strconv.Itoa(int(c.PortExternal))
			}

			text += fmt.Sprintf("  %-9s  %-50s  ->  %-50s  %-19s  %-19s  %-6s  %-9s  \n", connectionStatusToA(c.Status), listenAddress.String(), addressToA(c.Address), c.LastPacketIn.Format(dateFormat), c.LastPacketOut.Format(dateFormat), rttA, portEA)
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

// connectionStatusToA translates the connection status to a readable text
func connectionStatusToA(status int) (result string) {
	switch status {
	case core.ConnectionActive:
		return "active"
	case core.ConnectionInactive:
		return "inactive"
	case core.ConnectionRemoved:
		return "removed"
	case core.ConnectionRedundant:
		return "redundant"
	default:
		return "unknown"
	}
}

func GetPeerlistSorted() (peers []*core.PeerInfo) {
	peers = core.PeerlistGet()
	sort.Slice(peers, func(i, j int) bool {
		if peers[i].IsRootPeer && !peers[j].IsRootPeer {
			return true
		} else if peers[j].IsRootPeer && !peers[i].IsRootPeer {
			return false
		}
		return (string(peers[i].NodeID) > string(peers[j].NodeID))
	})

	return peers
}
