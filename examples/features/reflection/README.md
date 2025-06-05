# 服务端反射

本文档介绍如何使用服务端反射，具体包括两个方面：

1. 在 server 端开启反射功能
2. 在 client 端利用服务端反射发起一个普通的 trpc 调用

## 在 server 端开启反射功能

通常只建议在测试环境启用该功能，在正式环境中使用可能会有安全隐患。
启动该功能需要同时修改配置文件和代码。

### 修改配置文件

1. 在 server.service 字段中添加一个 trpc service。
2. 在 server.reflection_service 字段中指定第 1 步中添加的 service 为反射 service。

例如，为当前 server 添加一个名为 trpc.reflection.v1.ServerReflection 的反射 service。

```yaml
server:
  reflection_service: trpc.reflection.v1.ServerReflection # 指定反射 service，和下面的 service.name 保持一致。
  service:
    - name: trpc.reflection.v1.ServerReflection # 服务名，一般对应北极星上面的名字。
      ip: 127.0.0.1
      nic: eth0
      port: 8002
      network: tcp # 必须指定为 tcp
      protocol: trpc # 必须指定为 trpc
```

### 修改代码

在代码中匿名引入 reflection 包即可。

```go
import  _ "git.code.oa.com/trpc-go/trpc-go/reflection"
```

### 启动 Server

```bash
cd server
go run .
```

终端会输出一行 WARN 日志，显示当前 server 已经启用了反射功能：

```yaml
WARN    reflection/server.go:48 The server reflection feature is being enabled. Please note that this feature is typically only available in the testing environment, and using it in the production environment may cause security issues.
```

## 在 client 端利用服务端反射发起一个普通的 trpc 调用

存在两种方法：

- 使用 trpc-cli 工具
- 根据服务端反射提供服务接口，调用客户端的桩代码 (git.woa.com/trpc/trpc-protocol/pb/go/trpc/reflection)。

### trpc-cli 工具

安装 v4.0.0 版本[（暂未开发完成）](https://git.woa.com/trpc-go/trpc-cli/issues/66)以上的 [trpc-cli](https://git.woa.com/trpc-go/trpc-cli/blob/master/README.zh_CN.md) 工具。
可以通过 [trpc-cli 北极星名字寻址](https://git.woa.com/trpc-go/trpc-cli/blob/master/README.zh_CN.md#%E5%8C%97%E6%9E%81%E6%98%9F%E5%90%8D%E5%AD%97%E5%AF%BB%E5%9D%80) 来进行相关调用。
不过在本例子里面为了能够在本地运行，采用[指定-ipport-发请求](https://git.woa.com/trpc-go/trpc-cli/blob/master/README.zh_CN.md#%E6%8C%87%E5%AE%9A-ipport-%E5%8F%91%E8%AF%B7%E6%B1%82)进行示范。

关于 trpc-cli 工具的指定的参数说明：

- -callee 参数指定北极星上面的服务名（注意使用 -servicename 参数是无效的），采用北极星寻址；
- -target 参数指定 ip:port，采用 ip:port 寻址。
- -func 参数指定 pb 里面的方法名，用于发起 rpc 调用。
- -describe 参数指定 pb 文件里面的符号名，用于获取各种 pb 符号的具体描述。  

#### 列出所有 services 的接口服务名和路由服务名

- 输入：

```bash
./trpc-cli -listservice -target=ip://127.0.0.1:8002
```

- 输出：

```text
[service]:
  0. routing name:trpc.examples.echo.EchoYYY, interface name:trpc.examples.echo.Echo
  1. routing name:trpc.reflection.v1.ServerReflection, interface name:trpc.reflection.v1.ServerReflection
  2. routing name:trpc.test.helloworld.GreeterXXX, interface name:trpc.test.helloworld.Greeter
[node]:service:trpc.reflection.v1.ServerReflection, addr:21.6.100.33:8002, cost:666.32µs
[err]:<nil>
```

routing name 通常是北极星上面的名字，interface name 是 pb 里面的名字格式为<package>.<service>

#### 使用 service 的 interface name 获取该 service 在 pb 中的详细信息

- 输入：

```bash
./trpc-cli -target=ip://127.0.0.1:8002 -describe=trpc.examples.echo.Echo
```

- 输出：

    ```text
    trpc.examples.echo.Echo is a service:
    service Echo {
      rpc BidirectionalStreamingEcho ( stream .trpc.examples.echo.EchoRequest ) returns ( stream .trpc.examples.echo.EchoResponse );
      rpc ClientStreamingEcho ( stream .trpc.examples.echo.EchoRequest ) returns ( .trpc.examples.echo.EchoResponse );
      rpc ServerStreamingEcho ( .trpc.examples.echo.EchoRequest ) returns ( stream .trpc.examples.echo.EchoResponse );
      rpc UnaryEcho ( .trpc.examples.echo.EchoRequest ) returns ( .trpc.examples.echo.EchoResponse );
    }
    ```

###### 描述消息

描述请求/响应消息，需要提供完整的在 pb 中的类型名称（格式为`-message="<package>.<type>, <package>.<type>"`）。

- 输入：

```bash
./trpc-cli -target=ip://127.0.0.1:8002 -describe="trpc.examples.echo.EchoRequest, trpc.examples.echo.EchoResponse"
```

- 输出：

    ```text
    trpc.examples.echo.EchoRequest is a message:
    message EchoRequest {
      string message = 1;
    }
    
    trpc.examples.echo.EchoResponse is a message:
    message EchoResponse {
      string message = 1;
    }
    ```

#### 发起普通 rpc

- 输入：

```bash
./trpc-cli  -target=ip://127.0.0.1:8001 -func=/trpc.examples.echo.Echo/UnaryEcho -body='{"message":"hello"}'
```

- 输出：

    ```text
    [req head]:request_id:1  timeout:999  caller:"trpc.client.trpc-cli.service"  callee:"trpc.examples.echo.Echo"  func:"/trpc.examples.echo.Echo/UnaryEcho"  trans_info:{key:"traceparent"  value:"00-550032dfe632701179d56d14af39738b-88eb8246cafe3706-01"}  content_type:2
    [req json body]:{"message":"hello"}
    [rsp head]:request_id:1  trans_info:{key:"traceparent"  value:"00-550032dfe632701179d56d14af39738b-88eb8246cafe3706-01"}  content_type:2
    [rsp json body]:{"message":"hello"}
    [node]:service:127.0.0.1:8001, addr:127.0.0.1:8001, cost:1.541708ms
    [err]:<nil>
    ```

### 根据服务端反射提供服务接口，调用客户端的桩代码

参考 v4.0.0 版本（暂未开发完成）的相关代码实现。
