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

## Config

The config filename is hard-coded to `Config.yaml` and is created on the first run. Please see the [core library](https://github.com/PeernetOfficial/core#configuration) for individual settings to change.

The config contains the locations of important files and folders.

```yaml
LogFile:          "data/log.txt"                # Log file. It contains informational and error messages.
BlockchainMain:   "data/blockchain main/"       # Blockchain main stores the end-users blockchain data. It contains meta data of shared files, profile data, and social interactions.
BlockchainGlobal: "data/blockchain global/"     # Blockchain global stores blockchain data from global users.
WarehouseMain:    "data/warehouse main/"        # Warehouse main stores the actual data of files shared by the end-user.
```

## Web API

The web API described in the [core library](https://github.com/PeernetOfficial/core/tree/master/webapi#web-api) is only available if the listen parameter is specified either via command line parameter or via the settings file.

As described in the linked specification, do not expose this API on the internet or local network, it allows sensitive operations such as deleting the private key and access to local files. It shall only be used by local clients on the same machine. Set the listen parameter only to a loopback IP address such as `::1`.

API key authentication enforces the `x-api-key` header in each API request.

### Option 1: Command Line Parameter

Specify the `webapi` parameter with an IP:Port to listen and a random API key (UUID) in `apikey`:

```
Cmd -webapi=[::1]:1234 -apikey=a30c01eb-856c-4b79-bdde-3c56a248f71b
```

Multiple addresses can be specified by separating them with a comma:

```
Cmd -webapi=127.0.0.1:1337,[::1]:1234 -apikey=a30c01eb-856c-4b79-bdde-3c56a248f71b
```

Note that the command line parameter does not support the SSL and timeout settings. The API settings in the config file are ignored in case the command line parameter is specified.

### Option 2: Config File

In the `Config.yaml` specify the below line. The `APIListen` is a list of IP:Port pairs. IPv4 and IPv6 are supported. The SSL and timeout settings are optional. If the timeouts are not specified, they are not used. Valid units for the timeout settings are ms, s, m, h.

API key authentication can be disabled by specifying a null UUID (= `00000000-0000-0000-0000-000000000000`) in the config which may be useful for development purposes, but should never be disabled in production.

```yaml
APIListen:          ["127.0.0.1:112","[::1]:112"]
APIKey:             "a30c01eb-856c-4b79-bdde-3c56a248f71b"

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
| 10         | ExitParamApiKeyInvalid | API key parameter is invalid.                       |
| 0xC000013A | STATUS_CONTROL_C_EXIT  | The application terminated as a result of a CTRL+C. |

## Windows User Privileges

This application is supposed to run and connect to Peernet on Windows regardless if admin/non-admin and elevated/non-elevated.

The user running the application must have write access to the relevant local files (including the config, log, warehouse folders, blockchain folders).

## Firewall

It is advised to configure any firewalls to explicitly allow any traffic to the application. Users with NATs (typically with home routers) may manually enable port forwarding, although this application supports UPnP.

The Windows Firewall can be configured via the below command-line command (the path needs to be adjusted). Such rule will take "precedence over the default block setting" according to [Microsoft documentation](https://docs.microsoft.com/en-us/windows/security/threat-protection/windows-firewall/best-practices-configuring). This action requires elevated admin rights.

```
netsh advfirewall firewall add rule name="Peernet Cmd" dir=in program="C:\Users\User\Desktop\Peernet\Cmd.exe" profile=any action=allow
```

If such a rule is not set, Windows will open the dialogue "Windows Defender Firewall has blocked some features of this app" (Windows 11). The "Public networks" setting is checked by default, but the "Private networks" is not. Confirming the firewall exception requires admin rights, otherwise Windows create a "block" firewall entry.

* Windows 11: Listening on the default UDP port 112 or a random one (if 112 is already used) is itself not restricted for non-admin users. However, the Windows Firewall may block incoming traffic for non-admin users (see below).

### Impact to firewalled users

As non-admin user, the "Allow access" button opens the User Account Control window which requires an admin login. Failure to provide an admin login (or hiting the Cancel button) creates two inbound "block" rules (one for UDP, another for TCP) in the Windows Firewall based on the executable path for the public profile. The rules block both UDP/TCP traffic on all local/remote ports for any local/remote IP address. The Edge traversal setting is set to "Block edge traversal" which has the description "Prevent applications from receiving unsolicited traffic from the Internet through a NAT edge device".

This effectively means that Windows is likely block incoming traffic from uncontacted peers. This can be overcome via UDP hole punching.

In the current implementation this turned out to be a problem for users with non-admin rights on a server which do not use NAT (the public IP is directly assigned to the network adapter); in that case the reported internal and measured external ports are matching, and peers do not flag that peer as NAT (the N flag will not be set in the peer table output).

A potential fix for that would be a new self-reporting Firewall flag indicating that the peer believes it is behind a firewall and that the Traverse message (UDP hole punching) is required for connections from unknown peers.

### Windows Firewall Debugging

The Windows Firewall supports enabling a log file (by default the setting is disabled). This [article](https://www.howtogeek.com/220204/how-to-track-firewall-activity-with-the-windows-firewall-log/) shows how to enable the log. Following is a real log entry generated by the firewall when an external peer fails to contact the local peer due to the Windows Firewall:

| date time           | action | protocol | src-ip          | dst-ip          | src-port | dst-port | size | path    | pid  |
| ------------------- | ------ | -------- | --------------- | --------------- | -------- | -------- | ---- | ------- | ---- |
| 2021-11-16 00:49:08 | DROP   | UDP      | [IPv6 redacted] | [IPv6 redacted] | 112      | 112      | 143  | RECEIVE | 8924 |

### Expert Users

Expert user may manually edit the network listen config settings. Beware that invalid settings will negatively impact connectivity and might result in blacklisting by other peers. Make sure to manually configure any firewalls (including OS firewalls and network devices) according to your settings; only incoming and outgoing UDP traffic on the specified listening port is required for Peernet.
* Static public IP assigned to your network adapter (usually only servers): Set the `Listen` to the fixed public IP:Port. `EnableUPnP` must be false and `PortForward` must be 0.
* Dynamic public IP in home network but manual port forwarding on your router: Set `Listen` to your internal IP:Port from your network adapter that provides the connection to your router. Set `PortForward` to the external port. `EnableUPnP` must be false.
* For any other case, don't touch the default config!

## Integration as Backend

This application can be used as backend for a Peernet client.

### Process Exit Monitor

Use the parameter `-watchpid=[PID]` to specify a process ID to monitor for exit to automatically exit the application.

```
Cmd -watchpid=1234
```
