# Peernet Command Line Client

This is the test client.

## Todos

* The Peernet core should be a separate library.
* Windows: Auto-configure firewall via Set-NetFirewallRule https://docs.microsoft.com/en-us/powershell/module/netsecurity/set-netfirewallrule?view=win10-ps (might require elevated rights)
* Windows: Can non-admin open necessary ports?
* Support for UPnP. See example https://github.com/tendermint/go-p2p/blob/master/listener.go
* Check TURN code https://github.com/pion/turn
* Check STUN code https://github.com/ccding/go-stun
* Identify errors that are OS specific, such as in AutoAssignPort

## Encryption and Hashing functions

* Salsa20 is used for encrypting the packets.
* secp256k1 is used to generate the peer IDs (public keys).

## Dependencies

Before compiling, make sure to download and update all 3rd party packages:

```
go get -u github.com/gofrs/uuid
go get -u github.com/btcsuite/btcd/btcec
```

## Settings

* `ListenWorkers` defines the count of concurrent workers processing packets (decrypting them and then taking action). Default 2.
* `Listen` defines IP or IP:Port combinations to listen on. If not specified, it will listen on 0.0.0.0 and ::1. If an IP is specified and no port, it will try to listen on port 112, then port 12912, then up to 99 ports after that.
