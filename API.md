# API

Please see https://github.com/PeernetOfficial/core/webapi for the API specification.

This code extends the API by these function:

```
/console                    Console provides a websocket to send/receive internal commands
```


## Console

The `/console` websocket allows to execute internal commands. This should be only used for debugging purposes by the end-user. The same input and output as raw text as via the command-line is provided through this endpoint.

```
Request:    ws://127.0.0.1:112/console
```
