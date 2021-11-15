/*
File Name:  API.go
Copyright:  2021 Peernet Foundation s.r.o.
Author:     Peter Kleissner
*/

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PeernetOfficial/core"
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
	} else {
		return
	}

	webapi.Router.HandleFunc("/console", apiConsole).Methods("GET")
	webapi.Router.HandleFunc("/shutdown", apiShutdown).Methods("GET")
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

/*
apiShutdown gracefully shuts down the application. Actions: 0 = Shutdown.

Request:    GET /shutdown?action=[action]
Result:     200 with JSON structure apiShutdownStatus
*/
func apiShutdown(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	action, err := strconv.Atoi(r.Form.Get("action"))
	if err != nil || action != 0 {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	if action == 0 {
		// Later: Initiate shutdown signal to core library and wait for all requests to complete.

		core.Filters.LogError("apiShutdown", "Graceful shutdown via API requested from '%s'\n", r.RemoteAddr)

		EncodeJSONFlush(w, r, &apiShutdownStatus{Status: 0})

		os.Exit(core.ExitGraceful)
	}
}

type apiShutdownStatus struct {
	Status int `json:"status"` // Status of the API call. 0 = Success.
}

// EncodeJSONFlush encodes the data as JSON and flushes the writer. It sets the Content-Length header so no subsequent writes should be made.
func EncodeJSONFlush(w http.ResponseWriter, r *http.Request, data interface{}) (err error) {
	response, err := json.Marshal(data)
	if err != nil {
		core.Filters.LogError("EncodeJSONFlush", "Error marshalling data for route '%s': %v\n", r.URL.Path, err)
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(response)))
	w.WriteHeader(http.StatusOK)

	_, err = w.Write(response)

	// Flushing the buffer immediately is needed in case the application exits immediately after this call.
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	return
}
