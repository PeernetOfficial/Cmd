# Peernet Command Line Client

This is the command line client used for testing, debug and development purposes. It uses the [core library](https://github.com/PeernetOfficial/core). Check the core library for optional settings.

This client can be used as root peer to help speed up discovery of peers and data.

## Compile

Download the [latest version of Go](https://golang.org/dl/). To build:

```
go build
```

To reduce the binary size provide the linker switch `-s` which will "Omit the symbol table and debug information". This reduced the Windows binary by 26% in a test. The flag `-trimpath` removes "all file system paths from the resulting executable" which could be considered an information leak.

```
go build -trimpath -ldflags "-s"
```

### Windows Headless Version

To build a headless version (like a typical Windows GUI application) that does not show the command line window use the linker switch `-H=windowsgui`.

```
go build -trimpath -ldflags "-H=windowsgui -s"
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

The web API described in the [core library](https://github.com/PeernetOfficial/core/tree/master/webapi#web-api) is only available if the listen parameter is specified either via command line parameter or via the settings file.

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

## API Functions

All API functions provided by the core library are described [here](https://github.com/PeernetOfficial/core/tree/master/webapi#available-functions).

In addition, this application also provides these functions:

```
/console                        Websocket to send/receive internal commands
/shutdown                       Graceful shutdown
```

### Console

This provides a websocket to send/receive internal commands in the same way provided via the command line interface. The websocket messages sent to the API are the input texts from the end-user, and the messages received from the API are the text outputs.

This can be useful as internal debug interface in clients.

```
Request:    GET /console
Result:     Upgrade to websocket. The websocket message are texts to read/write.
```

### Shutdown

This gracefully shuts down the application. Actions: 0 = Shutdown.

```
Request:    GET /shutdown?action=[action]
Result:     200 with JSON structure apiShutdownStatus
```

```go
type apiShutdownStatus struct {
	Status int `json:"status"` // Status of the API call. 0 = Success.
}
```

Example request: `http://127.0.0.1:112/shutdown?action=0`

Example response:

```json
{
    "status": 0
}
```

## Error Handling

The application exits in case of the errors listed below and uses the specified exit code. Applications that launch this application can monitor for those exit codes. End users should look into the log file for additional information in case any of these errors occur, although some of them are pre log file initialization.

| Exit Code  | Constant               | Info                                                |
| ---------- | ---------------------- | --------------------------------------------------- |
| 0          | ExitSuccess            | This is actually never used.                        |
| 1          | ExitErrorConfigAccess  | Error accessing the config file.                    |
| 2          | ExitErrorConfigRead    | Error reading the config file.                      |
| 3          | ExitErrorConfigParse   | Error parsing the config file.                      |
| 4          | ExitErrorLogInit       | Error initializing log file.                        |
| 5          | ExitParamWebapiInvalid | Parameter for webapi is invalid.                    |
| 6          | ExitPrivateKeyCorrupt  | Private key is corrupt.                             |
| 7          | ExitPrivateKeyCreate   | Cannot create a new private key.                    |
| 8          | ExitBlockchainCorrupt  | Blockchain is corrupt.                              |
| 9          | ExitGraceful           | Graceful shutdown.                                  |
| 0xC000013A | STATUS_CONTROL_C_EXIT  | The application terminated as a result of a CTRL+C. |
