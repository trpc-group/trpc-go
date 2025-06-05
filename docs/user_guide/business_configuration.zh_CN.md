## 1 前言

配置管理是微服务治理体系中非常重要的一环，tRPC 框架为业务程序开发提供了一套支持从多种数据源获取配置，解析配置和感知配置变化的标准接口，框架屏蔽了和数据源对接细节，简化了开发。通过本文的介绍，旨在为用户提供以下信息：

- 什么是业务配置，它和框架配置的区别
- 业务配置的一些核心概念（比如：provider，codec...）
- 如何使用标准接口获取业务配置
- 如何感知配置项的变化
- 如何和多种数据源做对接

## 2 概念介绍

### 2.1 什么是业务配置

业务配置是供业务使用的配置，它由业务程序定义配置的格式，含义和参数范围，tRPC 框架并不使用业务配置，也不关心配置的含义。框架仅仅关心如何获取配置内容，解析配置，发现配置变化并告知业务程序。

业务配置和框架配置的区别在于使用配置的主体和管理方式不一样。框架配置是供 tRPC 框架使用的，由框架定义配置的格式和含义。框架配置仅支持从本地文件读取方式，在程序启动时读取配置，用于初始化框架。框架配置不支持动态更新配置，如果需要更新框架配置，则需要重启程序。

而业务配置则不同，业务配置支持从多种数据源获取配置，比如：本地文件，配置中心，数据库等。如果数据源支持配置项事件监听功能，tRPC 框架则提供了机制以实现配置的动态更新。

### 2.2 如何管理业务配置

对于业务配置的管理，我们建议最佳实践是使用配置中心来管理业务配置，使用配置中心有以下优点：

- 避免源代码泄露敏感信息
- 服务动态更新配置
- 多服务共享配置，避免一份配置拥有多个副本
- 支持灰度发布，配置回滚，拥有完善的权限管理和操作日志

业务配置也支持本地文件。对于本地文件，大部分使用场景是客户端作为独立的工具使用，或者程序在开发调试阶段使用。好处在于不需要依赖外部系统就能工作。

### 2.3 什么是多数据源

数据源就获取配置的来源，配置存储的地方。常见的数据源包括：file，etcd，configmap，tconf，rainbow 等。tRPC 框架支持对不同业务配置设定不同的数据源。框架采用插件化方式来扩展对更多数据源的支持。在后面的实现原理章节，我们会详细介绍框架是如何实现对多数据源的支持的。

### 2.4 什么是 Codec

业务配置中的 Codec 是指从配置源获取到的配置的格式，常见的配置文件格式为：yaml，json，toml 等。框架采用插件化方式来扩展对更多解码格式的支持。

## 3 实现原理

为了更好的了解配置接口的使用，以及如何和数据源做对接，我们简单看看配置接口模块是如何实现的。下面这张图是配置模块实现的示意图（非代码实现类图）：

![trpc](https://git.woa.com/trpc-go/trpc-go/raw/master/.resources/user_guide/business_configuration/trpc_cn.png)

图中的 config 接口为业务代码提供了获取配置项的标准接口，每种数据类型都有一个独立的接口，接口支持返回 default 值。

Codec 和 DataProvider 在第 2 节我们已经介绍过，这两个模块都提供了标准接口和注册函数以支持编解码和数据源的插件化。以实现多数据源为例，DataProvider 提供了以下三个标准接口，其中 Read 函数提供了如何读取配置的原始数据（未解码），而 Watch 函数提供了 callback 函数，当数据源的数据发生变化时，框架会执行此 callback 函数。

```go
type DataProvider interface {
    Name() string
    Read(string) ([]byte, error)
    Watch(ProviderCallback)
}
```

最后我们来看看，如何通过指定数据源，解码器来获取一个业务配置项：

```go
import (
    "log/slog"
    
    "git.code.oa.com/trpc-go/trpc-go/config"
)

// 加载 TConf 配置文件：config.WithProvider("tconf")
const configPath = "test.yaml"
c, err := config.Load(configPath, config.WithCodec("yaml"), config.WithProvider("tconf"))
if err != nil {
    slog.Error("loading config failed", "config path", configPath, "error", err)
}
// 读取 String 类型配置
c.GetString("auth.user", "admin")
```

在这个示例中，数据源为 tconf 配置中心，数据源中的业务配置文件为“test.yaml”。当 ConfigLoader 获取到"test.yaml"业务配置时，指定使用 yaml 格式对数据内容进行解码。最后通过`c.GetString("server.app", "default")`函数来获取 test.yaml 文件中`auth.user`这个配置型的值。

## 4 接口使用

本文仅从使用业务配置的角度来介绍相应的接口，如何用户需要开发数据源插件或者 Codec 插件，请参考 [tRPC-Go 开发配置插件](https://iwiki.woa.com/pages/viewpage.action?pageId=261303291 "tRPC-Go 开发配置插件")。具体接口参数请参考 [tRPC-Go API 手册](http://godoc.oa.com/git.woa.com/trpc-go/trpc-go/config "tRPC-Go API 手册")。

tRPC-Go 框架提供了两套接口分别用于“读取配置项”和“监听配置项”

### 4.1 获取配置项

**第一步：选择插件**
在使用配置接口之前需要提前配置好数据源插件，以及插件配置。插件的使用请在 插件生态 中查找。对于 tconf 和七彩石的配置请参考第 5 节。tRPC 框架默认支持本地文件数据源。

**第二步：插件初始化**
由于数据源采用的是插件方式实现的，需要 tRPC 框架在服务端初始化函数中，通过读取“trpc_go.yaml”文件来初始化所有插件。业务配置的读取操作必须在完成`trpc.NewServer()`之后

```go
import (
    trpc "git.code.oa.com/trpc-go/trpc-go"
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
// 加载 TConf 配置文件：config.WithProvider("tconf")
c, err := config.Load("test1.yaml", config.WithCodec("yaml"), config.WithProvider("tconf"))
if err != nil { 
    // handle error
}

// 加载本地配置文件，codec 为 json，数据源为 file
c, err = config.Load("../testdata/auth.yaml", config.WithCodec("json"), config.WithProvider("file"))
if err != nil {
    // handle error
}

// 加载本地配置文件，默认为 codec 为 yaml，数据源为 file
c, err = config.Load("../testdata/auth.yaml")
if err != nil {
    // handle error
}
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

### 4.2 监听配置项

对于 KV 型配置中心（tconf 和七彩石均为 KV 型配置中心），框架提供了 Watch 机制供业务程序根据接收的配置项变更事件，自行定义和执行业务逻辑。监控接口设计如下：

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

下面示例展示了业务程序监控 tconf 上的“test.yaml”文件，打印配置项变更事件并更新配置。

```go
import (
    "sync/atomic"
    ...
)

type yamlFile struct {
    Server struct {
        App string
    }
}

var cfg atomic.Value // 并发安全的 Value

// 使用 trpc-go/config 中 Watch 接口监听 tconf 远程配置变化
c, err := config.Get("tconf").Watch(context.TODO(), "test.yaml")
if err != nil {
    // handle error
}

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

## 5 数据源对接

本地配置，七彩石和 tconf 是常见的 2 种数据源接入模式，本节会详细介绍 tRPC 如何和这三个数据源做对接。对于 tconf 配置中心，后期会逐渐迁移到七彩石。

### 5.1 与本地文件对接

框架默认支持本地配置文件方式。用户无需做特别操作。直接使用第 4 节的接口获取配置项。框架不支持用户监听配置项功能。

### 5.2 与七彩石对接

**第一步：七彩石平台操作**

1. 访问 Web 端控制台 (<http://rainbow.oa.com/>)

2. 新建项目（如果已有项目，跳过），此时浏览器 URL 中间一串即为插件配置中需要的`appid`字段，如：<http://rainbow.oa.com/console/3482e0a7-3a00-401c-9505-7bdb0a12511c/list> 中的 appid 为`3482e0a7-3a00-401c-9505-7bdb0a12511c`

3. 新建分组（如果已有分组，跳过），分组名即为插件配置中的`group`字段

4. 新增配置，并发布配置

**第二步：插件配置**
provider 表示配置所属项目的分组，插件支持从多个 provider 中拉取配置。

| 配置项  | 配置说明  |
| ------------ | ------------ |
| name  | provider 标识，可以使用：config.WithProvider("tconf1")，指定从某个 provider 中拉取配置  |
| appid  | 配置所属的项目  |
| group  | 配置所属的分组  |
| type  | 七彩石数据格式，kv（默认）, table  |
| env_name  | rainbow 多环境配置，如果没有使用多环境特性，不需要配置此项  |
| timeout  | 拉取配置接口超时设置，单位毫秒，不填默认 2 秒  |
| address  | rainbow 服务端地址，内网无需填写，外网使用请咨询 rainbow_helper  |
| uin  | 客户端标识，可选配置  |
| file_cache  | 本地缓存文件设置，可选配置  |
| enable_sign  | 设置签名校验，可选配置，开启时需要设置 user_id、user_key  |
| user_id  | 用户 ID，平台生成。在拉取配置时生成签名，`enable_sign: true`时 必填  |
| user_key  | 用户密钥，平台生成。在拉取配置时生成签名，`enable_sign: true`时 必填  |
| enable_client_provider  | 使用 client provider，可不填，默认为 False  |

请在框架配置文件 `trpc_go.yaml` 中增加对应的插件配置

```yaml
plugins:
    config:
        rainbow: # 七彩石配置中心
            providers:
              - name: rainbow # provider 名字，代码使用如：`config.WithProvider("rainbow")`
                appid: 3482e0a7-3a00-401c-9505-7bdb0a12511c # appid
                group: dev # 配置所属组
                type: kv   # 七彩石数据格式，kv（默认）, table
                env_name: production
                file_cache: /tmp/a.backup
                uin: a3482e0a7
                enable_sign: true
                user_id: 2a9a63844fe24a8aadaxxx5d2f5e903a
                user_key: 599dd5a3480805e22bb6ac22eeaf40d34f8a
                enable_client_provider: true
                timeout: 2000

              - name: rainbow1
                appid: 3482e0a7-3a00-401c-9505-7bdb0a12511c
                group: dev1

```

**第三步：注册插件**

```go
import (
    // 根据插件配置自动注册 rainbow 插件
    _ "git.code.oa.com/trpc-go/trpc-config-rainbow"
)
```

**第四步：完成对接**
tRPC 和七彩石的对接工作已经完成，用户可使用第 3 节的接口进行配置读取操作。

### 5.3 与 tconf 对接
>
> tconf 后期计划迁入到七彩石，数据的迁入会由 tconf 后台来做，对业务透明。

**第一步：tconf 平台操作**

通过 Web 端控制台 <http://tconf.pcg.com/> 在 tconf 系统注册服务并创建配置。

**第二步：插件配置**

provider 表示配置所属 tconf 服务模块的组合 (appid、env_name, namespace)，插件支持从多个 provider 中拉取配置。在 tconf 中，一份配置文件必须归属于某一个 appid、env 下。

| 配置项  |  配置说明 |
| ------------ | ------------ |
| name  |  provider 标识，使用的 provider 中的配置时，可以使用：config.WithProvider("tconf1")  |
| appid  | 配置所属的 appid。可不填，默认会使用 trpc_go.yaml 中，server 下面的 app server  |
| env_name  | 配置所属的环境名。可不填，默认会使用 trpc_go.yaml 中 global 下的 env_name  |
| namespace  | 当前服务运行环境所属的命名空间（Development 或 Production），可不填，默认读取 trpc_go.yaml 中 global 下的 namespace  |
| enable_client_provider  | 使用 client provider，可不填，默认为 False  |

请在 tRPC 框架配置文件 `trpc_go.yaml` 中增加对应的插件配置

```yaml
plugins:
  config:
    tconf:
      providers:
       - name: tconf1
         appid: tconf.config
         env_name: test
         namespace: Development
         enable_client_provider: true

       - name: tconf2
         appid: test.trey.conf
         env_name: test
         namespace: Development

```

**第三步：注册插件**

由于 tconf 插件依赖北极星进行 tconf 服务寻址，所以在注册插件时，需要同时注册 tconf 和北极星

```go
import (
    // 根据插件配置自动注册 tconf 插件
    _ "git.code.oa.com/trpc-go/trpc-config-tconf"
    _ "git.code.oa.com/trpc-go/trpc-naming-polaris" // tconf 插件依赖北极星寻址
)
```

**第四步：完成对接**
tRPC 和 tconf 的对接工作已经完成，用户可使用第 3 节的接口进行配置读取操作。

## 更多问题

请参考 [tRPC 技术咨询](https://iwiki.woa.com/p/491739953) 以寻求帮助
