/*
File Name:  API.go
Copyright:  2021 Peernet Foundation s.r.o.
Author:     Peter Kleissner
*/

package main

import (
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/PeernetOfficial/core"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

func startAPI() {
	if len(config.APIListen) == 0 {
		return
	}

	// by default allow all requests
	wsUpgrader.CheckOrigin = func(r *http.Request) bool {
		return true
	}

	router := mux.NewRouter()

	router.HandleFunc("/test", apiTest).Methods("GET")
	router.HandleFunc("/status", apiStatus).Methods("GET")
	router.HandleFunc("/peer/self", apiPeerSelf).Methods("GET")
	router.HandleFunc("/console", apiConsole).Methods("GET")

	for _, listen := range config.APIListen {
		go startWebServer(listen, config.APIUseSSL, config.APICertificateFile, config.APICertificateKey, router, "API", parseDuration(config.APITimeoutRead), parseDuration(config.APITimeoutWrite))
	}
}

// startWebServer starts a web-server with given parameters and logs the status. If may block forever and only returns if there is an error.
func startWebServer(WebListen string, UseSSL bool, CertificateFile, CertificateKey string, Handler http.Handler, Info string, ReadTimeout, WriteTimeout time.Duration) {
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12} // for security reasons disable TLS 1.0/1.1

	server := &http.Server{
		Addr:         WebListen,
		Handler:      Handler,
		ReadTimeout:  ReadTimeout,  // ReadTimeout is the maximum duration for reading the entire request, including the body.
		WriteTimeout: WriteTimeout, // WriteTimeout is the maximum duration before timing out writes of the response. This includes processing time and is therefore the max time any HTTP function may take.
		//IdleTimeout:  IdleTimeout,  // IdleTimeout is the maximum amount of time to wait for the next request when keep-alives are enabled.
		TLSConfig: tlsConfig,
	}

	if UseSSL {
		// HTTPS
		if err := server.ListenAndServeTLS(CertificateFile, CertificateKey); err != nil {
			log.Printf("Error listening on '%s': %v\n", WebListen, err)
		}
	} else {
		// HTTP
		if err := server.ListenAndServe(); err != nil {
			log.Printf("Error listening on '%s': %v\n", WebListen, err)
		}
	}
}

// parseDuration is the same as time.ParseDuration without returning an error. Valid units are ms, s, m, h. For example "10s".
func parseDuration(input string) (result time.Duration) {
	result, _ = time.ParseDuration(input)
	return
}

func apiEncodeJSON(w http.ResponseWriter, r *http.Request, data interface{}) (err error) {
	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(data)
	if err != nil {
		log.Printf("Error writing data for route '%s': %v\n", r.URL.Path, err)
	}

	return err
}

func apiTest(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

type apiResponseStatus struct {
	Status        int  `json:"status"`        // Status code: 0 = Ok.
	IsConnected   bool `json:"isconnected"`   // Whether connected to Peernet.
	CountPeerList int  `json:"countpeerlist"` // Count of peers in the peer list. Note that this contains peers that are considered inactive, but have not yet been removed from the list.
}

/* apiStatus returns the current connectivity status to the network
Request:    GET /status
Result:     200 with JSON structure Status
*/
func apiStatus(w http.ResponseWriter, r *http.Request) {
	status := apiResponseStatus{Status: 0, CountPeerList: core.PeerlistCount()}

	// Connected: If at leat 2 peers.
	// This metric needs to be improved in the future, as root peers never disconnect.
	// Instead, the core should keep a count of "active peers".
	status.IsConnected = status.CountPeerList >= 2

	apiEncodeJSON(w, r, status)
}

type apiResponsePeerSelf struct {
	PeerID string `json:"peerid"` // Peer ID. This is derived from the public in compressed form.
	NodeID string `json:"nodeid"` // Node ID. This is the blake3 hash of the peer ID and used in the DHT.
}

/* apiPeerSelf provides information about the self peer details
Request:    GET /peer/self
Result:     200 with JSON structure apiResponsePeerSelf
*/
func apiPeerSelf(w http.ResponseWriter, r *http.Request) {
	response := apiResponsePeerSelf{}
	response.NodeID = hex.EncodeToString(core.SelfNodeID())

	_, publicKey := core.ExportPrivateKey()
	response.PeerID = hex.EncodeToString(publicKey.SerializeCompressed())

	apiEncodeJSON(w, r, response)
}

var wsUpgrader = websocket.Upgrader{} // use default options

/* apiConsole provides a websocket to send/receive internal commands
Request:    GET /console
Result:     200 with JSON structure apiResponsePeerSelf
*/
func apiConsole(w http.ResponseWriter, r *http.Request) {
	c, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		// May happen if request is simple HTTP request.
		return
	}

	// start reader/writer to forward

	defer c.Close()
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			//fmt.Println("read:", err)
			break
		}
		//c.NextWriter(websocket.BinaryMessage)

		//c.NextReader()
		//userCommands()
		fmt.Printf("recv: %s", message)
		//err = c.WriteMessage(mt, message)
		//if err != nil {
		//	//fmt.Println("write:", err)
		//	break
		//}
	}
}
