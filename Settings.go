/*
File Name:  Settings.go
Copyright:  2021 Peernet Foundation s.r.o.
Author:     Peter Kleissner
*/

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

const configFile = "Settings.yaml"

var config struct {
	LogFile string `yaml:"LogFile"` // Log file

	Listen []string `yaml:"Listen"` // IP:Port combinations

	// API server settings
	// TODO
}

// loadConfig reads the YAML configuration file
func loadConfig() {
	cfg, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Printf("Configuration file '%s' could not be found!\nPlease ensure that your current path leads to the right file.\n", configFile)
		os.Exit(1)
	}

	err = yaml.Unmarshal(cfg, &config)
	if err != nil {
		fmt.Printf("YAML data in '%s' couldn't be read!\nPlease ensure that the format of the file is valid YAML.\n", configFile)
		os.Exit(1)
	}

	// redirect all output to the log file
	logFile, err := os.OpenFile(config.LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("Error creating log file '%s': %v\n", config.LogFile, err)
	}
	//	defer logFile.Close()	// has to remain open until program closes

	log.SetOutput(logFile)
}
