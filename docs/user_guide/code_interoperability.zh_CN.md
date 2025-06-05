## 1 前言

业务难免会有许多陈旧的业务代码难以迁移和一次性改造成 trpc，但是我们也不能一味在这些代码上再叠加新的逻辑，那只会将代码变的越来越不可维护，所以更好的办法是**新接口用 trpc 来开发，老接口不再改动，有人力则重构老接口** 。
这其中就涉及到一个 trpc 与存量服务互通的问题，通常有这几种情况：

1. 存量老服务如何调用 trpc 接口
2. trpc 如何调用存量老服务接口
3. 存量老服务如何重构为 trpc 服务

下面将介绍如何解决这三个问题。

## 2 原理

名词解释：

- **tRPC protocol**，是指 tRPC 统一的协议，由帧头 + 包头 + 业务包体构成。
- **trpc-codec**, 是`tRPC-Go`的重要模块，负责业务协议打解包的实现，通过实现 codec 相关接口就可以和任意的第三方存量服务进行通信，已支持如`grpc\sso\oidb\tars`等业务协议，支持`PB/JSON/JCE`等序列化方式。
- **北极星**，公司统一、业界领先的服务治理平台，实现了 RPC 的服务注册发现、动态路由、负载均衡和容错问题。
- **北极星别名**，可以为北极星名字设置 L5 名字，打通存量服务到 tRPC-Go 服务的路由寻址，方便老服务继续以`L5 Agent`的方式访问老协议的`tRPC-GO`服务。
- **trpc-naming-polaris**，是 tRPC-Go 框架中默认使用的名字服务插件，提供了“服务注册、发现、负载均衡”等能力，北极星已打通`L5\ons\CMLB`。

## 3 实现

### 3.1 存量服务调用 tRPC 服务

这里可以简单分为两种情况：

- 基于 tRPC-Go 框架实现的老协议新服务，这种无需处理，同时通过设置 L5 别名的方式支持部署在 123 平台，使存量服务无成本切换。
- 基于 tRPC-Go 框架实现的`tRPC Protocol`统一协议的新服务，此时需要存量服务去支持对 tRPC 协议的访问，有一定开发成本。

这里主要介绍第二种场景下的解决方案。如果您的存量服务是非 tRPC-Go 框架的 Go 项目，请看示例 1；如果您的存量服务是非 Go 语言的框架，请看示例 2。

#### 3.1.1 Go stub client 访问 tRPC-Go 服务

如果你的存量服务是非 tRPC-Go 框架的 Go 项目，可以直接低成本使用`trpc 工具`或者`rick 平台`生成的`stub client`

1. 首先假设您已经您已经成功部署了一个 tRPC-Go 服务，假定服务为`trpc.qqva.vip_prividata_server.vip_prividata_server`
   并采用 [rick 平台](http://trpc.rick.oa.com/) 进行接口管理。
   如果您还不清楚如何构建 tRPC-Go 服务，请参考文档：
    - [tRPC-Go 快速上手 wiki](https://iwiki.woa.com/p/118272478)
    - [tRPC-Go 接口管理 wiki](https://iwiki.woa.com/p/99485686)

2. 然后使用协议生成`stub client`访问`tRPC-Go`服务

    ```go
    package main
    import (
        "fmt"
        trpc "git.code.oa.com/trpc-go/trpc-go"
        "git.code.oa.com/trpc-go/trpc-go/client"
        pselector "git.code.oa.com/trpc-go/trpc-naming-polaris/selector"
        pb "git.code.oa.com/trpcprotocol/qqva/vip_prividata_server"
    )
    func main() {
        // 不同于 tRPC-Go Server 在初始化时会帮忙初始化插件，这里需要自己手动初始化北极星
        pselector.RegisterDefault()
        // 利用协议生成的 xxx.trpc.go stub client 创建一个客户端调用代理
        proxy := pb.NewVipPrividataServerClientProxy()
        req := &pb.GetPriviDataRequest{}
        // 非 tRPC-Go 框架则必须自己通过 trpc.BackgroundContext() 创建 ctx，通过代码传入 option 参数
        // option 参数参考：https://iwiki.woa.com/pages/viewpage.action?pageId=284289117
        rsp, err := proxy.GetPriviData(trpc.BackgroundContext(), req,
            client.WithNamespace("Production"),    // 设置北极星路由环境示例
            client.WithMetaData("uid", []byte("10001"))) // 设置 Meta 参数示例
        if err != nil {
            fmt.Printf("get data fail: %v", err)
            return
        }
        fmt.Printf("rsp info[%+v]", rsp)
        return
    }
    ```

#### 3.1.2 C++ 访问 tRPC-Go 服务

如果你的存量服务是非 Go 语言项目，则需要自行封装 client，大体思路都是按照`trpc-protocol`统一协议去打解包。
首先我们看下`trpc protocol`的协议设计，具体协议可以参考  [trpc 统一协议](https://git.woa.com/trpc/trpc-protocol/blob/master/docs/protocol_design.md)
![trpc protocol](../../.resources/user_guide/code_interoperability/trpc-protocol.png)

1. 首先我们按照协上面的议进行打解包，这里展示`C++`打包`tRPC`请求的大致过程

    ```cpp
    // 请求包打包函数
    // return<0 编码失败
    // return>0 编码后的数据长度
    // return=0 缓存不够
    virtual int encode_req(unsigned flow, char *pui_buff, int len) {
        m_flow = flow;
        // trpc 包头 RequestProtocol 相关参数填充。.. 这里省略部分参数
        m_RequestProtocol.set_version(strParameter.version);
        m_RequestProtocol.set_callee(strParameter.callee.c_str());
        m_RequestProtocol.set_func(strParameter.func.c_str());
        uint16_t head_len= m_RequestProtocol.ByteSizeLong();
        uint32_t body_len = m_req.ByteSizeLong();
        int ret = 0;
        if (head_len + body_len + 16  > (uint32_t)len) {
            ret = ENUM_ERR_BUFFER_NOT_ENOUGH;
            return ENUM_ERR_BUFFER_NOT_ENOUGH;
        }
        int idx = 0;
        // 填充魔数
        *reinterpret_cast<uint16_t *>(pui_buff + idx) = htons(MAGIC_VALUE);
        idx += sizeof(uint16_t);
        // 填充协议类型
        *reinterpret_cast<uint8_t *>(pui_buff + idx) = htonl(0);
        idx += sizeof(uint8_t);
        // 填充协议状态
        *reinterpret_cast<uint8_t *>(pui_buff + idx) = htonl(0);
        idx += sizeof(uint8_t);
        // 填充数据包总大小
        *reinterpret_cast<uint32_t *>(pui_buff + idx) = htonl(16+head_len+body_len);
        idx += sizeof(uint32_t);
        // 填充头部长度
        *reinterpret_cast<uint16_t *>(pui_buff + idx) = htons(head_len);
        idx += sizeof(uint16_t);
        // 填充流 id
        *reinterpret_cast<uint16_t *>(pui_buff + idx) = htonl(0);
        idx += sizeof(uint16_t);
        // 保留字段，4 字节
        idx += sizeof(uint32_t);
        // 序列化包头
        ret = m_RequestProtocol.SerializeToArray(pui_buff + idx, head_len);
        if (!ret) {
            return ENUM_ERR_PACK;
        }
        idx += head_len;
        // 序列化包体
        ret = m_req.SerializeToArray(pui_buff + idx, body_len);
        if (!ret) {
            return ENUM_ERR_PACK;
        }
        idx += body_len;
        return idx;
    }
   ```

2. 然后通过北极星 SDK 进行名字路由。
   北极星是公司统一、业界领先的服务治理组件，[北极星接入文档](https://iwiki.woa.com/pages/viewpage.action?pageId=68848645)
   北极星 SDK 已支持 C++/Go/Java/NodeJS 等多语言，同时支持 L5 别名，可以通过 L5 Agent 进行路由。
3. 到这一步您的 C++ 项目就可以与 tRPC 服务进行通信了。
4. 最后解析回包数据同步骤 1。
   其余语言如 PHP/NodeJS 等需要自行实现 client 打解包策略。

### 3.2 tRPC 服务调用存量服务

#### 3.2.1 trpc-codec 解决通信协议互通问题

`Codec`模块以插件的形式可以拓展支持存量服务的协议，其中包含`codec 编码`、`serializer 序列化`、`compressor 压缩`等核心接口。
client 请求下游服务时的过程如下：

```raw
serializer marshal reqbody 
-> compressor compress reqbody-bytes 
-> codec encode request-buffer 
-> ...transport roundtrip call... 
-> codec decode response-buffer 
-> compressor decompress rspbody-bytes 
-> serializer unmarshal rspbody
```

目前已支持的第三方协议可见 [trpc-codec 仓库](https://git.woa.com/trpc-go/trpc-codec)。  
如果您的协议尚未支持，您需要自行实现下列接口。

- [FrameBuilder 拆包接口](/transport/transport.go)
- [Codec 打解包接口](/codec/codec.go)

更多实现细节请参考： [tRPC-Go 模块：codec wiki](https://iwiki.woa.com/p/99485474)

协议指定方式：可以在`trpc-go.yaml`文件中指定协议，也可以在编码`client.Option`中指定。

```go
client.WithProtocol("oidb")
```

同时框架已支持多种序列化方式：

```go
const (
    SerializationTypePB         = 0 // protobuf
    SerializationTypeJCE        = 1 // jce
    SerializationTypeJSON       = 2 // json
    SerializationTypeFlatBuffer = 3 // flat buffer
    SerializationTypeNoop       = 4 // bytes 二进制数据空序列化方式
    SerializationTypeXML        = 5 // http application/xml
    SerializationTypeTextXML    = 6 // http text/xml
    SerializationTypeUnsupported = 128 // 不支持
    SerializationTypeForm        = 129 // http form data 表单 kv 结构
    SerializationTypeGet         = 130 // http server 处理 get 请求
    SerializationTypeFormData    = 131 // 处理 form data 表单数据
)
```

序列指定方式：在编码`client Option`中指定。

```go
client.WithSerializationType(codec.SerializationTypeJCE)
```

#### 3.2.2 trpc-naming 解决服务发现、路由与负载均衡问题

名字服务插件目是保证服务位置的透明，避免调用方固定 ip:port 调用。框架中默认接入北极星插件 trpc-naming-polaris，同时北极星插件已打通 CMLB/L5/ONS。
插件使用方式，编码`client.Option`中指定：

```go
opts := []client.Option{
    client.WithNamespace("Production"),
    // trpc-go 框架内部使用
    client.WithServiceName("12587:65539"),
    // 纯客户端或者其他框架中使用 trpc-go 框架的 client
    // client.WithTarget("polaris://12587:65539"),
    client.WithDisableServiceRouter(),
}
```

如果还是不能满足业务需求时，可以自行实现 Selector 接口，然后注册到 selector 中，可以参考：

- trpc-selector-srf <https://git.woa.com/gamecenter-opensource/trpc-selector-srf/blob/master/srfselector.go#L24>

#### 3.2.3 OIDB 协议示例

下面以`oidb`协议为例介绍如何在`tRPC-Go`项目中访问`oidb`存量服务。
框架已经做了 oidb 协议 codec 相关接口的实现，见： /[trpc-code oidb](https://git.woa.com/trpc-go/trpc-codec/blob/master/oidb/codec.go#L124)/
我们可以直接在项目中实现对 oidb 服务的访问了。

1. 调用 oidb 协议服务的编码方式

    ```go
    import "git.code.oa.com/trpc-go/trpc-codec/oidb"
    head := &oidb.OIDBHead{
         Uint64Uin: proto.Uint64(10000), 
         Uint32Command: proto.Uint32(0x1100), 
         Uint32ServiceType: proto.Uint32(1),
    }
    err := oidb.Do(ctx, head, reqbody, rspbody)
    ```

2. 下面是 oidb.Do 的实现逻辑，封装了 oidb 协议 + pb 序列化访问存量 oidb 服务实例的方法，路由方式可以配置 cmlb 或者 L5。

    ```go
    func Do(ctx context.Context,
        head *OIDBHead,
        reqbody proto.Message,
        rspbody proto.Message,
        opts ...client.Option) error {
        // 指定 oidb cmd 和 serviceType
        cmd := head.GetUint32Command()
        serviceType := head.GetUint32ServiceType()
        if len(head.GetStrServiceName()) == 0 {
            head.StrServiceName = proto.String(path.Base(os.Args[0]))
        }
        ctx, msg := codec.WithCloneMessage(ctx)
        msg.WithClientReqHead(head)
        msg.WithClientRspHead(head)
        msg.WithClientRPCName(fmt.Sprintf("/0x%x/%d", cmd, serviceType))
        // serviceName: trpc.oidb.cmd0xc07.downservice，在 trpc_go.yaml 配置相关参数，如 target\timeout 等
        msg.WithCalleeServiceName(fmt.Sprintf("trpc.oidb.cmd0x%x.downservice", cmd))
        // pb 序列化
        msg.WithSerializationType(codec.SerializationTypePB)
        // 指定 protocol 为 oidb, network=udp
        opt := []client.Option{
            client.WithProtocol("oidb"),
            client.WithNetwork("udp"),
        }
        opts = append(opt, opts...)
        return client.DefaultClient.Invoke(ctx, reqbody, rspbody, opts...)
    }
    ```

### 3.3 存量服务重构

由于 trpc 已支持多种存量协议，在用 tRPC-Go 重构老服务时，可以选择 tRPC 统一协议或者继续使用老协议。

- tRPC-Go + 老协议，只需要实现业务逻辑重构，上游服务无需推动更改，成本低
- tRPC-Go + tRPC 统一协议，业务逻辑重构后，还需要继续推动上游服务流量逐步灰度切换到新重构服务，成本高

当然，我们这里强烈推荐使用 tRPC 统一协议，公司的目标是统一协议，可以享受到框架的新特性支持和维护服务。
然而很多业务面临很大的历史包袱，上游请求服务较多，不能一次性推动所有请求方切换到 tRPC-Go 新服务。此时我们可以做一层转发代理来过渡，即将老协议的请求转换为 trpc 协议包头的请求，并通过配置的方式转发到重构后的 tRPC-Go 服务。

#### 3.3.1 原理

整体思路如下所示，在老服务来不及重构为 trpc 协议的时候，可以通过代理转发请求到重构后的 tRPC 服务。
![proxy forward](../../.resources/user_guide/code_interoperability/proxy_forward.png)

注意这里代理只做一层协议转换，在实现上要做到高性能、可拓展、可配置。

#### 3.3.2 示例

推荐看点的 oidb 接入服务代理，各业务和协议可以参考实现。

看点 oidb 接入服务代理：[oidb-trpc-proxy](https://git.woa.com/tkd/proxy/oidb-trpc-proxy)
![oidb example](../../.resources/user_guide/code_interoperability/oidb_example.png)

实现逻辑：

- 收前端请求包，将 oidb req head 转成 trpc req head
- 读取配置中心数据，确定当前请求往哪里转发
- 调用 trpc 协议服务
- 收后端响应包，将 trpc rsp head 转成 oidb rsp head
- 回包给上游

## 4 FAQ

### Q: 存量服务重构后，如何绑定存量 L5？

A: stke 平台或者 123 平台都已经支持绑定存量 L5。

### Q: 如何为北极星地址构建 L5 别名，以兼容上游 L5 寻址？

A: <http://polaris.oa.com/#/polaris/aliases> 可以在此处新建 L5 别名

## 更多问题

请参考 [tRPC 技术咨询](https://iwiki.woa.com/p/491739953) 以寻求帮助
