[English](README.md) | 中文

# 前言

配置管理是微服务治理体系中非常重要的一环，tRPC 框架为业务程序开发提供了一套支持从多种数据源获取配置，解析配置和感知配置变化的标准接口，框架屏蔽了和数据源对接细节，简化了开发。通过本文的介绍， 旨在为用户提供以下信息：
- 什么是业务配置，它和框架配置的区别
- 业务配置的一些核心概念（比如： provider，codec...）
- 如何使用标准接口获取业务配置
- 如何感知配置项的变化

# 概念介绍
## 什么是业务配置
业务配置是供业务使用的配置，它由业务程序定义配置的格式，含义和参数范围，tRPC 框架并不使用业务配置，也不关心配置的含义。框架仅仅关心如何获取配置内容，解析配置，发现配置变化并告知业务程序。

业务配置和框架配置的区别在于使用配置的主体和管理方式不一样。框架配置是供 tRPC 框架使用的，由框架定义配置的格式和含义。框架配置仅支持从本地文件读取方式，在程序启动是读取配置，用于初始化框架。框架配置不支持动态更新配置，如果需要更新框架配置，则需要重启程序。

而业务配置则不同，业务配置支持从多种数据源获取配置，比如：本地文件，配置中心，数据库等。如果数据源支持配置项事件监听功能，tRPC 框架则提供了机制以实现配置的动态更新。

## 如何管理业务配置
对于业务配置的管理，我们建议最佳实践是使用配置中心来管理业务配置，使用配置中心有以下优点：
- 避免源代码泄露敏感信息
- 服务动态更新配置
- 多服务共享配置，避免一份配置拥有多个副本
- 支持灰度发布，配置回滚，拥有完善的权限管理和操作日志

业务配置也支持本地文件。对于本地文件，大部分使用场景是客户端作为独立的工具使用，或者程序在开发调试阶段使用。好处在于不需要依赖外部系统就能工作。

## 什么是多数据源
数据源就获取配置的来源，配置存储的地方。常见的数据源包括：file，etcd，configmap，env 等。tRPC 框架支持对不同业务配置设定不同的数据源。框架采用插件化方式来扩展对更多数据源的支持。在后面的实现原理章节，我们会详细介绍框架是如何实现对多数据源的支持的。

## 什么是 Codec
业务配置中的 Codec 是指从配置源获取到的配置的格式，常见的配置文件格式为：yaml，json，toml 等。框架采用插件化方式来扩展对更多解码格式的支持。

# 实现原理
为了更好的了解配置接口的使用，以及如何和数据源做对接，我们简单看看配置接口模块是如何实现的。下面这张图是配置模块实现的示意图（非代码实现类图）：

![trpc](/.resources/user_guide/business_configuration/trpc_cn.png)

图中的 config 接口为业务代码提供了获取配置项的标准接口，每种数据类型都有一个独立的接口，接口支持返回 default 值。

Codec 和 DataProvider 这两个模块都提供了标准接口和注册函数以支持编解码和数据源的插件化。以实现多数据源为例，DataProvider 提供了以下三个标准接口，其中 Read 函数提供了如何读取配置的原始数据（未解码），而 Watch 函数提供了 callback 函数，当数据源的数据发生变化时，框架会执行此 callback 函数。
```go
type DataProvider interface {
    Name() string
    Read(string) ([]byte, error)
    Watch(ProviderCallback)
}
```

最后我们来看看，如何通过指定数据源，解码器来获取一个业务配置项：
```go
// 加载 etcd 配置文件：config.WithProvider("etcd")
c, _ := config.Load("test.yaml", config.WithCodec("yaml"), config.WithProvider("etcd"))
// 读取 String 类型配置
c.GetString("auth.user", "admin")
```
在这个示例中，数据源为 etcd 配置中心，数据源中的业务配置文件为“test.yaml”。当 ConfigLoader 获取到"test.yaml"业务配置时，指定使用 yaml 格式对数据内容进行解码。最后通过`c.GetString("server.app", "default")`函数来获取 test.yaml 文件中`auth.user`这个配置型的值。

# 接口使用
本文仅从使用业务配置的角度来介绍相应的接口，如何用户需要开发数据源插件或者 Codec 插件，请参考 [tRPC-Go 开发配置插件](/docs/developer_guide/develop_plugins/config.zh_CN.md)。

tRPC-Go 框架提供了两套接口分别用于 “读取配置项” 和 “监听配置项”
## 获取配置项
**第一步：选择插件**
在使用配置接口之前需要提前配置好数据源插件，以及插件配置。tRPC 框架默认支持本地文件数据源。

**第二步：插件初始化**
由于数据源采用的是插件方式实现的，需要 tRPC 框架在服务端初始化函数中，通过读取“trpc_go.yaml”文件来初始化所有插件。业务配置的读取操作必须在完成`trpc.NewServer()`之后
```go
import (
    trpc "trpc.group/trpc-go/trpc-go"
)

// 实例化 server 时会初始化插件系统，所有配置读取操作需要在此之后
trpc.NewServer()
```

**第三步：加载配置**
从数据源加载配置文件，返回 config 数据结构。可指定数据源类型和编解码格式，框架默认为“file”数据源和“yaml”编解码。接口定义为：
```go
// 加载配置文件：path 为配置文件路径
func Load(path string, opts ...LoadOption) (Config, error)
// 更改编解码类型，默认为“yaml”格式
func WithCodec(name string) LoadOption
// 更改数据源，默认为“file”
func WithProvider(name string) LoadOption
```

示例代码为：
```go
// 加载 etcd 配置文件：config.WithProvider("etcd")
c, _ := config.Load("test1.yaml", config.WithCodec("yaml"), config.WithProvider("etcd"))

// 加载本地配置文件，codec 为 json，数据源为 file
c, _ := config.Load("../testdata/auth.yaml", config.WithCodec("json"), config.WithProvider("file"))

// 加载本地配置文件，默认为 codec 为 yaml，数据源为 file
c, _ := config.Load("../testdata/auth.yaml")
```

**第四步：获取配置项**
从 config 数据结构中获取指定配置项值。支持设置默认值，框架提供以下标准接口：
```go
// Config 配置通用接口
type Config interface {
    Load() error
    Reload()
    Get(string, interface{}) interface{}
    Unmarshal(interface{}) error
    IsSet(string) bool
    GetInt(string, int) int
    GetInt32(string, int32) int32
    GetInt64(string, int64) int64
    GetUint(string, uint) uint
    GetUint32(string, uint32) uint32
    GetUint64(string, uint64) uint64
    GetFloat32(string, float32) float32
    GetFloat64(string, float64) float64
    GetString(string, string) string
    GetBool(string, bool) bool
    Bytes() []byte
}
```

示例代码为：
```go
// 读取 bool 类型配置
c.GetBool("server.debug", false)

// 读取 String 类型配置
c.GetString("server.app", "default")
```

## 监听配置项
对于 KV 型配置中心，框架提供了 Watch 机制供业务程序根据接收的配置项变更事件，自行定义和执行业务逻辑。监控接口设计如下：
```go

// Get 根据名字使用 kvconfig
func Get(name string) KVConfig

// KVConfig kv 配置
type KVConfig interface {
    KV
    Watcher
    Name() string
}

// 监控接口定义
type Watcher interface {
    // Watch 监听配置项 key 的变更事件
    Watch(ctx context.Context, key string, opts ...Option) (<-chan Response, error)
}

// Response 配置中心响应
type Response interface {
    // Value 获取配置项对应的值
    Value() string
    // MetaData 额外元数据信息
    // 配置 Option 选项，可用于承载不同配置中心的额外功能实现，例如 namespace,group, 租约等概念
    MetaData() map[string]string
    // Event 获取 Watch 事件类型
    Event() EventType
}

// EventType 监听配置变更的事件类型
type EventType uint8
const (
    // EventTypeNull 空事件
    EventTypeNull EventType = 0
    // EventTypePut 设置或更新配置事件
    EventTypePut EventType = 1
    // EventTypeDel 删除配置项事件
    EventTypeDel EventType = 2
)
```

下面示例展示了业务程序监控 etcd 上的“test.yaml”文件，打印配置项变更事件并更新配置。
```go
import (
    "sync/atomic"
    // ...
)

type yamlFile struct {
    Server struct {
        App string
    }
}

var cfg atomic.Value // 并发安全的 Value

// 使用 trpc-go/config 中 Watch 接口监听 etcd 远程配置变化
c, _ := config.Get("etcd").Watch(context.TODO(), "test.yaml")

go func() {
    for r := range c {
        yf := &yamlFile{}
        fmt.Printf("event: %d, value: %s", r.Event(), r.Value())

        if err := yaml.Unmarshal([]byte(r.Value()), yf); err == nil {
            cfg.Store(yf)
        }
    }
}()

// 当配置初始化完成后，可以通过 atomic.Value 的 Load 方法获得最新的配置对象
cfg.Load().(*yamlFile)
```

# 数据源实现

参考：[trpc-ecosystem/go-config-etcd](https://github.com/trpc-ecosystem/go-config-etcd)
