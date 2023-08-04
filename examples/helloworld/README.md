## trpc-go helloworld 工程示例

## 业务服务开发步骤：
1. 每个服务单独创建一个 git，如：github.com/your_repo/helloworld
2. 初始化 go mod 文件：go mod init github.com/your_repo/helloworld
3. 编写服务协议文件，如：helloworld.proto, 协议规范如下： 
* 3.1 package 分成三级 trpc.app.server, app 是一个业务项目分类，server 是具体的进程服务名
* 3.2 必须指定 option go_package，表明协议的 git 地址
* 3.2 定义 service rpc 方法，一个 server 可以有多个 service，一般都是一个 server 一个 service
```golang
syntax = "proto3";

package trpc.test.helloworld;
option go_package="github.com/your_repo/protocol/helloworld";

service Greeter {
    rpc SayHello (HelloRequest) returns (HelloReply) {}
    rpc SayHi (HelloRequest) returns (HelloReply) {}
}

message HelloRequest {
    string msg = 1;
}

message HelloReply {
    string msg = 1;
}

```  
4. 通过命令行生成服务模型：trpc create --protofile=helloworld.proto（首先需要先[安装 trpc 工具](https://trpc.group/trpc-go/trpc-go-cmdline)）,
可以在 `trpc_go.yaml` 的 server service 中额外添加 HTTP RPC 服务:
```yaml
    - name: trpc.test.helloworld.Greeter  # service 的名字服务路由名称
      ip: 127.0.0.1                       # 服务监听 ip 地址
      port: 8080                          # 服务监听端口
      network: tcp                        # 网络监听类型 tcp udp
      protocol: http                      # 应用层协议 trpc http
      timeout: 1000                       # 请求最长处理时间 单位 毫秒
```
5. 开发具体业务逻辑
6. 开发完成，开始编译，根目录执行：go build
7. 执行单元测试：go test -v
8. 启动服务：./helloworld &
9. 自测 trpc 协议：trpc-cli -func "/trpc.test.helloworld.Greeter/SayHello" -target "ip://127.0.0.1:8000" -body '{"msg":"hello"}' -v
10. 自测 http 协议：curl -X POST -d '{"msg":"hello"}' -H "Content-Type:application/json" http://127.0.0.1:8080/trpc.test.helloworld.Greeter/SayHello
