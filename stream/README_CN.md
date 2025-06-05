# tRPC-Go 流式

## 服务端调用模式

```protobuf
syntax = "proto3";
package pb;
// The greeting service definition.
service Greeter {
    // Sends a greeting
    rpc SayHello (stream HelloRequest) returns (HelloReply) {}
}
// The request message containing the user's name.
message HelloRequest {
    string name = 1;
}
// The response message containing the greetings
message HelloReply {
    string message = 1;
}

```

```go
// SayHello 客户端流式，SayHello 传入 pb.Greeter_SayHelloServer 作为参数，返回 error
// pb.Greeter_SayHelloServer 提供 Recv() 和 SendAndClose() 等接口，用作流式交互
func (s *greeterServerImpl) SayHello(gs pb.Greeter_SayHelloServer) error {
    var names []string
    for {
        // 服务端使用 for 循环进行 Recv，接收来自客户的数据
        in, err := gs.Recv()
        // 如果返回 EOF，说明客户端流已经结束，客户端已经发送完所有数据
        if err == io.EOF {
            log.Infof("receive error io eof %v\n", err)
            // SendAndClose 发送并关闭流
            gs.SendAndClose(&pb.HelloReply{Message: "hello " + strings.Join(names, ",")})
            return nil
        }
        // 说明流发生异常，需要返回
        if err != nil {
            log.Errorf("receive from %v\n", err)
            return err
        }
        names = append(names, in.Name)
    }
}
```
