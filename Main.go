/*
File Name:  Main.go
Copyright:  2021 Peernet Foundation s.r.o.
Author:     Peter Kleissner
*/

package main

func init() {
	loadConfig()
	initPeerID()
	initMulticastIPv6()
	initBroadcastIPv4()
	initNetwork()
	initSeedList()
}

func main() {
	go bootstrap()
	go autoMulticastBroadcast()
	go autoAnnounce()
	
	userCommands()
}
