/*
File Name:  API.go
Copyright:  2021 Peernet Foundation s.r.o.
Author:     Peter Kleissner
*/

package main

import (
	"bytes"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
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
	router.HandleFunc("/blockchain/self/header", apiBlockchainSelfHeader).Methods("GET")
	router.HandleFunc("/blockchain/self/append", apiBlockchainSelfAppend).Methods("POST")
	router.HandleFunc("/blockchain/self/read", apiBlockchainSelfRead).Methods("GET")
	router.HandleFunc("/blockchain/self/add/file", apiBlockchainSelfAddFile).Methods("POST")

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

// apiDecodeJSON decodes input JSON data server side sent either via GET or POST. It does not limit the maximum amount to read.
// In case of error it will automatically send an error to the client.
func apiDecodeJSON(w http.ResponseWriter, r *http.Request, data interface{}) (err error) {
	if r.Body == nil {
		http.Error(w, "", http.StatusBadRequest)
		return errors.New("no data")
	}

	err = json.NewDecoder(r.Body).Decode(data)
	if err != nil {
		http.Error(w, "", http.StatusBadRequest)
		return err
	}

	return nil
}

func apiTest(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

type apiResponseStatus struct {
	Status        int  `json:"status"`        // Status code: 0 = Ok.
	IsConnected   bool `json:"isconnected"`   // Whether connected to Peernet.
	CountPeerList int  `json:"countpeerlist"` // Count of peers in the peer list. Note that this contains peers that are considered inactive, but have not yet been removed from the list.
	CountNetwork  int  `json:"countnetwork"`  // Count of total peers in the network.
	// This is usually a higher number than CountPeerList, which just represents the current number of connected peers.
	// The CountNetwork number is going to be queried from root peers which may or may not have a limited view.
}

/* apiStatus returns the current connectivity status to the network
Request:    GET /status
Result:     200 with JSON structure Status
*/
func apiStatus(w http.ResponseWriter, r *http.Request) {
	status := apiResponseStatus{Status: 0, CountPeerList: core.PeerlistCount()}
	status.CountNetwork = status.CountPeerList // For now always same as CountPeerList, until native Statistics message to root peers is available.

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

type apiBlockchainHeader struct {
	PeerID  string `json:"peerid"`  // Peer ID hex encoded.
	Version uint64 `json:"version"` // Current version number of the blockchain.
	Height  uint64 `json:"height"`  // Height of the blockchain (number of blocks). If 0, no data exists.
}

/*
apiBlockchainSelfHeader returns the current blockchain header information

Request:    GET /blockchain/self/header
Result:     200 with JSON structure apiResponsePeerSelf
*/
func apiBlockchainSelfHeader(w http.ResponseWriter, r *http.Request) {
	publicKey, height, version := core.UserBlockchainHeader()

	apiEncodeJSON(w, r, apiBlockchainHeader{Version: version, Height: height, PeerID: hex.EncodeToString(publicKey.SerializeCompressed())})
}

type apiBlockRecordRaw struct {
	Type uint8  `json:"type"` // Record Type. See core.RecordTypeX.
	Data []byte `json:"data"` // Data according to the type.
}

// apiBlockchainBlockRaw contains a raw block of the blockchain via API
type apiBlockchainBlockRaw struct {
	Records []apiBlockRecordRaw `json:"records"` // Block records in encoded raw format.
}

type apiBlockchainBlockStatus struct {
	Status int    `json:"status"` // Status: 0 = Success, 1 = Error invalid data
	Height uint64 `json:"height"` // New height of the blockchain (number of blocks).
}

/*
apiBlockchainSelfAppend appends a block to the blockchain. This is a low-level function for already encoded blocks.
Do not use this function. Adding invalid data to the blockchain may corrupt it which might result in blacklisting by other peers.

Request:    POST /blockchain/self/append with JSON structure apiBlockchainBlockRaw
Response:   200 with JSON structure apiBlockchainBlockStatus
*/
func apiBlockchainSelfAppend(w http.ResponseWriter, r *http.Request) {
	var input apiBlockchainBlockRaw
	if err := apiDecodeJSON(w, r, &input); err != nil {
		return
	}

	var records []core.BlockRecordRaw

	for _, record := range input.Records {
		records = append(records, core.BlockRecordRaw{Type: record.Type, Data: record.Data})
	}

	newHeight, status := core.UserBlockchainAppend(records)

	apiEncodeJSON(w, r, apiBlockchainBlockStatus{Status: status, Height: newHeight})
}

type apiBlockchainBlock struct {
	Status            int                 `json:"status"`            // Status: 0 = Success, 1 = Error block not found, 2 = Error block encoding (indicates that the blockchain is corrupt)
	PeerID            string              `json:"peerid"`            // Peer ID hex encoded.
	LastBlockHash     []byte              `json:"lastblockhash"`     // Hash of the last block. Blake3.
	BlockchainVersion uint64              `json:"blockchainversion"` // Blockchain version
	Number            uint64              `json:"blocknumber"`       // Block number
	RecordsRaw        []apiBlockRecordRaw `json:"recordsraw"`        // Block records raw. Successfully decoded records are parsed into the below fields.
	Decoded           struct {
		User  apiBlockRecordUser   `json:"user"`  // User details
		Files []apiBlockRecordFile `json:"files"` // Files
	} `json:"decoded"`
}

// apiBlockRecordUser specifies user information
type apiBlockRecordUser struct {
	NameValid     bool   `json:"namevalid"`     // Whether the name is supplied in this structure.
	Name          string `json:"name"`          // Arbitrary name of the user.
	NameSanitized string `json:"namesanitized"` // Sanitized version of the name.
}

// apiBlockRecordFile is the metadata of a file published on the blockchain
type apiBlockRecordFile struct {
	Hash      []byte `json:"hash"`      // Blake3 hash of the file data
	Type      uint8  `json:"type"`      // Type (low-level)
	Format    uint16 `json:"format"`    // Format (high-level)
	Size      uint64 `json:"size"`      // Size of the file
	Directory string `json:"directory"` // Directory
	Name      string `json:"name"`      // File name
}

/*
apiBlockchainSelfRead reads a block and returns the decoded information.

Request:    GET /blockchain/self/read?block=[number]
Result:     200 with JSON structure apiBlockchainBlock
*/
func apiBlockchainSelfRead(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	blockN, err := strconv.Atoi(r.Form.Get("block"))
	if err != nil || blockN < 0 {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	block, status := core.UserBlockchainRead(uint64(blockN))
	result := apiBlockchainBlock{Status: status}

	if status == 0 {
		for _, record := range block.RecordsRaw {
			result.RecordsRaw = append(result.RecordsRaw, apiBlockRecordRaw{Type: record.Type, Data: record.Data})
		}

		result.PeerID = hex.EncodeToString(block.OwnerPublicKey.SerializeCompressed())

		if block.User.Valid {
			result.Decoded.User.NameValid = true
			result.Decoded.User.Name = block.User.Name
			result.Decoded.User.NameSanitized = block.User.NameS
		}

		for _, file := range block.Files {
			result.Decoded.Files = append(result.Decoded.Files, apiBlockRecordFile{Hash: file.Hash, Type: file.Type, Format: file.Format, Size: file.Size, Directory: file.Directory, Name: file.Name})
		}
	}

	apiEncodeJSON(w, r, result)
}

// apiBlockAddFiles contains the file metadata to add to the blockchain
type apiBlockAddFiles struct {
	Files []apiBlockRecordFile `json:"files"`
}

/*
apiBlockchainSelfAddFile adds a file with the provided information to the blockchain.

Request:    POST /blockchain/self/add/file with JSON structure apiBlockAddFiles
Response:   200 with JSON structure apiBlockchainBlockStatus
*/
func apiBlockchainSelfAddFile(w http.ResponseWriter, r *http.Request) {
	var input apiBlockAddFiles
	if err := apiDecodeJSON(w, r, &input); err != nil {
		return
	}

	var files []core.BlockRecordFile

	for _, file := range input.Files {
		files = append(files, core.BlockRecordFile{Hash: file.Hash, Type: file.Type, Format: file.Format, Size: file.Size, Directory: file.Directory, Name: file.Name})
	}

	newHeight, status := core.UserBlockchainAddFiles(files)

	apiEncodeJSON(w, r, apiBlockchainBlockStatus{Status: status, Height: newHeight})
}
