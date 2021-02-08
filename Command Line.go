/*
File Name:  Command Line.go
Copyright:  2021 Peernet Foundation s.r.o.
Author:     Peter Kleissner
*/

package main

import (
	"bufio"
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
	fmt.Print("help          Show this help\n" +
		"net list all  Lists all IPs\n" +
		"\n")
}

func userCommands() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Peernet Cmd 0.1\n------------------------------\n")
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

		}
	}
}
