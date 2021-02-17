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
	//initMulticastIPv4)
	initNetwork()
	loadSeedList()
}

func main() {
	go autoAnnounce()
	userCommands()
}

