/*
File Name:  Main.go
Copyright:  2021 Peernet Foundation s.r.o.
Author:     Peter Kleissner
*/

package main

import (
	"fmt"
	"os"

	"github.com/PeernetOfficial/core"
)

const configFile = "Config.yaml"
const appName = "Peernet Cmd"

var config struct {
	// Log settings
	ErrorOutput int `yaml:"ErrorOutput"` // 0 = Log file (default),  1 = Command line, 2 = Log file + command line, 3 = None
}

func init() {
	if status, err := core.LoadConfigOut(configFile, &config); err != nil {
		switch status {
		case 0:
			fmt.Printf("Unknown error accessing config file '%s': %s", configFile, err.Error())
		case 1:
			fmt.Printf("Error reading config file '%s': %s", configFile, err.Error())
		case 2:
			fmt.Printf("Error parsing config file '%s' (make sure it is valid YAML format): %s", configFile, err.Error())
		case 3:
			fmt.Printf("Unknown error loading config file '%s': %s", configFile, err.Error())
		}
		os.Exit(1)
	}

	if err := core.InitLog(); err != nil {
		fmt.Printf("Error opening log file: %s", err.Error())
		os.Exit(1)
	}

	monitorKeys = make(map[string]struct{})

	core.UserAgent = appName + "/" + core.Version
	core.Filters.LogError = logError
	core.Filters.DHTSearchStatus = filterSearchStatus

	core.Init()
}

func main() {
	core.Connect()

	userCommands()
}
