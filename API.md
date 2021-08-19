# API

The API is intended to be used by a local full standalone client that provides a frontend for the end-user. The API allows to use Peernet effectively; it provides functions to share files, search for content and download files.

Note: This API code is likely to be moved into the core soon.

## Use considerations

It is not intended to be directly used in a legacy web browser, and shall not be exposed to the internet. It shall only run on a loopback IP such as `127.0.0.1` or `::1`. Special HTTP headers (including the Access-Control headers) are intentionally not set.

The API is currently unauthenticated and intentionally provides direct access to the users blockchain.

### Notes

The API is still in development and endpoints are subject to change. The API should be currently only used for debugging and early phase development purposes.

## Configuration

The configuration file (default `Config.yaml`) contains settings for the API. To listen on `http://127.0.0.1:112/` add this line:

```yaml
APIListen: ["127.0.0.1:112"]
```

## Overview

These are the functions provided by the API:

```
/status                     Provides current connectivity status to the network
/peer/self                  Provides information about the self peer details
/share/list                 List all files and directories that are shared
/share/file                 Share a file via the peers blockchain

/search                     Search Peernet for a file based on keywords or hash
/download/start             Download a file based on a hash
/download/status            Get the status of a download

/status/ws                  Starts a websocket to receive updates on operations immediately (push instead of pull)
/console                    Console provides a websocket to send/receive internal commands

/blockchain/self/header     Header of the blockchain
/blockchain/self/append     Append a block to the blockchain
/blockchain/self/read       Read a block of the blockchain
/blockchain/self/add/file   Add file to the blockchain
/blockchain/self/list/file  List all files stored on the blockchain
```

The `/share` functions are providing high-level functionality to work with files. The `/blockchain` functions provide low-level functionality which is typically not needed.

## Status

This function informs about the current connection status of the client to the network. Additional fields will be added in the future.

```
Request:    GET /status

Response:   200 with JSON structure apiResponseStatus
```

```go
type apiResponseStatus struct {
	Status        int  `json:"status"`        // Status code: 0 = Ok.
	IsConnected   bool `json:"isconnected"`   // Whether connected to Peernet.
	CountPeerList int  `json:"countpeerlist"` // Count of peers in the peer list. Note that this contains peers that are considered inactive, but have not yet been removed from the list.
	CountNetwork  int  `json:"countnetwork"`  // Count of total peers in the network.
	// This is usually a higher number than CountPeerList, which just represents the current number of connected peers.
	// The CountNetwork number is going to be queried from root peers which may or may not have a limited view into the network.
}
```

## Self Information

This function returns information about the current peer.

```
Request:    GET /peer/self

Response:   200 with JSON structure apiResponsePeerSelf
```

The peer and node IDs are encoded as hex encoded strings.

```go
type apiResponsePeerSelf struct {
	PeerID string `json:"peerid"` // Peer ID. This is derived from the public in compressed form.
	NodeID string `json:"nodeid"` // Node ID. This is the blake3 hash of the peer ID and used in the DHT.
}
```

## Console

The `/console` websocket allows to execute internal commands. This should be only used for debugging purposes by the end-user. The same input and output as raw text as via the command-line is provided through this endpoint.

```
Request:    ws://127.0.0.1:112/console
```

## Blockchain Self Header

This function returns information about the current peer. It is not required that a peer has a blockchain. If no data is shared, there are no blocks. The blockchain does not formally have a header as each block has the same structure.

```
Request:    GET /blockchain/self/header

Response:   200 with JSON structure apiBlockchainHeader
```

```go
type apiBlockchainHeader struct {
	PeerID  string `json:"peerid"`  // Peer ID hex encoded.
	Version uint64 `json:"version"` // Current version number of the blockchain.
	Height  uint64 `json:"height"`  // Height of the blockchain (number of blocks). If 0, no data exists.
}
```

## Blockchain Append Block

This appends a block to the blockchain. This is a low-level function for already encoded blocks.
Do not use this function. Adding invalid data to the blockchain may corrupt it which subsequently might result in blacklisting by other peers.

```
Request:    POST /blockchain/self/append with JSON structure apiBlockchainBlockRaw

Response:   200 with JSON structure apiBlockchainBlockStatus
```

```go
type apiBlockRecordRaw struct {
	Type uint8  `json:"type"` // Record Type. See core.RecordTypeX.
	Data []byte `json:"data"` // Data according to the type.
}

type apiBlockchainBlockRaw struct {
	Records []apiBlockRecordRaw `json:"records"` // Block records in encoded raw format.
}

type apiBlockchainBlockStatus struct {
	Status  int    `json:"status"`  // Status: 0 = Success, 1 = Error invalid data
	Version uint64 `json:"version"` // Current version number of the blockchain.
	Height  uint64 `json:"height"`  // Height of the blockchain (number of blocks).
}
```

## Blockchain Read Block

This reads a block of the current peer.

```
Request:    GET /blockchain/self/read?block=[number]

Response:   200 with JSON structure apiBlockchainBlock
```

```go
type apiBlockchainBlock struct {
	Status            int                 `json:"status"`            // Status: 0 = Success, 1 = Error block not found, 2 = Error block encoding (indicates that the blockchain is corrupt)
	PeerID            string              `json:"peerid"`            // Peer ID hex encoded.
	LastBlockHash     []byte              `json:"lastblockhash"`     // Hash of the last block. Blake3.
	BlockchainVersion uint64              `json:"blockchainversion"` // Blockchain version
	Number            uint64              `json:"blocknumber"`       // Block number
	RecordsRaw        []apiBlockRecordRaw `json:"recordsraw"`        // Block records raw. Successfully decoded records are parsed into the below fields.
	decoded           struct {
		User  apiBlockRecordUser   `json:"user"`  // User details
		Files []apiBlockRecordFile `json:"files"` // Files
	} `json:"decoded"`
}

type apiBlockRecordUser struct {
	NameValid     bool   `json:"namevalid"`     // Whether the name is supplied in this structure.
	Name          string `json:"name"`          // Arbitrary name of the user.
	NameSanitized string `json:"namesanitized"` // Sanitized version of the name.
}

type apiBlockRecordFile struct {
	Hash      []byte // Hash of the file data
	Type      uint8  // Type (low-level)
	Format    uint16 // Format (high-level)
	Size      uint64 // Size of the file
	Directory string // Directory
	Name      string // File name
}
```

## Blockchain Add File

This adds a file with the provided information to the blockchain.

```
Request:    POST /blockchain/self/add/file with JSON structure apiBlockAddFiles

Response:   200 with JSON structure apiBlockchainBlockStatus
```

```go
type apiBlockAddFiles struct {
	Files []apiBlockRecordFile `json:"files"`
}

type apiBlockchainBlockStatus struct {
	Status int    `json:"status"` // Status: 0 = Success, 1 = Error invalid data
	Height uint64 `json:"height"` // New height of the blockchain (number of blocks).
}
```

Example POST request data to `http://127.0.0.1:112/blockchain/self/add/file`:

```json
{
    "files": [{
        "hash": "aFad3zRACbk44dsOw5sVGxYmz+Rqh8ORDcGJNqIz+Ss=",
        "type": 1,
        "format": 10,
        "size": 4,
        "name": "Test.txt",
        "directory": "sample directory/sub folder"
    }]
}
```

## Blockchain List Files

This lists all files stored on the blockchain.

```
Request:    GET /blockchain/self/list/file

Response:   200 with JSON structure apiBlockAddFiles
```

Example output:

```json
{
    "files": [{
        "hash": "aFad3zRACbk44dsOw5sVGxYmz+Rqh8ORDcGJNqIz+Ss=",
        "type": 1,
        "format": 10,
        "size": 4,
        "directory": "sample directory/sub folder",
        "name": "Test.txt"
    }],
    "status": 0
}
```
