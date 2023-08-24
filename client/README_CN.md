# tRPC-Go 后端调用

## 客户端调用模式

```golang
proxy := pb.NewGreeterClientProxy()
rsp, err := proxy.SayHello(ctx, req)
if err != nil {
	log.Errorf("say hello fail:%v", err)
	return err
}
return nil
```

## 相关概念解析
- proxy 客户端调用桩函数或者调用代理，由 trpc 工具自动生成，内部调用 client，proxy 是个很轻量的结构，内部不会创建链接，每次请求每次生成即可
- target 后端服务的地址，规则为 selectorname://endpoint，默认使用北极星+servicename，一般不需要设置，用于自测或者兼容老寻址方式如 l5 cmlb ons 等
- config 后端服务的配置，框架提供 client.RegisterConfig 注册配置能力，这样每次调用后端就可以自动从配置读取后端访问参数，不需要用户自己设置，需要业务自己解析配置文件并注册进来，默认以被调方协议文件的 servicename(package.service) 为 key 获取相关配置
- WithReqHead 一般情况，用户不需要关心协议头，都在框架底层做了，但在跨协议调用时就需要用户自己设置请求包头
- WithRspHead 设置响应包头承载结构，回传协议响应头，一般用于接入转发层

## 后端配置管理
- 后端配置一般与环境相关，每个环境都有自己的独立配置，包括下游的 rpc 地址和数据库 db 地址
- 应该使用远程配置中心，配置更新会自动生效，无需重启，也可以自己灰度控制
- 没有配置中心的情况下才使用 trpc_go.yaml 里面的 client 配置区块，常用于自测过程
  远程配置样例如下：
```yaml
client:                                    # 客户端调用的后端配置
  timeout: 1000                            # 针对所有后端的请求最长处理时间
  namespace: Development                   # 针对所有后端的环境
  filter:                                  # 针对所有后端的拦截器配置数组
    - m007                                 # 所有后端接口请求都上报 007 监控
  service:                                 # 针对单个后端的配置
    - callee: trpc.test.helloworld.Greeter # 后端服务协议文件的 service name, 如果 callee 和下面的 name 一样，那只需要配置一个即可
      name: trpc.test.helloworld.Greeter1  # 后端服务名字路由的 service name，有注册到名字服务的话，下面 target 不用配置
      target: ip://127.0.0.1:8000          # 后端服务地址，ip://ip:port polaris://servicename cl5://sid cmlb://appid ons://zkname
      network: tcp                         # 后端服务的网络类型 tcp udp
      protocol: trpc                       # 应用层协议 trpc http
      timeout: 800                         # 当前这个请求最长处理时间
      serialization: 0                     # 序列化方式 0-pb 2-json 3-flatbuffer，默认不要配置
      compression: 1                       # 压缩方式 1-gzip 2-snappy 3-zlib，默认不要配置
      filter:                              # 针对单个后端的拦截器配置数组
        - tjg                              # 只有当前这个后端上报 tjg
```

配置中的 `callee` 和 `name` 的区别：

* `callee` 是指被调方的 pb 协议文件的 service name，格式是 `pbpackage.service`。

如 pb 为：
```protobuf
package trpc.a.b;
service Greeter {
    rpc SayHello(request) returns reply
}
```
那么 `callee` 即为 `trpc.a.b.Greeter`

* `name` 是指被调方注册在名字服务（如北极星）上面的服务名，也就是被调服务的 trpc_go.yaml 里面的 `server.service.name` 的配置值。

一般情况下，`callee` 和 `name` 是相同的，只需配置其中任何一个即可，但是有些场景下，如存储服务，同一份 pb 会部署多个实例，这个时候的名字服务的 service name 和 pb service name 就不一样了，此时配置文件就必须同时配置 `callee` 和 `name`

```yaml
client:
  service:
    - callee: pbpackage.service  # 必须同时配置 callee 和 name，callee 是 pb 的 service name，用于匹配 client proxy 和配置
      name: polaris-service-name # 北极星名字服务的 service name，用于寻址
      protocol: trpc
```
通过 pb 生成的 client 桩代码，默认会把 pb servicename 填入到 client 中，所以 client 寻找配置时只会 `以 callee 为 key`（也就是 pb 的 service name）来匹配

而通过类似 `redis.NewClientProxy("trpc.a.b.c")` 等（包括 database 下面所有插件以及 http）生成的 client，默认 service name 就是用户自己输入的字符串，所以 client 寻找配置时**以 NewClientProxy 的输入参数为 key**（即以上的 `trpc.a.b.c`）来匹配

在 v0.10.0 之后，支持了同时以 `callee` 及 `name` 为 key 来寻找配置，比如以下两个客户端配置共享了相同的 `callee`:

```yaml
client:
  service:
    - callee: pbpackage.service  # 必须同时配置 callee 和 name，callee 是 pb 的 service name，用于匹配 client proxy 和配置
      name: polaris-service-name1 # 北极星名字服务的 service name，用于寻址
      protocol: trpc
    - callee: pbpackage.service  # 必须同时配置 callee 和 name，callee 是 pb 的 service name，用于匹配 client proxy 和配置
      name: polaris-service-name2 # 北极星名字服务的 service name，用于寻址
      protocol: trpc
```

用户在代码中可以使用 `client.WithServiceName` 来同时用 `called` 以及 `name` 作为 key 进行配置的寻找：

```golang
// proxy1 使用第一项配置
proxy1 := pb.NewClientProxy(client.WithServiceName("polaris-service-name1"))
// proxy2 使用第二项配置
proxy2 := pb.NewClientProxy(client.WithServiceName("polaris-service-name2"))
```

在 < v0.10.0 的版本中，上述写法都只会找到第二项配置 (存在 `callee` 相同的配置时，后面的会覆盖前面的)

## 客户端调用流程
- 1. 设置相关配置参数
- 2. 通过 target 解析 selector，为空则使用框架自己的寻址流程
- 3. 通过 selector 获取节点信息
- 4. 通过节点信息加载节点配置
- 5. 调用前置拦截器
- 6. 序列化 body
- 7. 压缩 body
- 8. 打包整个请求
- 9. 调用网络请求
- 10. 解包整个响应
- 11. 解压 body
- 12. 反序列化 body
- 13. 调用后置拦截器
