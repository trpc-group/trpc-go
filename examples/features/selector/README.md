## Selector

The code demonstrates how to use a custom router selector, which can be seen as a "map[service-name][]backend-ip", to select a backend service address based on the service name.
In this example, the backend service address is "127.0.0.1:8000" and the service name is "trpc.examples.selector.Selector".

## Usage

* Start server.

```shell
$ go run server/main.go -conf server/trpc_go.yaml
```

* Start client.

Searching for nodes using the target configured in the client/trpc_go.yaml file.
Of course, you can set the target through `client.WithTarget` in the code, which has a higher priority than YAML configuration.

```shell
$ go run client/main.go -conf client/trpc_go.yaml
```

* Server output

```
2023-05-25 16:39:45.765 DEBUG   common/common.go:21     recv req:msg:"trpc-go-client"
2023-05-25 16:39:45.766 DEBUG   common/common.go:39     SayHi recv req:msg:"trpc-go-client"
```

* Client output

```
2023-05-25 16:39:45.767 INFO    client/main.go:40       SayHello success rsp[msg:"Hello Hi trpc-go-client"]
```
