/*
File Name:  Network IPv6.go
Copyright:  2021 Peernet Foundation s.r.o.
Author:     Peter Kleissner
*/

package main

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

// network6 is a connection adapter through one network interface (adapter).
// Note that for each IP on the same adapter separate network entries are created.
type network6 struct {
	iface       *net.Interface // Network interface belonging to the IP
	ipnet       *net.IPNet     // IP network the listening address belongs to
	multicastIP net.IP         // the multicast IP
	listen      *net.UDPAddr   // IP:Port where the server listens
	socket      *net.UDPConn   // active socket for send/receive
}

// networks is a list of all connected networks
var networks6 []*network6

// Default ports to use. This may be randomized in the future to prevent fingerprinting (and subsequent blocking) by corporate and ISP firewalls.
const defaultPort = 'p'         // 112
const startPortDefault2 = 12912 // 03270h = 12912 = 'p2'
const maxPorts = 100

// AutoAssignPort assigns a port for the given IP. Use port 0 for zero configuration.
func (network *network6) AutoAssignPort(ip net.IP, port int) (err error) {
	// Find an available port. Sometimes the network interface might not be ready when the service is started immediately after boot. Full error: "listen udp6 [REDACTED]:[REDACTED]: bind: The requested address is not valid in its context."
	// To fix this, try up to 4 times, wait 20 seconds in between each try.
	connectPortTry := func(port int) (listen *net.UDPAddr, socket *net.UDPConn, err error) {
		for retryCount := 0; retryCount < 4; retryCount++ {
			listen = &net.UDPAddr{IP: ip, Port: port}
			socket, err = net.ListenUDP("udp6", listen)
			if err == nil {
				return listen, socket, nil
			}
			if !strings.Contains(err.Error(), "address is not valid in its context") {
				break
			}
			time.Sleep(time.Second * 20)
		}
		return nil, nil, err
	}

	if port != 0 {
		network.listen, network.socket, err = connectPortTry(port)
		return err
	}

	// try default main port, then alternative ones
	network.listen, network.socket, err = connectPortTry(defaultPort)
	if err == nil {
		return nil
	}

	for port := startPortDefault2; port < startPortDefault2+maxPorts; port++ {
		network.listen, network.socket, err = connectPortTry(defaultPort)
		if err == nil {
			return nil
		}
	}

	return err
}

// send sends a message
func (network *network6) send(IP net.IP, port int, raw []byte) (err error) {
	_, err = network.socket.WriteTo(raw, &net.UDPAddr{IP: IP, Port: port})
	return err
}

// Listen starts listening for incoming packets on the given UDP connection
func (network *network6) Listen() {
	buffer := make([]byte, 4096)

	for {
		length, sender, err := network.socket.ReadFromUDP(buffer)

		if err != nil {
			log.Printf("Error receiving UDP message: %v\n", err) // Only log for debug purposes.
			time.Sleep(time.Millisecond * 50)                    // In case of endless errors, prevent ddos of CPU.
			continue
		}

		if length < packetLengthMin {
			// Discard packets that do not meet the minimum length.
			continue
		}

		// send the packet to a channel which is processed by multiple workers.
		rawPacketsIncoming <- networkWire{network: network, sender: sender, raw: buffer[:length]}
	}
}

func handlePackets(packets <-chan networkWire) {
	for packet := range packets {
		decoded, senderPublicKey, err := packetDecrypt(packet.raw, &peerIDcompressed)
		if err != nil {
			log.Printf("Error decrypting packet: %s\n", err.Error())
			continue
		}

		// immediately discard message if sender = self
		if senderPublicKey.IsEqual(peerPublicKey) {
			continue
		}

		// process the packet
		message := packet2{SenderPublicKey: senderPublicKey, packetRaw: *decoded, network: packet.network, sender: packet.sender}

		switch decoded.Command {
		case 0: // Announce
			receiveAnnouncement(&message)

		case 10: // debug: chat
			fmt.Printf("Chat: %s\n", string(decoded.Payload))

		default: // Unknown
		}

	}
}

// demo via https://gist.github.com/guileen/b9cc825ed18eb8350539

/*

const (
	// login client -> server
	CMD_LOGIN byte = byte(iota)
	CMD_LOGIN_RES

	// user list server -> client
	CMD_LIST
	CMD_LIST_RES

	// ping server -> client
	CMD_PING
	CMD_PONG

	// cone client -> client
	CMD_CONE
	CMD_CONE_RES

	// message client -> client
	CMD_MEG
	CMD_MSG_RES
)

var userlist []*net.UDPAddr
var listenAddr *net.UDPAddr
var socket *net.UDPConn

func startListen() {
	var err error

	// 设置log参数
	log.SetFlags(log.Lshortfile)

	// 用户集合
	userlist = make([]*net.UDPAddr, 0, 10)

	port := 8000
PORT:

	// 本地地址
	listenAddr, err = net.ResolveUDPAddr("udp4", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Println(err)
	}

	// 创建连接
	socket, err = net.ListenUDP("udp4", listenAddr)
	if err != nil {
		log.Println("连接失败!", err)
		port++
		goto PORT
		return
	}
	defer socket.Close()

	// 上线
	login_data := make([]byte, 0, 10)
	login_data = append(login_data, CMD_LOGIN)
	login_data = append(login_data, []byte("nickname")...)

	// 发送上线数据

	// 读取消息
	go readMsg()

	// 用户交互
	readCmd()
}

// 用户交互
func readCmd() {
	for {
		fmt.Printf("p2p > ")

		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() {
			continue
		}

		var line = scanner.Text()
		if err := scanner.Err(); err != nil {
			log.Println("read error: ", err)
			continue
		}

		switch {
		case strings.HasPrefix(line, "help"):
			fmt.Println("  list: show all user list\n  send: send message\n\tsend <id> <message>")
		case strings.HasPrefix(line, "list"):
			fmt.Println("user list:")
			for id, user := range userlist {
				fmt.Println(id+1, user.IP, user.Port)
			}
		case strings.HasPrefix(line, "send"):
			id := 0
			content := ""
			fmt.Sscanf(line, "send %d %s", &id, &content)

			if id <= 0 || id > len(userlist) {
				fmt.Printf("error: id %d not fund\n", id)
				continue
			}

			log.Printf("send message: %s %d, %s", userlist[id-1], id, content)

			sendData := make([]byte, 0, 100)
			sendData = append(sendData, CMD_MEG)
			sendData = append(sendData, []byte(content)...)

			n, err := socket.WriteToUDP(sendData, userlist[id-1])
			log.Println(n, err)
		default:
			fmt.Printf("command error: %s\nuse the 'help' command to get help\n", line)
		}
	}
}

func readMsg() {
	for {
		// 接收数据
		data := make([]byte, 1024)
		read, addr, err := socket.ReadFromUDP(data)
		if err != nil {
			log.Println("读取数据失败!", err)
			continue
		}
		log.Printf("UDP: %d, %s, %s\n", read, addr, hex.EncodeToString(data[:read]))

		switch data[0] {
		case CMD_LOGIN_RES:

		case CMD_LIST_RES:
			for i := 1; i < read; i += 6 {
				addrData := data[i:]
				touser := &net.UDPAddr{
					IP:   net.IP(addrData[:4]),
					Port: int(binary.LittleEndian.Uint16(addrData[4:])),
				}

				coneData := make([]byte, 0, 10)
				coneData = append(coneData, CMD_CONE)
				coneData = append(coneData, []byte("nickname")...)

				socket.WriteToUDP(coneData, touser)
				log.Println("cone: ", touser, coneData)

				userlist = append(userlist, touser)
			}
		case CMD_PING:
			log.Printf("CMD_PING\n")
			pong_data := make([]byte, 0, 15)
			pong_data = append(pong_data, CMD_PONG, 1)
			n, err := socket.WriteTo(pong_data, addr)
			log.Println("CMD_PING: ", n, err)
		case CMD_PONG:

		case CMD_CONE:
			coneResData := make([]byte, 0, 10)
			coneResData = append(coneResData, CMD_CONE_RES)
			coneResData = append(coneResData, []byte("nickname")...)

			socket.WriteToUDP(coneResData, addr)
		case CMD_CONE_RES:
			log.Println("CMD_CONE_RES:", addr)
		case CMD_MEG:
			fmt.Println(string(data[1:read]))
		case CMD_MSG_RES:

		default:
			log.Println("default UDP: ", data[0])
		}
	}
}
*/
