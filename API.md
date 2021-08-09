# API

The API is intended to be used by a local full standalone client that provides a frontend for the end-user. The API allows to use Peernet effectively; it provides functions to share files, search for content and download files.

Note: This API code is likely to be moved into the core soon.

## Use considerations

It is not intended to be directly used in a legacy web browser, and shall not be exposed to the internet. It shall only run on a loopback IP such as `127.0.0.1` or `::1`. Special HTTP headers (including the Access-Control headers) are intentionally not set.

The API is currently unauthenticated and intentionally provides direct access to the users blockchain.

### Notes

The API is still in development and endpoints are subject to change. The API should be currently only used for debugging and early phase development purposes.

## Configuartion

The configuration file (default `Config.yaml`) contains settings for the API.

## Overview

These are the functions provided by the API:

```
/status                     Provides current connectivity status to the network
/peer/self                  Provides information about the self peer details
/blockchain/self/header     Header of the self peers blockchain
/blockchain/self/read       Read the self peers blockchain
/blockchain/self/append     Add a record to the blockchain
/share/list                 List all files and directories that are shared
/share/file                 Share a file via the peers blockchain

/search                     Search Peernet for a file based on keywords or hash
/download/start             Download a file based on a hash
/download/status            Get the status of a download

/status/ws                  Starts a websocket to receive updates on operations immediately (push instead of pull)
/console                    Console provides a websocket to send/receive internal commands
```

The `/share` functions are providing high-level functionality to work with files; the `/blockchain` functions provide low-level functionality.

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
}
```

Example response data: Todo

## Self Information

This function returns information about the current peer.

```
Request:    GET /peer/self

Response:   200 with JSON structure apiResponsePeerSelf
```

The peer and node IDs are returned as hex encoded strings.

```go
type apiResponsePeerSelf struct {
	PeerID string `json:"peerid"` // Peer ID. This is derived from the public in compressed form.
	NodeID string `json:"nodeid"` // Node ID. This is the blake3 hash of the peer ID and used in the DHT.
}
```

Example response data: Todo

## Console

The `/console` web-socket allows to execute internal commands. This should be only used for debugging purposes by the end-user. The same input and output as raw text as via the command-line is provided through this endpoint.

```
Request:    ws://127.0.0.1:112/console
```
