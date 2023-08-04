# MetaData
trpc-go supports transmission of metadata between the client and server, and automatically transmits them throughout the entire call chain.

Here, this example will show how you can transmit metadata between the client and server.
## Usage

* Start server.
```shell
$ go run server/main.go -conf server/trpc_go.yaml
```

* Start client.
```shell
$ go run client/client.go -conf client/trpc_go.yaml
```

The server log will be displayed as follows:
```shell
2023-05-10 14:30:12.148 DEBUG   server/main.go:33       SayHello recv req:msg:"trpc-go-client"
2023-05-10 14:30:12.148 DEBUG   server/main.go:38       SayHello get key: say-hello-client, value: hello
2023-05-10 14:30:12.149 DEBUG   server/main.go:52       SayHi recv req:msg:"trpc-go-client"
2023-05-10 14:30:12.149 DEBUG   server/main.go:58       SayHi get key: say-hi-client, value: hi
2023-05-10 14:30:12.149 DEBUG   server/main.go:58       SayHi get key: key1, value: val1
2023-05-10 14:30:12.149 DEBUG   server/main.go:58       SayHi get key: key2, value: val2
```

The client log will be displayed as follows:
```shell
2023-05-10 14:30:12.149 DEBUG   client/main.go:34       say hello trans info: key: say-hello-server, val: hello
2023-05-10 14:30:12.149 DEBUG   client/main.go:44       say hi trans info: key: say-hi-server, val: hi
```

### MetaData in Server

* Get MetaData

You can get `MetaData` from `ctx` which is passed by the framework.
```go
msg := codec.Message(ctx)
md := msg.ServerMetaData()
```

Also, if you want to get a value by a specified key, you can just use `trpc.GetMetaData` to get it.
```go
// GetMetaData returns metadata from ctx by key.
func GetMetaData(ctx context.Context, key string) []byte {
	msg := codec.Message(ctx)
	if len(msg.ServerMetaData()) > 0 {
		return msg.ServerMetaData()[key]
	}
	return nil
}
```
* Set MetaData

To set metadata that is returned to the client, you can use `trcp.SetMetaData`.
```go
trpc.SetMetaData("key", []byte("val"))
```


### MetaData in Client

* Set MetaData

In a client, you can set metadata by adding options with a key-value pair and multiple key-value pairs can be added.

```go
opts := []client.Option{
    client.WithMetaData("key1", []byte("val1")),
    client.WithMetaData("key2", []byte("val2")),
}
```

* Get MetaData

The upstream client can get metadata by setting `trpc.ResponseProtocol` when sending a request.
```go
head := &trpc.ResponseProtocol{}
opts := []client.Option{
    client.WithRspHead(head),
}
rsp, err := proxy.SayHello(ctx, opts...)
for key, val := range head.TransInfo {
	// ...
}
```



### Difference between ServerMetaData and ClientMetaData
In trpc-go framework, `ServerMetaData` is the transmitted data parsed from business protocol in server. And `ClientMetaData` is set to business protocol by client when it sends a request to backend. 

In this example, we set `ClientMetaData` when we call a request, but the server receives it as `ServerMetaData`. They are the same data.

The ClientCodec will set `ClientMetaData` to `TransInfo` when it encodes request protocol.
```go
// Set tracing MetaData.
if len(msg.ClientMetaData()) > 0 {
	req.TransInfo = make(map[string][]byte)
	for k, v := range msg.ClientMetaData() {
		req.TransInfo[k] = v
	}
}
```

`TransInfo` is defined as follows：
```protobuf
	// Key-value pairs transmitted by framework, are divided two part:
	// 1.Framework information which key is started with "trpc-".
	// 2.Business information which key can be set arbitrarily.
	TransInfo map[string][]byte `protobuf:"bytes,9,rep,name=trans_info,json=transInfo,proto3" json:"trans_info,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
```

The ServerCodec will obtain metadata from `TransInfo` and put it into `ServerMetaData` when it decodes request protocol.
```go
if len(req.TransInfo) > 0 {
	msg.WithServerMetaData(req.GetTransInfo())
	
	// Dyeing.
	if bs, ok := req.TransInfo[DyeingKey]; ok {
		msg.WithDyeingKey(string(bs))
	}
}
```

