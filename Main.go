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
	"github.com/google/uuid"
)

const configFile = "Config.yaml"
const appName = "Peernet Cmd"

var config struct {
	// Log settings
	ErrorOutput int `yaml:"ErrorOutput"` // 0 = Log file (default),  1 = Command line, 2 = Log file + command line, 3 = None

	// API settings
	APIListen          []string  `yaml:"APIListen"`          // WebListen is in format IP:Port and declares where the web-interface should listen on. IP can also be ommitted to listen on any.
	APIUseSSL          bool      `yaml:"APIUseSSL"`          // Enables SSL.
	APICertificateFile string    `yaml:"APICertificateFile"` // This is the certificate received from the CA. This can also include the intermediate certificate from the CA.
	APICertificateKey  string    `yaml:"APICertificateKey"`  // This is the private key.
	APITimeoutRead     string    `yaml:"APITimeoutRead"`     // The maximum duration for reading the entire request, including the body.
	APITimeoutWrite    string    `yaml:"APITimeoutWrite"`    // The maximum duration before timing out writes of the response. This includes processing time and is therefore the max time any HTTP function may take.
	APIKey             uuid.UUID `yaml:"APIKey"`             // API key. Empty UUID 00000000-0000-0000-0000-000000000000 = not used.
}

func init() {
	if status, err := core.LoadConfigOut(configFile, &config); err != nil {
		var exitCode int
		switch status {
		case 0:
			exitCode = core.ExitErrorConfigAccess
			fmt.Printf("Unknown error accessing config file '%s': %s\n", configFile, err.Error())
		case 1:
			exitCode = core.ExitErrorConfigRead
			fmt.Printf("Error reading config file '%s': %s\n", configFile, err.Error())
		case 2:
			exitCode = core.ExitErrorConfigParse
			fmt.Printf("Error parsing config file '%s' (make sure it is valid YAML format): %s\n", configFile, err.Error())
		default:
			exitCode = core.ExitErrorConfigAccess
			fmt.Printf("Unknown error loading config file '%s': %s\n", configFile, err.Error())
		}
		os.Exit(exitCode)
	}

	if err := core.InitLog(); err != nil {
		fmt.Printf("Error opening log file: %s\n", err.Error())
		os.Exit(core.ExitErrorLogInit)
	}

	monitorKeys = make(map[string]struct{})

	core.Filters.LogError = logError
	core.Filters.DHTSearchStatus = filterSearchStatus
	core.Filters.IncomingRequest = filterIncomingRequest
	core.Filters.MessageIn = filterMessageIn
	core.Filters.MessageOutAnnouncement = filterMessageOutAnnouncement
	core.Filters.MessageOutResponse = filterMessageOutResponse
	core.Filters.MessageOutTraverse = filterMessageOutTraverse
	core.Filters.MessageOutPing = filterMessageOutPing
	core.Filters.MessageOutPong = filterMessageOutPong

	userAgent := appName + "/" + core.Version

	core.Init(userAgent)
}

func main() {
	apiListen, apiKey, watchPID := parseCmdParams()
	startAPI(apiListen, apiKey)

	go processExitMonitor(watchPID)

	core.Connect()

	userCommands(os.Stdin, os.Stdout, nil)
}
