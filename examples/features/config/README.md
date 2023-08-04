## Config

The code demonstrates how to read business custom configuration.
In this example, the code specifically demonstrates how the server reads a custom configuration file and sends the parameters of the configuration file to the client as text.

## Usage

* Start server.

```shell
$ go run server/main.go -conf server/trpc_go.yaml
```

* Start client.

```shell
$ go run client/main.go
```

* Server output

```
Get config - custom : {{customConfigFromServer {value1 true 1234}}} 
test : customConfigFromServer 
key1 : value1 
key2 : true 
key2 : 1234 
2023-06-06 22:01:07.655 DEBUG   maxprocs/maxprocs.go:47 maxprocs: Leaving GOMAXPROCS=4: CPU quota undefined
2023-06-06 22:01:07.655 INFO    server/service.go:164   process:3694704, trpc service:trpc.examples.config.Config launch success, tcp:127.0.0.1:8000, serving ...
trpc-go-server SayHello, req.msg:trpc-go-client
trpc-go-server SayHello, rsp.msg:trpc-go-server response: Hello trpc-go-client. Custom config from server: customConfigFromServer
```

* Client output

```
Get msg: trpc-go-server response: Hello trpc-go-client. Custom config from server: customConfigFromServer
```
