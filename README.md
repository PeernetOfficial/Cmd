# Peernet Command Line Client

This is the command line client used for testing, debug and development purposes. It uses the [core library](https://github.com/PeernetOfficial/core). Check the core library for optional settings.

This client can be used as root peer to help speed up discovery of peers and data.

## Compile

To build:

```
go build
```

## Use

The config filename is hard-coded to `Config.yaml` and is created on the first run. Please see the core library for individual settings to change.

Simply start it and then use the listed commands:

```
C:\Peernet\Cmd>Cmd
Peernet Cmd 0.1
------------------------------
Please enter a command:
help               Show this help
net list           Lists all network adapters and their IPs
status             Get current status
chat               Send text to all peers
peer list          List current peers
debug key create   Create Public-Private Key pair
debug key self     List current Public-Private Key pair
```
