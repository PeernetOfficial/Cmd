# Peernet Command Line Client

This is the command line client used for testing, debug and development purposes. It uses the [core library](https://github.com/PeernetOfficial/core). Check the core library for optional settings.

This client can be used as root peer to help speed up discovery of peers and data.

## Compile

First get all the dependencies. Below list contains both dependencies from the core package and this tool.

```
go get -u github.com/PeernetOfficial/core
go get -u github.com/gofrs/uuid
go get -u github.com/btcsuite/btcd/btcec
go get -u github.com/libp2p/go-reuseport
go get -u lukechampine.com/blake3
```

To build:

```
go build
```
