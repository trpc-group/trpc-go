# tRPC-Go 进程服务

## 服务端调用模式
```golang
type greeterServerImpl struct{}

func (s *greeterServerImpl) SayHello(ctx context.Context, req *pb.HelloRequest, rsp *pb.HelloReply) error {
	// implement business logic here ...
	// ...

	return nil
}

func main() {
	
	s := trpc.NewServer()
	
	pb.RegisterGreeterServer(s, &greeterServerImpl{})
	
	if err := s.Serve(); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
```

## 相关概念解析
 - server：代表一个服务实例，即 一个进程（只有一个 service 的 server 也可以当作 service 来处理）
 - service：代表一个逻辑服务，即 一个真正监听端口的对外服务，与配置文件的 service 一一对应，一个 server 可能包含多个 service，一个端口一个 service
 - proto service：代表一个协议描述服务，protobuf 协议文件里面定义的 service，通常 service 与 proto service 是一一对应的，也可由用户自己通过 Register 任意组合

## service 映射关系

 - 假如协议文件写了多个 service，如：
   ```pb
    service hello {
        rpc SayHello(Request) returns (Response) {};
    }
    service bye {
        rpc SayBye(Request) returns (Response) {};
    }
   ```
 - 配置文件也写了多个 service，如：
   ```yaml
   server:                                             #服务端配置
      app: test                                        #业务的应用名
      server: helloworld                               #进程服务名
      close_wait_time: 5000                            #关闭服务时的最小等待时间，用于等待服务反注册完成，单位 ms
      max_close_wait_time: 60000                       #关闭服务时的最大等待时间，用于等待请求处理完成，单位 ms
      service:                                         #业务服务提供的 service，可以有多个
        - name: trpc.test.helloworld.Greeter1          #service 的路由名称
          ip: 127.0.0.1                                #服务监听 ip 地址 可使用占位符 ${ip},ip 和 nic 二选一，优先 ip
          port: 8000                                   #服务监听端口 可使用占位符 ${port}
          protocol: trpc                               #应用层协议 trpc http
        - name: trpc.test.helloworld.Greeter2          #service 的路由名称
          ip: 127.0.0.1                                #服务监听 ip 地址 可使用占位符 ${ip},ip 和 nic 二选一，优先 ip
          port: 8080                                   #服务监听端口 可使用占位符 ${port}
          protocol: http                               #应用层协议 trpc http
   ```
 - 首先创建一个 server，svr := trpc.NewServer()，配置文件定义了多少个 service，就会启动多少个 service 逻辑服务
 - 组合方式：
  - 单个 proto service 注册到 server 里面：pb.RegisterHelloServer(svr, helloImpl) 这里会将协议文件内部的 hello server desc 注册到 server 内部的所有 service 里面
  - 单个 proto service 注册到 service 里面：pb.RegisterByeServer(svr.Service("trpc.test.helloworld.Greeter1"), byeImpl) 这里只会将协议文件内部的 bye server desc 注册到指定 service name 的 service 里面
  - 多个 proto service 注册到同一个 service 里面：pb.RegisterHelloServer(svr.Service("trpc.test.helloworld.Greeter1"), helloImpl) pb.RegisterByeServer(svr.Service("trpc.test.helloworld.Greeter1"), byeImpl)，这个 Greeter1 逻辑 service 同时支持处理不同的协议 service 处理函数

## 服务端执行流程
- 1. accept 一个新链接启动一个 goroutine 接收该链接数据
- 2. 收到一个完整数据包，解包整个请求
- 3. 查询 handler map，定位到具体处理函数
- 4. 解压请求 body
- 5. 设置消息整体超时
- 6. 反序列化请求 body
- 7. 调用前置拦截器
- 8. 调用业务处理函数
- 9. 调用后置拦截器
- 10. 序列化响应 body
- 11. 压缩响应 body
- 12. 打包整个响应
- 13. 回包给上游客户端
