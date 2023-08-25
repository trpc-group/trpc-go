# tRPC-Go Reverse Proxy

## Introduction

In some special scenarios, such as reverse proxy forwarding services, where fully transitive binary body data is required without serializing (and deserializing) requests (and responses) to improve forwarding performance, tRPC-Go provides support for these scenarios as well by providing custom serialization methods.

## Implementation

### Pass-through Server

Pass-through server takes out the binary body directly to the handler function when it receives the request, without deserialization, and packs the binary body directly to the upstream when it returns the packet, without serialization.

#### Custom stub code

Since there is no serialization and deserialization process, i.e. no pb protocol file, users need to write their own service stub code and processing functions.
The key point is to use `codec.Body` (or implement BytesBodyIn BytesBodyOut interface, see [here](https://git.woa.com/trpc-go/trpc-go/blob/v0.8.5/codec/serialization_noop.go#L20)) to pass through the binary, use `wildcards*` for forwarding, and execute the interceptors by yourself.

```go
type AccessServer interface {
	Forward(ctx context.Context, reqbody *codec.Body, rspbody *codec.Body) (err error)
}

// AccessServer_Forward_Handler is a message handling callback function. 
func AccessServer_Forward_Handler(svr interface{}, ctx context.Context, f server.FilterFunc) (rspbody interface{}, err error) {
	req := &codec.Body{}
	rsp := &codec.Body{}
	filters, err := f(req)
	if err ! = nil {
		return nil, err
	}
	handleFunc := func(ctx context.Context, reqbody interface{}, rspbody interface{}) error {
		return svr.(AccessServer).Forward(ctx, reqbody.(*codec.Body), rspbody.(*codec.Body))
	}
	err = filters.Handle(ctx, req, rsp, handleFunc)
	if err ! = nil {
		return nil, err
	}
	return rsp, nil
}



// AccessServer_ServiceDesc describes service description information, and use wildcards * for forwarding.
var AccessServer_ServiceDesc = server.ServiceDesc{
	ServiceName: "trpc.kandian.oidb_trpc_proxy.Access",
	HandlerType: ((*AccessServer)(nil)), 
	Methods: []server.Mehod{
		server.Method{
			Name: "*", 
			Func: AccessServer_Forward_Handler,
		},
	},
}

// RegisterAccessService registers the service.
func RegisterAccessService(s server.Service, svr AccessServer) {
	s.Register(&AccessServer_ServiceDesc, svr)
}
```

#### Specifying the empty serialization method

After defining the stub code, you can implement the handler function and start the service. 
The key point is to pass `WithCurrentSerializationType(codec.SerializationTypeNoop)` when you are calling `NewServer` to tell the framework that the current message is only transmitted and not serialized.

```go
type AccessServerImpl struct{}

// Forward implements forwarding proxy logic
func (s *AccessServerImpl) Forward(ctx context.Context, reqbody *codec.Body, rspbody *codec.Body) (err error) {
	// Your own internal processing logic
}

func main() {
	s := trpc.NewServer(
		server.WithCurrentSerializationType(codec.SerializationTypeNoop).
	) // No serialization
	
	RegisterAccessService(s, &AccessServerImpl{})

	if err := s.Serve(); err ! = nil { 
		panic(err) 
	} 
}
```

### Pass-through Client

Pass-through Client packages and sends out the binary body directly when the rpc request is made to the downstream, without serialization, and is returned directly after the packet is returned, without deserialization.

#### Specifying the empty serialization method

Although the current framework is not serialized, but still need to tell the downstream the current binary has been packaged by what serialization method, because the downstream need to parse through this serialization method, so you should set `WithSerializationType` ` WithCurrentSerializationType` these two options.

```go
ctx, msg := codec.WithCloneMessage(ctx) // copy a ctx, generate caller callee and other information, easy to monitor the framework to report
msg.WithClientRPCName("/trpc.test.helloworld.Greeter/SayHello") // set the downstream method name
msg.WithCalleeServiceName("trpc.test.helloworld.Greeter") // set the downstream service name
callopts := []client.Option{
	client.WithProtocol("trpc").
	client.WithSerializationType(codec.SerializationTypePB), // tell downstream that the current body has been serialized with pb
	WithCurrentSerializationType(codec.SerializationTypeNoop), // tells the framework that the current client is only transitive and not serialized
}

req := &codec.Body{Data: []byte("I am a binary data that has been packaged by other serialization methods")}
rsp := &codec.Body{} // After the packetization, the framework will automatically fill this rsp.Data with binary data
DefaultClient.Invoke(ctx, req, rsp, callopts...) // req rsp is the binary data that the user has already serialized himself
if err ! = nil {
	return err
}
```

## Examples

### oidb_trpc_proxy

oidb_trpc_proxy is a proxy service supported by Tencent Watchpoint for stock compatibility, which *provides oidb protocols services externally* and *forwards trpc protocol backwards*, *with body fully transmissive*, to achieve seamless access to trpc for HandQ clients and old services upstream. The code is available [here](https://git.woa.com/tkd/proxy/oidb-trpc-proxy).


## FAQ

### Q1: SerializationType and CurrentSerializationType what do these two options mean and what is the difference

tRPC-Go provids `SerializationType` and `CurrentSerializationType` to support proxy forwarding.

SerializationType is mainly used for context passing of network calls, and CurrentSerializationType is mainly used for current framework data parsing.
`SerializationType` refers to the original serialization method of the body, which is normally specified inside the protocol field. 
tRPC default serialization type is pb.
`CurrentSerializationType` refers to the framework received data, the real way to perform serialization operations, generally do not need to fill in, and the default is equal to SerializationType.
CurrentSerializationType allows users to set their own arbitrary serialization method, In above reverse proxy example CurrentSerializationType is set to [NoopSerializationType](https://git.woa.com/trpc-go/trpc-go/blob/master/codec/serialization_noop.go).