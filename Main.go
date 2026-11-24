/*
File Name:  Main.go
Copyright:  2021 Kleissner Ventures Ltd
Author:     Peter Kleissner
*/

package main

import (
	"fmt"
	"os"
	"uuid"

	"github.com/PeernetOfficial/core"
)

const configFile = "Config.yaml"
const appName = "Peernet Cmd"

var config struct {
	// Warning: These settings are currently overwritten (deleted) when the config file is updated by core.
	// In the future the core package will consider custom config fields.

	// API settings
	APIListen          []string  `yaml:"APIListen"`          // WebListen is in format IP:Port and declares where the web-interface should listen on. IP can also be ommitted to listen on any.
	APIUseSSL          bool      `yaml:"APIUseSSL"`          // Enables SSL.
	APICertificateFile string    `yaml:"APICertificateFile"` // This is the certificate received from the CA. This can also include the intermediate certificate from the CA.
	APICertificateKey  string    `yaml:"APICertificateKey"`  // This is the private key.
	APITimeoutRead     string    `yaml:"APITimeoutRead"`     // The maximum duration for reading the entire request, including the body.
	APITimeoutWrite    string    `yaml:"APITimeoutWrite"`    // The maximum duration before timing out writes of the response. This includes processing time and is therefore the max time any HTTP function may take.
	APIKey             uuid.UUID `yaml:"APIKey"`             // API key. Empty UUID 00000000-0000-0000-0000-000000000000 = not used.
	DebugAPI           bool      `yaml:"DebugAPI"`           // Enables the debug API which allows profiling. Do not enable in production. Only available if compiled with debug tag.

	// Blockchain cache settings: the reasons for which the blockchain of another peer is kept.
	CacheRequested bool `yaml:"CacheRequested"` // Keeps a blockchain that the user requested, for 30 days.
	CacheNeighbour bool `yaml:"CacheNeighbour"` // Keeps the blockchains of the peers that this peer is among the 20 closest peers to, for 1 to 7 days after the peer was last seen.
	CacheMet       bool `yaml:"CacheMet"`       // Keeps the blockchain of every peer met, for 24 hours after the peer was last seen.
}

func main() {
	userAgent := appName + "/" + core.Version

	filters := &core.Filters{
		DHTSearchStatus:        filterSearchStatus,
		IncomingRequest:        filterIncomingRequest,
		MessageIn:              filterMessageIn,
		MessageOutAnnouncement: filterMessageOutAnnouncement,
		MessageOutResponse:     filterMessageOutResponse,
		MessageOutTraverse:     filterMessageOutTraverse,
		MessageOutPing:         filterMessageOutPing,
		MessageOutPong:         filterMessageOutPong,
		BlockchainReported:     filterBlockchainReported,
	}

	// Defaults of the settings. A setting in the config file replaces its default.
	config.CacheRequested = true
	config.CacheNeighbour = true

	backend, status, err := core.Init(userAgent, configFile, filters, &config)

	if status != core.ExitSuccess {
		switch status {
		case core.ExitErrorLocked:
			fmt.Printf("Another process is running with the config file '%s'.\n", configFile)
		case core.ExitErrorLockFile:
			fmt.Printf("Error locking the file '%s.lock': %s\n", configFile, err.Error())
		case core.ExitErrorConfigAccess:
			fmt.Printf("Unknown error accessing config file '%s': %s\n", configFile, err.Error())
		case core.ExitErrorConfigRead:
			fmt.Printf("Error reading config file '%s': %s\n", configFile, err.Error())
		case core.ExitErrorConfigParse:
			fmt.Printf("Error parsing config file '%s' (make sure it is valid YAML format): %s\n", configFile, err.Error())
		case core.ExitErrorLogInit:
			fmt.Printf("Error opening log file: %s\n", err.Error())
		default:
			fmt.Printf("Unknown error %d initializing backend: %s\n", status, err.Error())
		}
		os.Exit(status)
	}

	backend.Stdout.Subscribe(os.Stdout)

	apiListen, apiKey, watchPID := parseCmdParams()
	startAPI(backend, apiListen, apiKey)

	go processExitMonitor(backend, watchPID)

	backend.Connect()

	userCommands(backend, os.Stdin, os.Stdout, nil)
}
