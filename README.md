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

## Web API

The web API described in the [core library](https://github.com/PeernetOfficial/core/tree/master/webapi) is only available if the listen parameter is specified either via command line parameter or via the settings file.

As described in the linked specification, do not expose this API on the internet or local network, it allows sensitive operations such as deleting the private key and access to local files. It shall only be used by local clients on the same machine. Set the listen parameter only to a loopback IP address such as `::1`.

### Option 1: Command Line Parameter

Specify the `webapi` parameter with an IP:Port to listen:

```
Cmd -webapi=[::1]:1234
```

Multiple addresses can be specified by separating them with a comma:

```
Cmd -webapi=127.0.0.1:1337,[::1]:1234
```

Note that the command line parameter does not support the SSL and timeout settings. The API settings in the config file are ignored in case the command line parameter is specified.

### Option 2: Config File

In the `Config.yaml` specify the below line. The `APIListen` is a list of IP:Port pairs. IPv4 and IPv6 are supported. The SSL and timeout settings are optional. If the timeouts are not specified, they are not used. Valid units for the timeout settings are ms, s, m, h.

```yaml
APIListen: ["127.0.0.1:112","[::1]:112"]

# optional enable SSL
APIUseSSL:          true
APICertificateFile: "certificate.crt"   # Certificate received from the CA. This can also include the intermediate certificate from the CA.
APICertificateKey:  "certificate.key"   # Private Key

# optional timeouts
APITimeoutRead:     "10m"               # The maximum duration for reading the entire request, including the body. In this example 10 minutes.
APITimeoutWrite:    "10m"               # The maximum duration before timing out writes of the response. This includes processing time and is therefore the max time any HTTP function may take.
```
