[English](./README.md)| 中文
# tRPC-Go Server 模块


## 背景

一个服务进程可能会监听多个端口，在不同端口上提供不同的业务服务。因此 server 模块提出了服务实例(Server)、逻辑服务(Service)和协议服务(proto service)的概念。通常一个进程包含一个服务实例，每个服务实例可以包含一个或者多个逻辑服务。逻辑服务用于名字注册，客户端会使用逻辑服务名进行路由寻址发起网络请求，服务端接收到请求后根据指定的协议服务执行服务端的业务逻辑。

- `Server`：代表一个服务实例，即一个进程
- `Service`：代表一个逻辑服务，即一个真正监听端口的对外服务，与配置文件中的 service 一一对应，一个 server 可能包含多个 service，一个端口一个 service
- `Proto service`：代表一个协议服务，protobuf 协议文件里面定义的 service，通常 service 与 proto service 是一一对应的，也可由用户自己通过 Register 任意组合

```golang
// Server is a tRPC server. One process, one server.
// A server may offer one or more services.
type Server struct {
    MaxCloseWaitTime time.Duration
}

// Service is the interface that provides services.
type Service interface {
    // Register registers a proto service.
    Register(serviceDesc interface{}, serviceImpl interface{}) error
    // Serve starts serving.
    Serve() error
    // Close stops serving.
    Close(chan struct{}) error
}
```

## service 映射关系

假如协议文件中提供 hello service，如：

```protobuf
service hello {
    rpc SayHello(HelloRequest) returns (HelloReply) {};
}
```

配置文件写了多个 service，分别提供 trpc 和 http 协议服务如：

```yaml
server: # 服务端配置
  app: test # 业务的应用名
  server: helloworld # 进程服务名
  close_wait_time: 5000 # 关闭服务时的最小等待时间，用于等待服务反注册完成，单位 ms
  max_close_wait_time: 60000 # 关闭服务时的最大等待时间，用于等待请求处理完成，单位 ms
  service: # 业务服务提供两个 service，监听不同的端口提供不同协议的服务
    - name: trpc.test.helloworld.HelloTrpc # 第一个 service 的路由名称
      ip: 127.0.0.1 # 服务监听 ip 地址
      port: 8000 # 服务监听端口 8000
      protocol: trpc # 提供 trpc 协议服务
    - name: trpc.test.helloworld.HelloHttp # 另一个 service 的路由名称
      ip: 127.0.0.1 # 服务监听 ip 地址
      port: 8080 # 监听端口 8080
      protocol: http # 提供 http 协议服务
```

为不同的逻辑服务注册协议服务

```golang
type helloImpl struct{}

func (s *helloImpl) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
    rsp := &pb.HelloReply{}
    // implement business logic here ...
    return rsp, nil
}

func main() {
    s := trpc.NewServer()

    // 推荐：为每个 service 单独注册 proto service
    pb.RegisterHiServer(s.Service("trpc.test.helloworld.HelloTrpc"), helloImpl)
    pb.RegisterHiServer(s.Service("trpc.test.helloworld.HelloHttp"), helloImpl)

    // 第二种方式，为 server 中的所有 service 注册同一个 proto service
    pb.RegisterHelloServer(s, helloImpl)
}
```

## 服务端执行流程

1. 网络层 Accept 到一个新连接启动一个协程处理该连接的数据
2. 收到一个完整数据包，解包整个请求
3. 查询根据具体的 proto service 名，定位到具体处理函数
4. 解码请求体
5. 设置消息整体超时
6. 解压缩，反序列化请求体
7. 调用前置拦截器
8. 进入业务处理函数
9. 退出业务处理函数
10. 调用后置拦截器
11. 序列化，压缩响应体
12. 打包整个响应
13. 回包给上游客户端
