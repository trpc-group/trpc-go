# tRPC-Go Streaming

## Server call mode

```protobuf
syntax = "proto3";
package pb;
// The greeting service definition.
service Greeter {
  // Sends a greeting.
  rpc SayHello (stream HelloRequest) returns (HelloReply) {}
}
// The request message containing the user's name.
message HelloRequest {
  string name = 1;
}
// The response message containing the greetings.
message HelloReply {
  string message = 1;
}

```

```golang
// SayHello client stream implementation, SayHello passes in pb.Greeter_SayHelloServer as a parameter, returns error.
// pb.Greeter_SayHelloServer provides interfaces such as Recv() and SendAndClose() for streaming interaction.
func (s *greeterServerImpl) SayHello(gs pb.Greeter_SayHelloServer) error {
	var names []string
	for {
		// The server uses a for loop for Recv to receive data from the client.
		in, err := gs.Recv()
		// If EOF is returned, the client stream has ended and the client has sent all data.
		if err == io.EOF {
			log.Infof("recveive error io eof %v\n", err)
			// SendAndClose sends and closes the stream.
			gs.SendAndClose(&pb.HelloReply{Message: "hello " + strings.Join(names, ",")})
			return nil
		}
		// Indicates that the stream has an exception and needs to return.
		if err != nil {
			log.Errorf("receive from %v\n", err)
			return err
		}
		names = append(names, in.Name)
	}
}
```
