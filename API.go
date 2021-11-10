/*
File Name:  API.go
Copyright:  2021 Peernet Foundation s.r.o.
Author:     Peter Kleissner
*/

package main

import (
	"bytes"
	"flag"
	"net/http"
	"strings"
	"time"

	"github.com/PeernetOfficial/core/webapi"
	"github.com/gorilla/websocket"
)

// startAPI starts the API if enabled via command line parameter or if the settings are set in the config file.
// Using the command line option always ignores any API settings from the config (including timeout settings).
func startAPI() {
	if apiListen := parseCmdParamWebapi(); len(apiListen) > 0 {
		// API listen parameter via command line argument.
		// Note that read and write timeouts are set to 0 which means they are not used. SSL is not enabled.
		webapi.Start(apiListen, false, "", "", 0, 0)

	} else if len(config.APIListen) != 0 {
		// API settings via config file.
		webapi.Start(config.APIListen, config.APIUseSSL, config.APICertificateFile, config.APICertificateKey, parseDuration(config.APITimeoutRead), parseDuration(config.APITimeoutWrite))
		return
	} else {
		return
	}

	webapi.Router.HandleFunc("/console", apiConsole).Methods("GET")
}

// parseDuration is the same as time.ParseDuration without returning an error. Valid units are ms, s, m, h. For example "10s".
func parseDuration(input string) (result time.Duration) {
	result, _ = time.ParseDuration(input)
	return
}

/*
apiConsole provides a websocket to send/receive internal commands
Request:    GET /console
Result:     200 with JSON structure apiResponsePeerSelf
*/
func apiConsole(w http.ResponseWriter, r *http.Request) {
	c, err := webapi.WSUpgrader.Upgrade(w, r, nil)
	if err != nil {
		// May happen if request is simple HTTP request.
		return
	}
	defer c.Close()

	bufferR := bytes.NewBuffer(make([]byte, 0, 4096))
	bufferW := bytes.NewBuffer(make([]byte, 0, 4096))

	terminateSignal := make(chan struct{})
	defer close(terminateSignal)

	// start userCommands which handles the actual commands
	go userCommands(bufferR, bufferW, terminateSignal)

	// go routine to receive output from userCommands and forward to websocket
	go func() {
		bufferW2 := make([]byte, 4096)
		for {
			select {
			case <-terminateSignal:
				return
			default:
			}

			countRead, err := bufferW.Read(bufferW2)
			if err != nil || countRead == 0 {
				time.Sleep(250 * time.Millisecond)
				continue
			}

			c.WriteMessage(websocket.TextMessage, bufferW2[:countRead])
		}
	}()

	// read from websocket loop and forward to the userCommands routine
	for {
		_, message, err := c.ReadMessage()
		if err != nil { // when channel is closed, an error is returned here
			break
		}

		// make sure the message has the \n delimiter which is used to detect a line
		if !bytes.HasSuffix(message, []byte{'\n'}) {
			message = append(message, '\n')
		}

		bufferR.Write(message)
	}
}

// parseCmdParamWebapi parses a "-webapi=" command line parameter
func parseCmdParamWebapi() (apiListen []string) {
	var param string
	flag.StringVar(&param, "webapi", "", "Specify the list of IP:Ports for the webapi to listen. Example: -webapi=127.0.0.1:1234")
	flag.Parse()

	if len(param) == 0 {
		return nil
	}

	return strings.Split(param, ",")
}
