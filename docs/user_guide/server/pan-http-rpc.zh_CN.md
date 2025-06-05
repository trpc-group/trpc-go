# 1 前言

tRPC 是一个实现远程过程调用（RPC）的开发框架。对于 RPC 的实现，框架除了提供基于 tRPC 私有协议的实现外，同时也提供了基于 http，https，http2，http3 等协议的实现。通过本文的介绍，旨在为用户提供如何搭建基于 **“泛 HTTP 协议”** 的 RPC 服务，并帮助用户梳理以下问题：

- 什么是泛 HTTP 协议？
- 如何理解 RPC 和 HTTP 的关系？
- 如何理解泛 HTTP RPC 服务和泛 HTTP 标准服务的区别？
- 如何理解 tRPC 服务和泛 HTTP RPC 服务的区别？
- 如何设置泛 HTTP RPC 服务的底层协议？
- 如何搭建一个泛 HTTP RPC 服务服务？

tRPC-Go 从 v0.19.0 后支持 fasthttp 搭建泛 HTTP RPC 标准服务，[使用 fasthttp 搭建泛 HTTP RPC 服务](#8-基于-fasthttp-搭建泛-http-rpc-服务)。

在设计上，tfasthttp 在行为和用法上尽可能地与 thttp 保持一致，但由于各种原因（主要是 `net/http` 与 `fasthttp` 带来的不一致），其用法可能兼容性较差。

本文主要从如何使用出发，指导用户快速上手 fasthttp，关于细节，请用户查看 [tfasthttp 使用指南](https://doc.weixin.qq.com/doc/w3_Ac0AYwanAIUfx1rVLYYTm2A4u2oHj?scode=AJEAIQdfAAowr0OpC7Ac0AYwanAIU&version=4.1.28.6010&platform=win)。

# 2 概念介绍

## 2.1 什么是泛 HTTP 协议

在 tRPC-Go 的文档中，我们用 **泛 HTTP RPC 服务**来描述 RPC 服务的底层协议为 http，https，http2 和 http3 等协议的 RPC 服务。我们把这几种协议归为一类的原因在于：它们都是使用 http 语义的协议，在服务创建和调用上基本一致，唯一的区别仅在于“naming service”的“protocol”配置不一样。

## 2.2 什么是泛 HTTP RPC

**RPC** 是一种服务接口实现技术，它和 **RESTful** 是两种最常见的 API 设计模式，而 HTTP 是一种通信协议，用于承载服务通信数据。RPC 可以通过私有协议来实现，也能通过 HTTP 通用协议来实现。

**泛 HTTP RPC** 特指通过泛 HTTP 协议来实现的 RPC 服务，框架通过在 HTTP Head 添加 RPC 控制字段来实现 RPC 服务调用控制。协议细节都在框架内部封装了，对用户来说是透明的，所以不管底层使用什么协议，用户在使用上是没有变化的。

## 2.3 与泛 HTTP 标准服务的区别

泛 HTTP RPC 服务是 RPC 服务，服务调用接口由 PB 文件定义，可以由工具生产桩代码。而泛 HTTP 标准服务是一个普通的 HTTP 服务，不使用 PB 文件定义服务，用户需要自己编写代码定义服务接口，注册 URL，组包和解包 HTTP 报文。

# 3 泛 HTTP RPC 协议实现

框架默认使用 tRPC 协议来实现 RPC 调用模型的。tRPC 协议分为 "Head" 和 "Body" 两部分。Head 部分用于提供 RPC 调用的控制信息，包括：协议版本，请求 ID，调用类型，超时控制，染色等，关于控制字段可以参考 [tRPC 协议](https://iwiki.woa.com/pages/viewpage.action?pageId=145446228) 文档。Body 部分用于提供接口业务数据，字段由业务在定义服务时决定的。

泛 HTTP 协议都是采用 HTTP 语义，分成 Head 和 Body 两部分，可以无缝兼容 RPC 的设计模型，只需要把 tRPC 协议的 Head 字段映射到 HTTP 的 Head 上来，Body 部分则不需要改变。通过两者在 Head 字段上的一一映射，就达到了 RPC 功能在泛 HTTP 协议上的实现。

## 3.1 RPC 方法名映射

RPC 方法名在 tRPC 协议中对应的字段为 ["func 字段"](https://git.woa.com/trpc/trpc-protocol/blob/v0.2.1/trpc/proto/trpc.proto#L445)，在泛 HTTP 协议中对应于**“URL”**。在 [接口规范](https://iwiki.woa.com/pages/viewpage.action?pageId=99485634#接口规范) 中，RPC 方法的命名格式为：**“/package.service/method”**，映射到泛 HTTP 协议，URL 命名格式默认为：`http://ip:port/package.service/method`, 其中“ip:port”为服务对外提供服务的地址，可以使用域名

**注意**：此处的 `/package.service/method` 以桩代码中定义的标识为准，比如 [trpc-go/testdata/helloworld.trpc.go#L60](https://git.woa.com/trpc-go/trpc-go/blob/v0.18.3/testdata/helloworld.trpc.go#L60) 中：

```go
var GreeterServer_ServiceDesc = server.ServiceDesc{
    ServiceName: "trpc.test.helloworld.Greeter",
    HandlerType: ((*GreeterService)(nil)),
    Methods: []server.Method{
        {
            Name: "/trpc.test.helloworld.Greeter/SayHello", // <= 以此处的字符串作为 `/package.server/method` 
            Func: GreeterService_SayHello_Handler,
        },
        {
            Name: "/v1/hello", // <= alias 形式，同样可以作为访问路径以替代 `/package.server/method` 
            Func: GreeterService_SayHello_Handler,
        },
        // ... 
    },
}
```

不要使用 `trpc_go.yaml` 框架配置中服务端配置的 `name` （这个 `name` 用于服务注册，不一定和桩代码中使用的标识完全相同）。

此外，如果存在 `alias`，也同样以桩代码中的 `alias` 标识为准。

## 3.2 请求报文头映射

我们通过把 tRPC 协议的 RPC[请求控制字段](https://git.woa.com/trpc/trpc-protocol/blob/v0.2.1/trpc/proto/trpc.proto#L418) 映射到 HTTP 请求报文头里来实现基于泛 HTTP 协议的 RPC 调用。控制字段的映射关系如下表：
> 为了防止命名冲突，除“Content-Type”和“Content-Encoding”外，所有控制字段统一加上了 **“trpc-”** 前缀）

| 泛 HTTP 协议 Head 字段 | tRPC 协议包头字段      | 字段解释                                                         |
|-------------------|------------------|--------------------------------------------------------------|
| trpc-version      | version          | tRPC 协议版本                                                    |
| trpc-call-type    | call_type        | 请求的调用类型，比如：普通调用，单向调用                                         |
| trpc-request-id   | request_id       | 请求唯一 id                                                      |
| trpc-timeout      | timeout          | 请求的超时时间，单位 ms                                                |
| trpc-caller       | caller           | 主调服务的名称，trpc 协议下的规范格式：trpc. 应用名。服务名。pb 的 service 名，4 段       |
| trpc-callee       | callee           | 被调服务的路由名称，trpc 协议下的规范格式，trpc. 应用名。服务名。pb 的 service 名 [. 接口名] |
| trpc-message-type | message_type     | 框架信息透传的消息类型比如调用链、染色 key、灰度、鉴权、多环境、set 名称等的标识                 |
| trpc-trans-info   | trans_info       | 框架透传的信息 key-value 对                                          |
| Content-Type      | content_type     | 请求数据的序列化类型，比如：proto/jce/json 等                               |
| Content-Encoding  | content_encoding | 请求数据使用的压缩方式，比如：gzip/snappy/..., 默认不使用                        |

## 3.3 协议透传字段

对于**trpc-trans-info** 透传字段，其格式需要遵循以下原则：

- 字段格式为：key-value json 字符串经过 Base64 编码后的字符。框架在解析时，将该 json 字符串解析出来，逐个字段设置到 trpc trans_info map 里面
- 在 trpc-trans-info 中增加了一个 user ip 字段，携带客户端地址，字段名为“trpc-user-ip”

**示例**：如 http header 的 trpc-trans-info 字段内容为 "{\"key1\": \"value1\", \"key2\": \"value2\"}"，则 trpc trans_info map 内容为 {"key1": "value1", "key2": "value2", "trpc-user-ip": "58.100.19.11"}，每个 value 都需要经过 base64 编码

## 3.4 序列化字段

**HTTP GET 请求**

对于 HTTP GET 请求，框架不处理“Content-Type”字段，直接使用内置序列化方式来解析 GET URL 后缀参数。比如 URL 请求为：`http://ip:port/package.service/method?k1=v1&k2=v2`, 后缀参数字符串：k1=v1&k2=v2，框架会将 v1，v2 的值解析到 pb 文件里面的 key 为 k1，k2 的字段。

**HTTP POST 请求**

对于“Content-Type”字段，序列化类型和 tRPC 协议的映射关系如下：

| Content-Type                      | tRPC 协议 content_type 字段 |
|-----------------------------------|-------------------------|
| application/proto                 | protobuf（值为 0）          |
| application/jce                   | jce（值为 1）               |
| application/json                  | jsonpb（值为 2）            |
| application/flatbuffer            | flatbuffer（值为 3）        |
| application/octet-stream          | noop（二进制流，不序列化，值为 4）    |
| application/x-www-form-urlencoded | http-form（值为 129）       |

对于 HTTP POST 请求，框架通过泛 HTTP Head 中的“Content-Type”字段来选择具体的序列化方式解析 body，所以发起 http 调用时，客户端务必要填写正确的 Content-Type 来指定 HTTP Body 的数据类型。

**响应报文**

在服务给上游回包时，默认打包序列化方式和请求时的“Content-Type”保持一致，如 POST 请求是 form 格式，那么回包也是 form 格式，如果需要返回 json 格式，需要自行设置。设置方法如下：

```go
msg := trpc.Message(ctx)
msg.WithSerializationType(codec.SerializationTypeJSON)
```

用户可以自定义序列化类型，具体描述请参考 [7.1 自定义序列化类型](#7.1 自定义序列化类型)

**设置仅支持 HTTP POST 请求**

在 HTTP RPC 服务中，GET/POST 请求都是可以接受的，假如只希望用户通过 POST 方法进行请求，可以设置 `thttp.ServerCodec` 的 `POSTOnly` 字段（要求版本 >= v0.16.0）

```go
// 更改所有 protocol: http 的服务只接收 POST 请求
thttp.DefaultServerCodec.POSTOnly = true
```

此时当使用 GET 方法发送请求时，发送方会收到 "400 Bad Request" 的错误码，并在 "trpc-error-msg" header 中看到如下错误信息："service codec Decode: server codec only allows POST method request, the current method is GET"


## 3.5 解压缩字段

对于“Content-Encoding”字段，压缩方式和 tRPC 协议的映射关系为：

| Content-Encoding | tRPC 协议 content_encoding 字段 |
|:----------------:|:---------------------------:|
|       gzip       |         gzip（值为：1）          |

用户可以自定义解压缩方式，具体描述请参考 [7.2 自定义解压缩方式](#7.2 自定义解压缩方式)

## 3.6 响应报文头映射

我们通过把 tRPC 协议的 RPC[响应控制字段](https://git.woa.com/trpc/trpc-protocol/blob/v0.2.1/trpc/proto/trpc.proto#L469) 映射到 HTTP 响应报文头里来实现基于泛 HTTP 协议的 RPC 调用。控制字段的映射关系如下表：
> 为了防止命名冲突，除“Content-Type”和“Content-Encoding”外，所有控制字段统一加上了 **“trpc-”** 前缀）

| 泛 HTTP 响应报文头      | trpc 响应包头字段      | 字段解释                                         |
|-------------------|------------------|----------------------------------------------|
| trpc-version      | version          | tRPC 协议版本                                    |
| trpc-call-type    | call_type        | 请求的调用类型，比如：普通调用，单向调用                         |
| trpc-request-id   | request_id       | 请求唯一 id                                      |
| trpc-ret          | ret              | 请求在框架层的错误返回码                                 |
| trpc-func-ret     | func_ret         | 接口的错误返回码                                     |
| trpc-error-msg    | error_msg        | 调用结果信息描述                                     |
| trpc-message-type | message_type     | 框架信息透传的消息类型比如调用链、染色 key、灰度、鉴权、多环境、set 名称等的标识 |
| trpc-trans-info   | trans_info       | 框架透传的信息 key-value 对                          |
| Content-Type      | content_type     | 请求数据的序列化类型，比如：proto/jce/json 等               |
| Content-Encoding  | content_encoding | 请求数据使用的压缩方式，比如：gzip/snappy/..., 默认不使用        |

框架定义了 RPC 请求的错误字段和错误处理逻辑。用户可以自定义错误字段和错误处理逻辑，具体请参考 [7.3 自定义错误处理函数](#7.3 自定义错误处理函数) 。

# 4 Proto Service 定义

## 4.1 定义接口文件

对于泛 HTTP RPC 服务的定义和 tRPC 服务的定义方式完全一样，都遵循 pb v3 版本的标准规范。服务定义示例如下：

```proto
syntax = "proto3";
package trpc.test.rpchttp;
option go_package="git.woa.com/trpcprotocol/test/rpchttp"; // 把这个路径定义为你自己可以控制的仓库路径

service Hello {
    rpc SayHello (HelloRequest) returns (HelloReply) {}
}

// 请求参数
message HelloRequest {
    string msg = 1;
}
// 响应参数
message HelloReply {
    string msg = 1;
}
```

在这个服务中，package 为 trpc.test.rpchttp，proto service 为 Hello，方法名为 SayHello，假设使用 http 协议，所以 rpc name 对应的 URL 为：`http://host:port/trpc.test.rpchttp.Hello/SayHello`（需要自行替换 host 和 port）

## 4.2 自定义接口 URL

如果业务需要使用其它 URL 命名格式，trpc 工具提供了 methodoption 和注解两种方式来实现。（如果 PB 文件是通过 rick 平台来管理，目前需要采用注释法）, 更加详细介绍请参考 [trpc-go-cmdline 工具](https://iwiki.woa.com/pages/viewpage.action?pageId=278972980 "trpc-go-cmdline 工具")

- 方法一：为 rpc 指定 methodoption option (trpc.alias) = "/cgi-bin/hello"（**注意：必须要 import "trpc.proto"文件（由于历史原因，rick 平台需要 `import "trpc/common/trpc.proto";`）**）

```proto
syntax = "proto3";
package trpc.test.rpchttp;
option go_package="git.woa.com/trpcprotocol/test/rpchttp";

import "trpc.proto";

service Hello {
    rpc SayHello (HelloRequest) returns (HelloReply) {option (trpc.alias) = "/cgi-bin/hello";};
}

// 请求参数
message HelloRequest {
    string msg = 1;
}
// 响应参数
message HelloReply {
    string msg = 1;
}
```

- 方法二：为 rpc 添加注释 //@alias=/cgi-bin/hello，leadingComments、trailingComments 均可（此方法主要兼容存量代码，推荐使用上面的方法一）

```proto
syntax = "proto3";
package trpc.test.rpchttp;
option go_package="git.woa.com/trpcprotocol/test/rpchttp";

service Hello {
    //@alias=/cgi-bin/hello
    rpc SayHello (HelloRequest) returns (HelloReply);
}

// 请求参数
message HelloRequest {
    string msg = 1;
}
// 响应参数
message HelloReply {
    string msg = 1;
}
```

在使用方法二时，生成桩代码时需要添加命令参数“--alias”，命令如下：

```shell
trpc create --protofile hello.proto --alias --protocol http
```

这样接口的 URL 就设置成 `http://127.0.0.1:8000/cgi-bin/hello` 了。

## 4.3 自定义字段 Json 别名

系统支持为参数字段设置 json 别名，也就是在 json 格式发送 HTTP Body 时，字段名称使用 json 别名。

```proto
message HelloRequest {
    string msg = 1  [json_name="xmsg"];
}
```

同时需要修改框架中默认 JSON 序列化对象“OrigName”为 `false`

```go
import (
    "git.code.oa.com/trpc-go/trpc-go/codec"
)

func main() {
    codec.Marshaler.OrigName = false
}
```

这样，在 HTTP 请求报文 Body 中使用的字段是“xmsg”。需要注意的是，在业务代码中使用的“HelloRequest”仍然为“msg”字段。

## 4.4 生成桩代码

通过 tRPC 工具生成 tRPC-Go 桩代码和默认“trpc_go.yaml”框架配置文件：

``` shell
trpc create --protofile=helloworld.proto --protocol http
```

可运行的代码示例见：<https://git.woa.com/trpc-go/trpc-go/tree/master/examples/features/httprpc>

# 5 Naming Service 定义

对于泛 HTTP RPC 服务，我可以在“trpc_go.yaml”框架配置文件中通过“protocol”字段来指定具体协议类型。

## 5.1 作为 http 服务

我们可以通过设置“protocol”为“http”即可启动一个 http 服务。

```yaml
...
server:                                            # 服务端配置
  service:                                         # 业务服务提供的 service，可以有多个
    - name: trpc.test.rpchttp.Hello                # service 的路由名称
      ip: 127.0.0.1                                # 服务监听 ip 地址 可使用占位符 ${ip},ip 和 nic 二选一，优先 ip
      port: 8000                                   # 服务监听端口 可使用占位符 ${port}
      network: tcp                                 # 网络监听类型  tcp udp
      protocol: http                               # 应用层协议 trpc http 
      timeout: 1000                                # 请求最长处理时间 单位 毫秒
```

## 5.2 作为 https 服务

我们可以通过设置 `protocol` 为 `http`, 并同时设置私钥 `tls_key` 和证书 `tls_cert` 即可启动一个 https 服务。

**框架版本 >= v0.19.0 时**，支持在 `tls_key`, `tls_cert` 和 `ca_cert` 字段配置多个文件路径，两个文件路径之间用 **英文冒号`:`** 分隔，中间不要带空格。

https 协议分为“单向认证”和“双向认证”两种。

**单向认证：**只有一方验证另一方是否合法，通常是客户端验证服务端，因此服务端配置只需要设置 `tls_key`、`tls_cert` 即可开启单向认证。一般面向公众的 HTTPS 网站都是单向认证。

```yaml
server:                                            # 服务端配置
  service:                                         # 业务服务提供的 service，可以有多个
    - name: trpc.test.rpchttp.Hello                # service 的路由名称
      ip: 127.0.0.1                                # 服务监听 ip 地址 可使用占位符 ${ip},ip 和 nic 二选一，优先 ip
      port: 8000                                   # 服务监听端口 可使用占位符 ${port}
      network: tcp                                 # 网络监听类型  tcp udp
      protocol: http                               # 应用层协议 trpc http 
      timeout: 1000                                # 请求最长处理时间 单位 毫秒
      tls_key: ./license.key                       # 私钥路径
      # tls_key: ./licenseA.key:./licenseB.key     # 多个私钥路径，框架版本 >= v0.19.0
      tls_cert: ./license.crt                      # 证书路径
      # tls_cert: ./licenseA.crt:./licenseB.crt    # 多个证书路径，框架版本 >= v0.19.0
```

**双向认证：**服务端与客户端需要互相验证，在单向认证的基础上，增加 ca_cert 配置来验证客户端的合法性。一般银行等金融网站使用双向认证。

```yaml
server:                                            # 服务端配置
  service:                                         # 业务服务提供的 service，可以有多个
    - name: trpc.test.rpchttp.Hello                # service 的路由名称
      ip: 127.0.0.1                                # 服务监听 ip 地址 可使用占位符 ${ip},ip 和 nic 二选一，优先 ip
      port: 8000                                   # 服务监听端口 可使用占位符 ${port}
      network: tcp                                 # 网络监听类型  tcp udp
      protocol: http                               # 应用层协议 trpc http 
      timeout: 1000                                # 请求最长处理时间 单位 毫秒
      tls_key: ./license.key                       # 私钥路径
      # tls_key: ./licenseA.key:./licenseB.key     # 多个私钥路径，框架版本 >= v0.19.0
      tls_cert: ./license.crt                      # 证书路径
      # tls_cert: ./licenseA.crt:./licenseB.crt    # 多个证书路径，框架版本 >= v0.19.0
      ca_cert: ca.cert                             # ca 证书，用于校验 client 证书，以更严格识别客户端的身份，限制客户端的访问
      # ca_cert: ./caA.cert:./caB.cert             # 多个 ca 证书，框架版本 >= v0.19.0
```

## 5.3  作为 http2 服务

因为 http2 协议需要在 https 协议的基础上使用，所以我们需要通过设置“protocol”为“http2”，并设置 tls 配置即可启动一个 http2 服务。http2 同样支持“单向认证”和“双向认证”两种方式，具体参考 https 的配置。

```yaml
server:                                            # 服务端配置
  service:                                         # 业务服务提供的 service，可以有多个
    - name: trpc.test.rpchttp.Hello                # service 的路由名称
      ip: 127.0.0.1                                # 服务监听 ip 地址 可使用占位符 ${ip},ip 和 nic 二选一，优先 ip
      port: 8000                                   # 服务监听端口 可使用占位符 ${port}
      network: tcp                                 # 网络监听类型 tcp udp
      protocol: http2                              # 应用层协议 a
      timeout: 1000                                # 请求最长处理时间 单位 毫秒
      tls_key: ./license.key                       # https key 的路径
      # tls_key: ./licenseA.key:./licenseB.key     # 多个私钥路径，框架版本 >= v0.19.0
      tls_cert: ./license.crt                      # https key 的路径
      # tls_cert: ./licenseA.crt:./licenseB.crt    # 多个证书路径，框架版本 >= v0.19.0
```

## 5.4  作为 http3 服务

因为 http3 协议需要在 https 协议的基础上使用，所以我们需要通过设置 network 为**udp**，protocol 为 http3，并设置 tls 配置即可启动一个 http3 服务。http3 同样支持“单向认证”和“双向认证”两种方式，具体参考 https 的配置。

```yaml
server:                                            # 服务端配置
  service:                                         # 业务服务提供的 service，可以有多个
    - name: trpc.test.rpchttp.Hello                # service 的路由名称
      ip: 127.0.0.1                                # 服务监听 ip 地址 可使用占位符 ${ip},ip 和 nic 二选一，优先 ip
      port: 8000                                   # 服务监听端口 可使用占位符 ${port}
      network: udp                                 # 网络监听类型 tcp udp
      protocol: http3                              # 应用层协议 
      timeout: 1000                                # 请求最长处理时间 单位 毫秒
      tls_key: ./license.key                       # https key 的路径
      # tls_key: ./licenseA.key:./licenseB.key     # 多个私钥路径，框架版本 >= v0.19.0
      tls_cert: ./license.crt                      # https key 的路径
      # tls_cert: ./licenseA.crt:./licenseB.crt    # 多个证书路径，框架版本 >= v0.19.0
```

# 6 服务开发

绝大部分场景下，泛 HTTP RPC 服务的开发和 tRPC 服务的开发接口是完全一样的，业务不需要感知底层协议是 HTTP 协议还是 tRPC 协议。具体使用请参考 tRPC 服务的搭建。本章主要对一些需要感知 HTTP 协议场景的服务端开发做一些介绍，并通过代码示例来展示所有场景。

## 6.1 导入协议插件

- 对于 http，https 和 http2，生成桩代码时已经**自动**导入了插件
- 对于 http3 协议，需要额外**手动**导入协议插件：

```go
import (
    _ "git.code.oa.com/trpc-go/trpc-transport-quic/http3"
)
```

## 6.2 错误码使用

tRPC-Go 框架为泛 HTTP 服务提供了业务程序错误返回码机制。框架通过 HTTP 响应报文头中“trpc-error-msg”，“trpc-func-ret”，“trpc-ret”三个字段来返回错误码信息。关于错误码的约定，请参考 [tRPC-Go 错误码手册](https://iwiki.woa.com/pages/viewpage.action?pageId=276029299 "错误码手册")

- **trpc-error-msg**：返回错误具体信息，字符串格式
- **trpc-func-ret**：如果是业务侧错误，返回业务错误码
- **trpc-ret**：如果是框架错误，返回框架错误码

tRPC-Go 推荐：业务错误时，使用`errs.New()`来返回业务错误码，而不是在 body 里面自己定义错误码，这样框架就会自动上报业务错误的监控了，自己定义的话，那只能自己调用监控 sdk 自己上报。

## 6.3 定义 HTTP 状态码

tRPC-Go 框架对 RPC 的错误码和 HTTP 状态码默认进行了映射。错误码映射关系如下（最新代码定义请查看 [文档](https://godoc.woa.com/git.woa.com/trpc-go/trpc-go/http#pkg-variables) 中的 `ErrsToHTTPStatus` 变量）

| 错误码 | HTTP 状态码 | 错误信息                          |
|:---:|:--------:|-------------------------------|
|  1  |   400    | 服务端解码错误，解包失败                  |
|  2  |   500    | 服务端编码错误，序列化响应包失败，具体看 error 信息 |
| 11  |   404    | 服务端没有调用相应的 service 实现         |
| 12  |   404    | 服务端没有调用相应的接口实现                |
| 31  |   500    | 服务端系统错误，一般是 panic 引起的错误       |

对于不在上面表中的错误码，HTTP 状态码全部设置为 200。框架提供了接口供用户注册错误码和 HTTP 状态码的映射关系。API 定义为：

```go
// 注册错误码和 HTTP 状态码的映射关系
// 参数：code 表示错误码，httpStatus 表示 HTTP 状态码
func RegisterStatus(code int32, httpStatus int)
```

示例：我们设置服务端超载错误码的 HTTP 状态码为 503

```go
package main

import (
    "git.code.oa.com/trpc-go/trpc-go/errs"
    thttp "git.code.oa.com/trpc-go/trpc-go/http"
)

func init() {
    thttp.RegisterStatus(errs.RetServerOverload, 503)
}
```

## 6.4 操作原始数据

对于大部分泛 HTTP RPC 服务场景，业务层只需要使用框架反序列化后的 Body 数据就可以了。Head 数据通常只用于 RPC 框架用来管理 RPC 交互控制信息的。但是也存在少数业务会使用 HTTP Head 头携带业务层信息，或者处理 Cookie 等信息。

泛 HTTP RPC 服务实现是基于“net/http”标准库做封装的，框架提供了接口给业务，来直接获取和修改 HTTP 报文的原始信息，包括 HEAD。获取 HTTP 报文的 API 为：

```go
// 在请求报文处理上下文获取 HTTP 请求报文原始信息
func Head(ctx context.Context) *Header

type Header struct {
    // HTTP 包体二进制数据
    ReqBody  []byte

    // “net/http”标准库里的 Request
    Request  *http.Request

    // “net/http”标准库里的 ResponseWriter
    Response http.ResponseWriter
}
```

用户可以通过`Head(ctx)`函数获得“net/http”提供的“Request”和“ResponseWriter”变量，这样就可以通过“net/http”标准库提供的接口进行 Head 的操作了。

## 6.5 代码示例

下面的示例展示的是 HTTP RPC 服务，提供`SayHello()`接口。服务端读取 HTTP 请求报文头里的“request”字段，为响应报文头添加“Cookie”和“reply”字段，并返回"Hello, World!"消息给客户端。

```go
package main

import (
    "context"
    "net/http"

    "git.code.oa.com/trpc-go/trpc-go/log"

    "git.code.oa.com/trpc-go/trpc-go/errs"
    thttp "git.code.oa.com/trpc-go/trpc-go/http"
    pb "git.woa.com/trpcprotocol/test/rpchttp"
)

// SayHello ...
func (s *helloServiceImpl) SayHello(ctx context.Context, req *pb.HelloRequest, rsp *pb.HelloReply) error {
    head := thttp.Head(ctx)
    // 判断请求报文是否为泛 http 协议
    if head == nil {
    // 使用业务自定义错误码
        return errs.New(10000, "not http request")
    }
    // 获取请求报文头里的 "request" 字段
    reqHead := head.Request.Header.Get("request")
    // 获取请求报文头里的 "Cookie" 字段
    cookieStr := head.Request.Header.Get("Cookie")

     log.Infof("Msg: %s, reqHead: %s, cookie is: %s\n", req.Msg, reqHead, cookieStr)

     rsp.Msg = "Hello, World!"
     // 为响应报文设置 Cookie
     cookie := &http.Cookie{Name: "sample", Value: "sample", HttpOnly: false}
     http.SetCookie(head.Response, cookie)
     // 为响应报文头添加“reply”字段
     head.Response.Header().Add("reply", "tested")
     return nil
}
```

**可以通过`curl`命令来验证接口：**

```shell
curl -X POST -d '{"msg":"hello"}' -H "Content-Type:application/json" "http://127.0.0.1:8000/trpc.test.rpchttp.Hello/SayHello"
```

# 7 高级用法

## 7.1 自定义序列化类型

如果框架自带的序列化类型不满足业务需求，业务可以自定义序列化类型。自定义序列化接口函数为：

```go
// 注册自定义序列化类型
// 参数：httpContentType 为自定义序列化类型在“Content-Type”字段中填充的值
// 参数：serializationType 为自定义序列化类型在框架中对应的值，业务自定义序列化值必须大于等于 1000
func RegisterSerializer(httpContentType string, serializationType int, serializer codec.Serializer)

// 自定义序列化类型必须要实现的接口
type Serializer interface {
    // 反序列化函数
    Unmarshal(in []byte, body interface{}) error
    // 序列化函数
    Marshal(body interface{}) (out []byte, err error)
}
```

示例代码如下：

```go
package main

import (
    thttp "git.code.oa.com/trpc-go/trpc-go/http"
)

// ExampleSerialization 
type ExampleSerialization struct {
}

// Unmarshal 反序列
func (s *ExampleSerialization) Unmarshal(in []byte, body interface{}) error {
    // 业务需要实现把 in 反序列化的数据写到 body 中
    ...
}

// Marshal 序列化
func (s *ExampleSerialization) Marshal(body interface{}) ([]byte, error) {
    // 业务需要实现把 body 的数据序列化，并返回
    ...
}

func init() {
    thttp.RegisterSerializer("application/x-example", 1101, &ExampleSerialization{})
}

```

## 7.2 自定义解压缩方式

如果框架自带的解压缩方式不满足业务需求，业务可以自定义解压缩方式。自定义解压缩接口函数为：

```go
// 注册自定义解压缩方式，compressType 为解压缩方式编号，0~3 为系统保留，注意各业务不要重复。
func RegisterCompressor(compressType int, s Compressor)

// Compressor body 解压缩接口
type Compressor interface {
    Compress(in []byte) (out []byte, err error)
    Decompress(in []byte) (out []byte, err error)
}

// 注册 Http ContentEncoding 字段
// 参数：httpContentEncoding 为自定义解压缩方式在“Content-Encoding”字段中填充的值
// 参数：compressType 为自定义解压缩方式在框架中对应的值
func RegisterContentEncoding(httpContentEncoding string, compressType int)
```

示例代码如下：

```go
package main

import (
    "git.code.oa.com/trpc-go/trpc-go/codec"
    thttp "git.code.oa.com/trpc-go/trpc-go/http"
)

// ExampleCompressor 
type ExampleCompressor struct {
}

// Compress 压缩
func (e *ExampleCompressor) Compress(in []byte) (out []byte, err error) {
    ...
}

// Decompress 解压缩
func (e *ExampleCompressor) Decompress(in []byte) (out []byte, err error) {
    ...
}

func init() {
    codec.RegisterCompressor(100, &ExampleCompressor{})
    thttp.RegisterContentEncoding("y-example", 100)
}
```

## 7.3 自定义错误处理函数

tRPC-Go 框架为泛 HTTP RPC 服务提供了默认的错误处理行为，通过“trpc-error-msg”， ”trpc-func-ret”，”trpc-ret”来携带错误码信息。框架在捕获到错误后，默认行为如下：

1. 将错误的信息写入响应 Header 中的“trpc-error-msg”字段
2. 将业务返回的错误写入响应 Header 中的“trpc-func-ret”字段，
3. 将框架返回的错误写入响应 Header 中的“trpc-ret”字段
4. 设置错误码对应的 HTTP 状态码

但是对于如下典型场景：服务端使用 HTTP RPC 模式开发，但客户端不使用 tRPC-Go 框架，直接构造 HTTP 请求，并且要求 HTTP 错误响应报文遵循以下格式写 HTTP Boby：

```yaml
{
    "retcode": 10000,
    "retmsg": "服务器超载"
}
```

对于这种场景，tRPC-Go 框架提供了“自定义错误处理函数”API，供用户定制错误码逻辑，示例代码如下

```go
import (
    "net/http"

    "git.code.oa.com/trpc-go/trpc-go/errs"
    thttp "git.code.oa.com/trpc-go/trpc-go/http"
)

func init() {
    thttp.DefaultServerCodec.ErrHandler = func(w http.ResponseWriter, r *http.Request, e *errs.Error) {
        // 填充指定格式错误信息到 HTTP Body
        w.Write([]byte(fmt.Sprintf(`{"retcode": %d, "retmsg": "%s"}`, e.Code, e.Msg)))
    }
}
```

## 7.4 自定义返回数据处理函数

tRPC-Go 框架为泛 HTTP RPC 服务响应报文提供了默认处理函数：直接将数据写入到 http responseWriter 中，不会对数据进行任何修改处理。

但是对于如下典型场景：服务端使用 HTTP RPC 模式开发，但客户端不使用 tRPC-Go 框架，直接构造 HTTP 请求，并且要求所有 HTTP 响应报文遵循以下格式写 HTTP Boby：

```yaml
{
    "code": 0,
    "message": "",
    "data": ...,
}
```

对于这种场景，tRPC-Go 框架提供了“自定义返回数据处理函数”API，供用户定制响应报文的统一格式输出，示例代码如下

```go
import (
    "net/http"

    thttp "git.code.oa.com/trpc-go/trpc-go/http"
)

type Response struct {
    Code    int32           `json:"code"`
    Message string          `json:"message"`
    Data    json.RawMessage `json:"data"`
}

func init() {
    thttp.DefaultServerCodec.RspHandler = func(w http.ResponseWriter, r *http.Request, rspbody []byte) error {
        if len(rspbody) == 0 {
            return nil
        }
        bs, err := json.Marshal(&Response{Code: 0, Message: "OK", Data: rspbody})
        if err != nil {
            return err
        }
        _, err = w.Write(bs)
        return err
    }
}
```

## 7.5 如何支持跨域

可以选择使用**“cors filter”**插件，具体配置请参考 [配置 cors filter](https://git.woa.com/tRPC-Go/trpc-filter/tree/master/cors)

## 7.6 重复读取 HTTP 请求体

背景：在 HTTP RPC 服务下，HTTP 请求体 (`Request.Body`) 会被自动读取然后反序列化到请求结构体上，在一些情况下，用户期望在业务逻辑处理中重新读取原始的 HTTP 请求体。

版本要求：trpc-go 框架版本 >=v0.13.0

用法：

```go
import thttp "git.code.oa.com/trpc-go/trpc-go/http"

type impl struct{}

func (*impl) Hello(ctx context.Context, req *pb.Request) (*pb.Response, error) {
    r := thttp.Request(ctx)
    body, err := ioutil.ReadAll(r.Body)
    // ... 
}
```

## 7.7 如何避免自动回复 chunked 响应

thttp 底层默认依赖 go 的 net/http，底层实现会在 chunked writer 上面包一层 buffer，[这个 buffer 大小为 2048](https://github.com/golang/go/blob/go1.23.3/src/net/http/server.go#L1115)，当请求处理过程中，在 `http.ResponseWriter` 上调用 `Write` 写入数据的时候，会先写到这个上层 buffer 中，只要不超过 buffer 大小，就不会触发 chunked writer 本身的 `Write` 操作，当用户 handler 处理结束后，net/http 通过再触发 chunked writer 的写操作，此时[Content-Length 字段可以自动生成](https://github.com/golang/go/blob/go1.23.3/src/net/http/server.go#L1371)，不会以 chunked 形式回包。但是假如中间的写入操作大于 buffer 大小，会直接触发 chunked writer 本身的 `Write`，此时由于 handler 尚未处理完，chunked writer 会自动启用 chunked encoding 形式进行回包。

如果想要避免 chunked 响应，可以在 `RspHandler` 中明确带上回包的 Content-Length，如下：

```go
import (
    "net/http"

    thttp "git.code.oa.com/trpc-go/trpc-go/http"
)

func init() {
    thttp.DefaultServerCodec.RspHandler = func(w http.ResponseWriter, _ *http.Request, rspBody []byte) error {
        if len(rspBody) == 0 {
            return nil
        }
        w.Header().Add("Content-Length", len(rspBody))
        // 或者单独添加 
        // w.Header().Add("Transfer-Encoding", "identity")
        // 但是会导致额外带有 Connection: close（触发短连接），并且不会自动带 Content-Length
        if _, err := w.Write(rspBody); err != nil {
            return fmt.Errorf("http write response error: %s", err.Error())
        }
        return nil
    }
}
```

## 7.8 避免自动缓存请求体

tRPC-Go 框架 HTTP RPC 服务默认会自动缓存请求体，如果请求体过大，可能会导致内存占用过高。用户可以通过设置 `CacheRequestBody` 为 false 来避免自动缓存请求体。

版本要求：trpc-go 框架版本 >=v0.16.0

```go
import thttp "git.code.oa.com/trpc-go/trpc-go/http"

func init() {
    cacheRequestBody := false
    thttp.DefaultServerCodec.CacheRequestBody = &cacheRequestBody
}
```

# 8 基于 fasthttp 搭建泛 HTTP RPC 服务

基于 fasthttp 搭建泛 HTTP RPC 服务可以显著提高性能，具体请见 [tRPC-Go FastHTTP 性能测试](https://doc.weixin.qq.com/doc/w3_Ac0AYwanAIU1SL8vl0oS5SBRmTBVo?scode=AJEAIQdfAAokjbqh6wAc0AYwanAIU&version=4.1.28.6010&platform=win)

## 8.1 Naming Service 定义

对于泛 HTTP RPC 服务，我可以在 `trpc_go.yaml` 框架配置文件中通过 `protocol` 字段来指定具体协议类型，该部分与 http 仅有 `protocol` 字段的差距。不过 fasthttp 并不提供 http2 的支持。

### 8.1.1 提供 fasthttp 调用方式

```yaml
...
server:                                            # 服务端配置
  service:                                         # 业务服务提供的 service，可以有多个
    - name: trpc.test.rpchttp.Hello                # service 的路由名称
      ip: 127.0.0.1                                # 服务监听 ip 地址 可使用占位符 ${ip},ip 和 nic 二选一，优先 ip
      port: 8000                                   # 服务监听端口 可使用占位符 ${port}
      network: tcp                                 # 网络监听类型  tcp udp
      protocol: fasthttp                           # 应用层协议 trpc http 
      timeout: 1000                                # 请求最长处理时间 单位 毫秒
```

### 8.1.2 提供 fasthttps 调用方式

fasthttps 通过设置 TLS 配置进行启用。

**框架版本 >= v0.19.0 时**，支持在 `tls_key`, `tls_cert` 和 `ca_cert` 字段配置多个文件路径，两个文件路径之间用 **英文冒号`:`** 分隔，中间不要带空格。

**单向认证**：只有一方验证另一方是否合法，通常是客户端验证服务端，因此服务端配置只需要设置 `tls_key`、`tls_cert` 即可开启单向认证。一般面向公众的 HTTPS 网站都是单向认证。

```yaml
server:                                            # 服务端配置
  service:                                         # 业务服务提供的 service，可以有多个
    - name: trpc.test.rpchttp.Hello                # service 的路由名称
      ip: 127.0.0.1                                # 服务监听 ip 地址 可使用占位符 ${ip},ip 和 nic 二选一，优先 ip
      port: 8000                                   # 服务监听端口 可使用占位符 ${port}
      network: tcp                                 # 网络监听类型  tcp udp
      protocol: fasthttp                           # 应用层协议 trpc http 
      timeout: 1000                                # 请求最长处理时间 单位 毫秒
      tls_key: ./license.key                       # 私钥路径
      # tls_key: ./licenseA.key:./licenseB.key     # 多个私钥路径，框架版本 >= v0.19.0
      tls_cert: ./license.crt                      # 证书路径
      # tls_cert: ./licenseA.crt:./licenseB.crt    # 多个证书路径，框架版本 >= v0.19.0
```

**双向认证**：服务端与客户端需要互相验证，在单向认证的基础上，增加 ca_cert 配置来验证客户端的合法性。一般银行等金融网站使用双向认证。

```yaml
server:                                            # 服务端配置
  service:                                         # 业务服务提供的 service，可以有多个
    - name: trpc.test.rpchttp.Hello                # service 的路由名称
      ip: 127.0.0.1                                # 服务监听 ip 地址 可使用占位符 ${ip},ip 和 nic 二选一，优先 ip
      port: 8000                                   # 服务监听端口 可使用占位符 ${port}
      network: tcp                                 # 网络监听类型  tcp udp
      protocol: fasthttp                           # 应用层协议 trpc http 
      timeout: 1000                                # 请求最长处理时间 单位 毫秒
      tls_key: ./license.key                       # 私钥路径
      # tls_key: ./licenseA.key:./licenseB.key     # 多个私钥路径，框架版本 >= v0.19.0
      tls_cert: ./license.crt                      # 证书路径
      # tls_cert: ./licenseA.crt:./licenseB.crt    # 多个证书路径，框架版本 >= v0.19.0
      ca_cert: ca.cert                             # ca 证书，用于校验 client 证书，以更严格识别客户端的身份，限制客户端的访问
      # ca_cert: ./caA.cert:./caB.cert             # 多个 ca 证书，框架版本 >= v0.19.0
```

## 8.2 服务开发

FastHTTP RPC 在设计时尽可能保持了与 HTTP RPC 在行为上的一致性，但由于各种原因，可能在用法上的一致性欠佳，以下给出与 HTTP RPC 用法不同的各种案例代码以供用户迁移，如果没有给出代码案例，则直接使用 HTTP RPC 的代码即可。

若想更加细致了解 tfasthttp 和 thttp 的区别，请见 [tfasthttp 使用指南](https://doc.weixin.qq.com/doc/w3_Ac0AYwanAIUfx1rVLYYTm2A4u2oHj?scode=AJEAIQdfAAowr0OpC7Ac0AYwanAIU&version=4.1.28.6010&platform=win)。

### 8.2.1 操作原始数据

对于大部分泛 HTTP RPC 服务场景，业务层只需要使用框架反序列化后的 Body 数据就可以了。但是也存在少数业务会使用 HTTP Head 头携带业务层信息，或者处理 Cookie 等信息。与 HTTP RPC 不一样的时，用户需要获取 requestCtx 而非 Header。用户可以通过 RequestCtx(ctx) 函数获得 fasthttp 提供的 Request 和 Response 变量，这样就可以通过 fasthttp 库提供的接口进行操作了。

```go
// 在请求报文处理上下文获取 HTTP 请求报文原始信息
func RequestCtx(ctx context.Context) *fasthttp.RequestCtx

type RequestCtx struct {
    ...
    noCopy noCopy
    // fasthttp 库里的 Request
    Request Request
    // fasthttp 库里的 Request
    Response Response
    ...
}
```

### 8.2.2 代码示例

下面的示例展示的是 FastHTTP RPC 服务，提供 SayHello() 接口。服务端读取 FastHTTP 请求报文头里的 "request" 字段，为响应报文头添加 "Cookie" 和 "reply" 字段，并返回 "Hello, World!" 消息给客户端。

```go
// SayHello ...
func (s *helloImpl) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
    rsp := &pb.HelloReply{}
    requestCtx := thttp.RequestCtx(ctx)
    // 判断请求报文是否为泛 http 协议
    if requestCtx == nil {
        // 使用业务自定义错误码
        return nil, errs.New(10000, "not fasthttp requestCtx")
    }
    // 获取请求报文头里的 "request" 字段
    reqHead := string(requestCtx.Request.Header.Peek("request"))
    // 获取请求报文头里的 "Cookie" 字段
    cookieStr := string(requestCtx.Request.Header.Peek("Cookie"))
    log.Infof("Msg: %s, reqHead: %s, cookie is: %s\n", req.Msg, reqHead, cookieStr)
    rsp.Msg = "Hello, World!"
    // 为响应报文设置 Cookie
    cookie := fasthttp.AcquireCookie()
    defer fasthttp.ReleaseCookie(cookie)
    cookie.SetKey("sample")
    cookie.SetValue("sample")
    cookie.SetHTTPOnly(false)
    requestCtx.Response.Header.SetCookie(cookie)
    // 为响应报文头添加 "reply" 字段
    requestCtx.Response.Header.Add("reply", "tested")
    return rsp, nil
}
```

## 8.3 高级用法

### 8.3.1 自定义错误处理函数

FastHTTP RPC 主要是 `ErrHandler` 的入参和类型与 HTTP RPC 不一样。FastHTTP RPC 要处理的是 `fasthttp` 的请求和响应，而 HTTP RPC 处理的是 `net/http` 的请求和响应，两者的类型存在差异。

```go
import (
    "net/http"

    "git.code.oa.com/trpc-go/trpc-go/errs"
    thttp "git.code.oa.com/trpc-go/trpc-go/http"
)

func init() {
    thttp.DefaultFastHTTPServerCodec.ErrHandler = func(requestCtx *fasthttp.RequestCtx, e *errs.Error) {
        // 填充指定格式错误信息到 FastHTTP Body
        requestCtx.WriteString(fmt.Sprintf(`{"retcode": %d, "retmsg": "%s"}`, e.Code, e.Msg))
    }
}
```

### 8.3.2 自定义返回数据处理函数

与自定义错误处理函数类似，FastHTTP RPC 主要是 `RspHandler` 的入参和类型与 HTTP RPC 不一样。FastHTTP RPC 要处理的是 `fasthttp` 的请求和响应，而 HTTP RPC 处理的是 `net/http` 的请求和响应，两者的类型存在差异。

```go
import (
    "net/http"

    thttp "git.code.oa.com/trpc-go/trpc-go/http"
)

type Response struct {
    Code    int32           `json:"code"`
    Message string          `json:"message"`
    Data    json.RawMessage `json:"data"`
}

func init() {
    thttp.DefaultFastHTTPServerCodec.RspHandler = func(requestCtx *fasthttp.RequestCtx, rspBody []byte) error {
        if len(rspBody) == 0 {
            return nil
        }
        bs, err := json.Marshal(&Response{Code: 0, Message: "OK", Data: rspBody})
        if err != nil {
            return err
        }
        requestCtx.Write(bs)
        return nil
    }
}
```

### 8.3.3 重复读取 FastHTTP 请求体

`fasthttp` 的请求 Body 与 `net/http` 的请求 Body 类型不一样，因此可以重复读取，但请注意其生命周期与 `fasthttp.Request` 保持一致。

```go
// fasthttp
// Body returns request body.
// The returned value is valid until the request is released, either though ReleaseRequest or your request handler returning. Do not store references to returned value. Make copies instead.
func (req *Request) Body() []byte

// net/http
type Request struct {
    ...
    // Body is the request's body.
    //
    // For client requests, a nil body means the request has no
    // body, such as a GET request. The HTTP Client's Transport
    // is responsible for calling the Close method.
    //
    // For server requests, the Request Body is always non-nil
    // but will return EOF immediately when no body is present.
    // The Server will close the request body. The ServeHTTP
    // Handler does not need to.
    //
    // Body must allow Read to be called concurrently with Close.
    // In particular, calling Close should unblock a Read waiting
    // for input.
    Body io.ReadCloser
    ...
}
```

## 更多问题

请参考 [tRPC 技术咨询](https://iwiki.woa.com/p/491739953) 以寻求帮助
