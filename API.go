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
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// startAPI starts the API if enabled via command line parameter or if the settings are set in the config file.
// Using the command line option always ignores any API settings from the config (including timeout settings).
func startAPI(backend *core.Backend, apiListen []string, apiKey uuid.UUID) {
	var api *webapi.WebapiInstance

	if len(apiListen) > 0 {
		// API listen parameter via command line argument.
		// Note that read and write timeouts are set to 0 which means they are not used. SSL is not enabled.
		api = webapi.Start(backend, apiListen, false, "", "", 0, 0, apiKey)

	} else if len(config.APIListen) != 0 {
		// API settings via config file.
		api = webapi.Start(backend, config.APIListen, config.APIUseSSL, config.APICertificateFile, config.APICertificateKey, parseDuration(config.APITimeoutRead), parseDuration(config.APITimeoutWrite), config.APIKey)
	} else {
		return
	}

	api.InitGeoIPDatabase(backend.Config.GeoIPDatabase)

	api.AllowKeyInParam = append(api.AllowKeyInParam, "/console")
	api.Router.HandleFunc("/console", apiConsole(backend)).Methods("GET")
	api.Router.HandleFunc("/shutdown", apiShutdown(backend)).Methods("GET")

	if config.DebugAPI {
		attachDebugAPI(api)
	}
}

// parseCmdParams parses the "-webapi", "-apikey", and "-watchpid" command line parameters.
// The API key is optional (for now) and set to 00000000-0000-0000-0000-000000000000 if none is provided.
// The watch PID is set to 0 if not provided.
func parseCmdParams() (apiListen []string, apiKey uuid.UUID, watchPID int) {
	var paramWebapi, paramWebKeyA string
	flag.StringVar(&paramWebapi, "webapi", "", "Specify the list of IP:Ports for the webapi to listen. Example: -webapi=127.0.0.1:1234")
	flag.StringVar(&paramWebKeyA, "apikey", "", "Specify the API key to use. Must be a UUID.")
	flag.IntVar(&watchPID, "watchpid", 0, "Monitor the specified process ID for exit to exit this application")
	flag.Parse()

	if len(paramWebapi) == 0 {
		return nil, apiKey, watchPID
	}

	if len(paramWebKeyA) != 0 {
		var err error
		if apiKey, err = uuid.Parse(paramWebKeyA); err != nil {
			os.Exit(core.ExitParamApiKeyInvalid)
		}
	}

	return strings.Split(paramWebapi, ","), apiKey, watchPID
}

// parseDuration is the same as time.ParseDuration without returning an error. Valid units are ms, s, m, h. For example "10s".
func parseDuration(input string) (result time.Duration) {
	result, _ = time.ParseDuration(input)
	return
}

/*
apiConsole provides a websocket to send/receive internal commands.

Request:    GET /console
Result:     Upgrade to websocket. The websocket message are texts to read/write.
*/
func apiConsole(backend *core.Backend) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := webapi.WSUpgrader.Upgrade(w, r, nil)
		if err != nil {
			// May happen if request is simple HTTP request.
			return
		}
		defer c.Close()

		bufferR := bytes.NewBuffer(make([]byte, 0, 4096))
		bufferW := bytes.NewBuffer(make([]byte, 0, 4096))

		// subscribe to any output sent to backend.Stdout
		subscribeID := backend.Stdout.Subscribe(bufferW)
		defer backend.Stdout.Unsubscribe(subscribeID)

		// the terminate signal is used to signal the command handler in case the websocket is closed
		terminateSignal := make(chan struct{})
		defer close(terminateSignal)

		// start userCommands which handles the actual commands
		go userCommands(backend, bufferR, bufferW, terminateSignal)

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
					time.Sleep(50 * time.Millisecond)
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
}

/*
apiShutdown gracefully shuts down the application. Actions: 0 = Shutdown.

Request:    GET /shutdown?action=[action]
Result:     200 with JSON structure apiShutdownStatus
*/
func apiShutdown(backend *core.Backend) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		action, err := strconv.Atoi(r.Form.Get("action"))
		if err != nil || action != 0 {
			http.Error(w, "", http.StatusBadRequest)
			return
		}

		if action == 0 {
			// Later: Initiate shutdown signal to core library and wait for all requests to complete.

			backend.LogError("apiShutdown", "graceful shutdown via API requested from '%s'\n", r.RemoteAddr)

			EncodeJSONFlush(backend, w, r, &apiShutdownStatus{Status: 0})

			os.Exit(core.ExitGraceful)
		}
	}
}

type apiShutdownStatus struct {
	Status int `json:"status"` // Status of the API call. 0 = Success.
}

// EncodeJSONFlush encodes the data as JSON and flushes the writer. It sets the Content-Length header so no subsequent writes should be made.
func EncodeJSONFlush(backend *core.Backend, w http.ResponseWriter, r *http.Request, data interface{}) (err error) {
	response, err := json.Marshal(data)
	if err != nil {
		backend.LogError("EncodeJSONFlush", "error marshalling data for route '%s': %v\n", r.URL.Path, err)
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

// processExitMonitor monitors for a shutdown of a process based on its process ID (PID) and shuts down the application.
// It uses the command line parameter "-watchpid=[PID]".
// This can be useful to automatically shut down the application in case the frontend shuts down unexpectedly.
// Graceful shutdown should be initiated via the /shutdown API.
func processExitMonitor(backend *core.Backend, watchPID int) {
	if watchPID == 0 {
		return
	}

	// monitor the process
	process, err := os.FindProcess(watchPID)
	if err != nil {
		backend.LogError("processExitMonitor", "error finding monitored process ID %d: %v\n", watchPID, err)
		return
	}

	_, err = process.Wait()
	if err == nil {
		backend.LogError("processExitMonitor", "graceful shutdown via exit signal from process ID %d\n", watchPID)

		// Later: Initiate shutdown signal to core library and wait for all requests to complete.

		os.Exit(core.ExitGraceful)
	}
}
