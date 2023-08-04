## compression demo

### start the server

```shell
$ go run server/main.go -conf server/trpc_go.yaml 
```

### start the client

#### 1. send request with compress type `gzip`

```shell
$ go run client/main.go -conf client/trpc_go.yaml -type "gzip"
```

The server log will be displayed as follows:

```shell
2023-05-23 19:39:11.629 DEBUG   common/common.go:21     recv req:msg:"trpc-go-client"
2023-05-23 19:39:11.631 DEBUG   common/common.go:39     SayHi recv req:msg:"trpc-go-client"
```

The client log will be displayed as follows:

```shell
2023-05-23 19:39:11.627 DEBUG   client/main.go:39       request with compressType : gzip
2023-05-23 19:39:11.632 INFO    client/main.go:56       reply is: msg:"Hello Hi trpc-go-client"
```

#### 2. send request with compress type `snappy`

```shell
$ go run client/main.go -conf client/trpc_go.yaml -type "snappy"
```

The server log will be displayed as follows:

```shell
2023-05-23 19:39:11.629 DEBUG   common/common.go:21     recv req:msg:"trpc-go-client"
2023-05-23 19:39:11.631 DEBUG   common/common.go:39     SayHi recv req:msg:"trpc-go-client"
```

The client log will be displayed as follows:

```shell
2023-05-23 19:39:11.627 DEBUG   client/main.go:39       request with compressType : snappy
2023-05-23 19:39:11.632 INFO    client/main.go:56       reply is: msg:"Hello Hi trpc-go-client"
```

#### 3. send request with compress type `zlib`

```shell
$ go run client/main.go -conf client/trpc_go.yaml -type "zlib"
```

The server log will be displayed as follows:

```shell
2023-05-23 19:39:11.629 DEBUG   common/common.go:21     recv req:msg:"trpc-go-client"
2023-05-23 19:39:11.631 DEBUG   common/common.go:39     SayHi recv req:msg:"trpc-go-client"
```

The client log will be displayed as follows:

```shell
2023-05-23 19:39:11.627 DEBUG   client/main.go:39       request with compressType : zlib
2023-05-23 19:39:11.632 INFO    client/main.go:56       reply is: msg:"Hello Hi trpc-go-client"
```

#### 4. send request with compress type `streamSnappy`

```shell
$ go run client/main.go -conf client/trpc_go.yaml -type "streamSnappy"
```

The server log will be displayed as follows:

```shell
2023-05-23 19:39:11.629 DEBUG   common/common.go:21     recv req:msg:"trpc-go-client"
2023-05-23 19:39:11.631 DEBUG   common/common.go:39     SayHi recv req:msg:"trpc-go-client"
```

The client log will be displayed as follows:

```shell
2023-05-23 19:39:11.627 DEBUG   client/main.go:39       request with compressType : streamSnappy
2023-05-23 19:39:11.632 INFO    client/main.go:56       reply is: msg:"Hello Hi trpc-go-client"
```

#### 5. send request with compress type `blockSnappy`

```shell
$ go run client/main.go -conf client/trpc_go.yaml -type "blockSnappy"
```

The server log will be displayed as follows:

```shell
2023-05-23 19:39:11.629 DEBUG   common/common.go:21     recv req:msg:"trpc-go-client"
2023-05-23 19:39:11.631 DEBUG   common/common.go:39     SayHi recv req:msg:"trpc-go-client"
```

The client log will be displayed as follows:

```shell
2023-05-23 19:39:11.627 DEBUG   client/main.go:39       request with compressType : blockSnappy
2023-05-23 19:39:11.632 INFO    client/main.go:56       reply is: msg:"Hello Hi trpc-go-client"
```


### use `rpcz` to check the `RequestSize`

```shell
$ curl http://ip:port/cmds/rpcz/spans?num=2

1:
  span: (server, 2710336014210592128)
    time: (May 23 20:13:22.130331, May 23 20:13:22.130817)
    duration: (0, 486.091µs, 0)
    attributes: (RequestSize, 150),(ResponseSize, 63),(RPCName, /trpc.test.helloworld.Greeter/SayHello),(Error, success)
2:
  span: (server, 3356845803395109080)
    time: (May 23 20:13:22.130616, May 23 20:13:22.130685)
    duration: (0, 68.264µs, 0)
    attributes: (RequestSize, 134),(ResponseSize, 37),(RPCName, /trpc.test.helloworld.Greeter/SayHi),(Error, success)
```



