# Stream
trpc-go supports stream RPC，with stream RPC, the client and server can establish a continuous connection to send and receive data continuously, allowing the server to provide continuous responses.

Here, this example will show how you can use stream RPC between the client and server.
## Usage

* Start server.
```shell
$ go run server/main.go -conf server/trpc_go.yaml 
```

* Start ClientStream client.
```shell
$ go run client/main.go -conf client/trpc_go.yaml -type "ClientStream"
```

The ClientStream server log will be displayed as follows:
```shell
2023-05-19 10:13:43.806 INFO    server/main.go:57       ClientStream receive Msg: ping : 0
2023-05-19 10:13:43.806 INFO    server/main.go:57       ClientStream receive Msg: ping : 1
2023-05-19 10:13:43.806 INFO    server/main.go:57       ClientStream receive Msg: ping : 2
2023-05-19 10:13:43.806 INFO    server/main.go:57       ClientStream receive Msg: ping : 3
2023-05-19 10:13:43.806 INFO    server/main.go:57       ClientStream receive Msg: ping : 4
2023-05-19 10:13:43.806 INFO    server/main.go:47       ClientStream receive EOF, then close receive and send pong message
```

The ClientStream client log will be displayed as follows:
```shell
2023-05-19 10:13:43.806 INFO    client/main.go:85       ClientStream reply message is: pong
```

* Start ServerStream client.
```shell
$ go run client/main.go -conf client/trpc_go.yaml  -type "ServerStream"
```

The ServerStream server log will be displayed as follows:
```shell
2023-05-19 10:14:34.082 INFO    server/main.go:65       ServerStream receive Msg: ping
```

The ServerStream client log will be displayed as follows:
```shell
2023-05-19 10:14:34.082 INFO    client/main.go:108      ServerStream reply message is:  pong: 0
2023-05-19 10:14:34.082 INFO    client/main.go:108      ServerStream reply message is:  pong: 1
2023-05-19 10:14:34.082 INFO    client/main.go:108      ServerStream reply message is:  pong: 2
2023-05-19 10:14:34.082 INFO    client/main.go:108      ServerStream reply message is:  pong: 3
2023-05-19 10:14:34.082 INFO    client/main.go:108      ServerStream reply message is:  pong: 4
```


* Start BidirectionalStream client.
```shell
$ go run client/main.go -conf client/trpc_go.yaml -type "BidirectionalStream"
```

The BidirectionalStream server log will be displayed as follows:
```shell
2023-05-19 10:15:26.359 INFO    server/main.go:93       BidirectionalStream receive Msg: ping: 0
2023-05-19 10:15:26.359 INFO    server/main.go:93       BidirectionalStream receive Msg: ping: 1
2023-05-19 10:15:26.359 INFO    server/main.go:93       BidirectionalStream receive Msg: ping: 2
2023-05-19 10:15:26.359 INFO    server/main.go:93       BidirectionalStream receive Msg: ping: 3
2023-05-19 10:15:26.359 INFO    server/main.go:93       BidirectionalStream receive Msg: ping: 4
2023-05-19 10:15:26.359 INFO    server/main.go:85       BidirectionalStream EOF error, then close receive
```

The BidirectionalStream client log will be displayed as follows:
```shell
2023-05-19 10:15:26.359 INFO    client/main.go:147      BidirectionalStream reply message is: pong: :ping: 0
2023-05-19 10:15:26.359 INFO    client/main.go:147      BidirectionalStream reply message is: pong: :ping: 1
2023-05-19 10:15:26.359 INFO    client/main.go:147      BidirectionalStream reply message is: pong: :ping: 2
2023-05-19 10:15:26.359 INFO    client/main.go:147      BidirectionalStream reply message is: pong: :ping: 3
2023-05-19 10:15:26.360 INFO    client/main.go:147      BidirectionalStream reply message is: pong: :ping: 4
2023-05-19 10:15:26.360 INFO    client/main.go:140      BidirectionalStream EOF error, then close receive
```
