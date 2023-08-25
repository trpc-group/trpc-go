[TOC]

tRPC-Go 插件开发向导



# 前言

tRPC-Go 设计遵循了插件化架构设计理念，通过插件来实现框架核心与各种生态体系的对接（包括不同协议的服务和纷繁多样的服务治理组件)，提供了框架的开放性和可扩展性。本文从插件模型，插件注册和接口实现 3 个方面来阐述如何开发一个插件。

# 插件模型

tRPC-Go 框架是采用基于接口编程的思想，通过把框架功能抽象成一系列的功能组件，为组件定义标准接口，由插件实现具体功能。框架负责串联这些插件组件，拼装出完整的框架功能。我们可以把插件模型分为以下三层：

- 框架设计层：框架只定义标准接口，没有任何插件实现，与平台完全解耦；
- 插件实现层：将插件按框架标准接口封装起来即可实现框架插件；
- 用户使用层：业务开发只需要引入自己需要的插件即可，按需索取，拿来即用。

**框架** 提供的标准接口按功能组件分为以下 5 类：

- 配置：提供获取配置的标准接口，通过从本地文件，配置中心等多种数据源获取配置数据，提供 json，yaml 等多种格式的配置解析，同时框架也提供了 watch 机制，来实现配置的动态更新
- 日志：提供了统一的日志打印和日志上报标准接口。日志插件通过实现日志上报接口来完成和远程日志系统的对接
- 协议：提供协议编解码相关的标准接口，允许通过插件的方式来扩展业务协议、序列化类型、数据压缩方式等协议处理
- 名字服务：包括服务注册，服务发现，策略路由，负载均衡，熔断等标准接口，用于实现服务路由寻址
- 拦截器：提供了通用拦截器接口，用户可以在服务调用的上下文设置埋点，实现例如模调监控，横切日志，链路跟踪，过载保护等功能

**框架** 通过“插件工厂”来管理所有插件的，每个插件都需要注册到插件工厂。插件工厂采用了两级管理模式：第一级为插件类型（比如：log,conf,selector 等），第二级为插件名称（比如 conf 的插件有很多，包括七彩石，tconf，本地配置，数据库等）。框架只提供了插件的管理模式，和初始化流程。对于插件的类型，框架并没有做限制，用户可以自行添加插件类型。

![plugin_factory](/.resources/developer_guide/develop_plugins/overview/plugin_factory_zh_CN.png)

**插件** 是框架核心和外部服务治理组件串联起来的桥梁。插件一边需要按框架标准接口实现插件，注册到框架核心，并完成插件实例化；另一边插件需要调用外部服务治理服务的 SDK/API, 实现如服务发现，负载均衡，监控，调用链等服务治理功能。下图是一个典型的名字系统实现方式

![plugin](/.resources/developer_guide/develop_plugins/overview/plugin_zh_CN.png)

# 插件注册

框架内的插件都是通过插件工厂来管理的，所有插件都需要注册到插件工厂。对于插接件的注册，框架提供了两种注册方式：有配置文件方式 和 无配置文件方式。

## 有配置文件

对于大部分的插件，都是需要有配置才能实例化的，比如远程日志插件，它在初始化时需要知道日志的打印方式（文件，console 或者远程日志服务）。对于这类插件，框架在框架配置文件中开辟了一块配置区域，用于插件的配置设置。配置文件格式为：

```yaml
# 插件配置
plugins:
  # 插件类型
  log:
    # 插件名
    logger1:
      # 插件详细配置，具体请参考各个插件的说明
      ....
    logger2:
      # 插件详细配置，具体请参考各个插件的说明
      ....
  # 插件类型
  config:
    # 插件名
    rainbow:
      # 插件详细配置，具体请参考各个插件的说明
      ....
    tconf:
      # 插件详细配置，具体请参考各个插件的说明
      ....
```

这里定义了 2 个插件类型和 4 个插件：日志插件类型下有 logger1，logger2 日志插件。配置插件类型下有七彩石和 tconf 两个配置中心插件。

插件的配置是由各插件自行定义的，框架并不使用，也不理解插件的配置含义。各插件需要自行实现插件配置的解析和插件的初始化。插件的注册需要实现以下接口：

```go
// Factory 插件工厂统一抽象 外部插件需要实现该接口，通过该工厂接口生成具体的插件并注册到具体的插件类型里面
type Factory interface {
    // Type 插件的类型 如 selector log config tracing
    Type() string
    // Setup 根据配置项节点装载插件，用户自己先定义好具体插件的配置数据结构
    Setup(name string, configDec Decoder) error
}

// Register 注册插件工厂 可自己指定插件名，支持相同的实现 不同的配置注册不同的工厂实例
func Register(name string, p Factory)
```

然后在插件的 init() 函数中调用注册函数，把插件加入到插件工厂的对应插件类型下。例如：

``` go
// 注册插件到插件工厂，插件类型为tconfPlugin下定义的Type()
plugin.Register("tconf", &tconfPlugin{})
```

在使用插件之前，需要在 main package 下 import 插件包，例如：

``` go
package main

import (
    _ "git.code.oa.com/trpc-go/trpc-config-tconf"
)
```

插件配置的导入发生在 tRPC-Go server 初始化时，通过调用 trpc.NewServer() 函数。tRPC-Go 框架在初始化时，通过读取框架配置文件中“plugins”中所有插件配置，调用每个插件的“Setup()”函数来完成插件配置的解析和初始化，从而完成插件的注册。

一般情况下，插件之间是相互独立的，框架会按随机顺序逐个初始化插件。如果某个插件依赖其它插件（比如 tconf 插件依赖北极星寻址插件），插件可以实现以下方法来声明依赖关系。

``` go
// 强依赖其他插件，返回插件数组，每个元素格式是：插件类型-插件名（注意：中间是个减号）。
// 该接口是可选的，只有存在强依赖关系时才需要实现该接口。
type Depender interface {
    DependsOn() []string
}

// 弱依赖接口，与强依赖不同，插件并不要求被依赖的插件必须存在。
// 如果被依赖者存在，则一定在被依赖者之后进行初始化。
// 可选，只有存在弱依赖关系时才需要实现该接口。
type FlexDepender interface {
    FlexDependsOn() []string
}
```

依赖关系分为强依赖和弱依赖。强依赖要求被依赖的插件必须存在，不存在框架会 panic。弱依赖则不会 panic。框架会先确保所有强依赖都被满足，然后再检查弱依赖。

例如，在下面的例子中，插件的初始化强依赖于寻址（selector）插件北极星（polaris），弱依赖于配置（config）插件七彩石（rainbow）。

``` go
func (p *Plugin) DependsOn() []string {
    return []string{"selector-polaris"}
}
func (p *Plugin) FlexDependsOn() []string {
    return []string{"config-rainbow"}
}
```

## 无配置文件

tRPC-Go 框架中有少数插件类型，在初始化时并不需要有配置。这些模块的插件，没有使用框架配置文件的方式来完成插件注册。而是由框架模块提供独立的插件注册函数，供用户在需要的时候自行注册插件。系统使用此方法的功能组件包括：

- Codec 的注册函数

``` go
type Codec interface {
  //server解包 从完整的二进制网络数据包解析出二进制请求包体
  Decode(message Msg, request-buffer []byte) (reqbody []byte, err error)
  //server回包 把二进制响应包体打包成一个完整的二进制网络数据
  Encode(message Msg, rspbody []byte) (response-buffer []byte, err error)
}
// Register 通过协议名注册Codec，由第三方具体实现包的init函数调用, 只有client没有server的情况，serverCodec填nil即可
func Register(name string, serverCodec Codec, clientCodec Codec)
```

- 序列化类型的注册函数

``` go
type Serializer interface {
  //server解包出二进制包体后，调用该函数解析到具体的reqbody结构体
  Unmarshal(req-body-bytes []byte, reqbody interface{}) error
  //server回包rspbody结构体，调用该函数转成二进制包体
  Marshal(rspbody interface{}) (rsp-body-bytes []byte, err error)
}
// RegisterSerializer 注册序列化具体实现Serializer，由第三方包的init函数调用
func RegisterSerializer(serializationType int, s Serializer)
```

- 压缩方式的注册函数

``` go
type Compressor interface {
  //server解出二进制包体，调用该函数，解压出原始二进制数据
  Decompress(in []byte) (out []byte, err error)
  //server回包二进制包体，调用该函数，压缩成小的二进制数据
  Compress(in []byte) (out []byte, err error)
}
// RegisterCompressor 注册解压缩具体实现，由第三方包的init函数调用
func RegisterCompressor(compressType int, s Compressor)
```

# 接口实现

在完成插件的配置和注册之后，用户就需要实现为插件的定义的标准接口了。框架定义的标准接口主要包括：filter，config，log，selector，codec。用户可以参考以下开发文档获取标准接口的定义。

- 拦截器的开发请参考 [tRPC-Go 开发拦截器插件](https://git.woa.com/trpc-go/trpc-wiki/blob/main/developer_guide/develop_plugins/interceptor_zh-CN.md) 文档
- 配置插件的开发请参考 [tRPC-Go 开发配置插件](https://git.woa.com/trpc-go/trpc-wiki/blob/main/developer_guide/develop_plugins/config_CN.md) 文档
- 日志插件的开发请参考 [tRPC-Go 开发日志插件](https://git.woa.com/trpc-go/trpc-wiki/blob/main/developer_guide/develop_plugins/log_CN.md) 文档
- 名字服务的开发请参考 [tRPC-Go 开发名字服务插件](https://git.woa.com/trpc-go/trpc-wiki/blob/main/developer_guide/develop_plugins/naming_zh_cn.md) 文档
- 监控插件的开发请参考 [tRPC-Go 开发监控插件](https://git.woa.com/trpc-go/trpc-wiki/blob/main/developer_guide/develop_plugins/metric_ZH.md) 文档
- 分布式追踪插件的开发请参考 [tRPC-Go 开发分布式追踪插件](https://git.woa.com/trpc-go/trpc-wiki/blob/main/developer_guide/develop_plugins/open_tracing_zh_cn.md) 文档
- 协议插件的开发请参考 [tRPC-Go 协议开发](todo) 文档

# 示例

下面的例子，提供了一个使用配置文件的自定义插件的注册部分代码。插件类型为 foo，插件名称为 default，

```go
const (
    pluginType = "foo"
    pluginName = "default"
)

type Foo struct {
    Nums []int `yaml:"nums"`
    Name string `yaml:"name"`
}

func init() {
    plugin.Register(pluginName, &Foo{})
}

func (f *Foo) Type() string {
    return pluginType
}

func (f *Foo) Setup(name string, cfgDec plugin.Decoder) error {
    if err := cfgDec.Decode(f); err != nil {
        return err
    }
    return nil
}
```

插件配置为：

```yaml
plugins:
  foo:
    default:
      nums: [1, 2, 3]
      name: "default_foo"
```

# OWNER
