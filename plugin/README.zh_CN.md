[English](README.md) | 中文

# plugin

tRPC-Go 在设计上遵循了插件化架构理念，通过插件来实现框架核心与各种生态体系的对接，提供了框架的开放性和可扩展性。
`plugin` 包用于管理需要依赖配置进行加载的插件。
而不需要依赖配置的插件相对简单，如 [codec 插件](../codec/README_CN.md)，这里不再讨论。
因此我们将先介绍 `plugin` 包的设计，然后在此基础上阐述如何开发一个需要依赖配置进行加载的插件。

## `plugin` 包的设计

`plugin` 包通过“插件工厂”来管理所有插件的，每个插件都需要注册到插件工厂。
插件工厂采用了两级管理模式：
第一级为插件类型，例如，log 类型, conf 类型， selector 类型等。
第二级为插件名称，例如，conf 的插件有本地文件配置，远程文件配置，本地数据库配置等。

```ascii
                       +-----------------+
                   +---+  Plugin Factory +----+
                   |   +--------+--------+    |
                   |            |             |
               +---v--+      +--v--+     +----v-----+
     +---------+ conf |      | log |     | selector |
     |         +---+--+      +-----+     +----+-----+
     |             |                          |
+----v-----+ +-----v-------+             +----v----+
|local-file| | remote-file |    ......   | polaris |
+----------+ +-------------+             +---------+
```

对于插件的类型，`plugin` 包并没有做限制，你可以自行添加插件类型。


### 常见的插件类型

按照功能划分，框架提供以下5种类型的常见插件：

- 配置：提供获取配置的标准接口，通过从本地文件，配置中心等多种数据源获取配置数据，提供 json，yaml 等多种格式的配置解析，同时框架也提供了 watch 机制，来实现配置的动态更新。
- 日志：提供了统一的日志打印和日志上报标准接口。日志插件通过实现日志上报接口来完成和远程日志系统的对接
- 协议：提供协议编解码相关的标准接口，允许通过插件的方式来扩展业务协议、序列化类型、数据压缩方式等协议处理
- 名字服务：提供包括服务注册，服务发现，策略路由，负载均衡，熔断等标准接口，用于实现服务路由寻址。
- 拦截器：提供通用拦截器接口，用户可以在服务调用的上下文设置埋点，实现例如模调监控，横切日志，链路跟踪，过载保护等功能。


## 如何开发一个需要依赖配置进行加载的插件

开发一个需要依赖配置进行加载的插件通常需要实现插件和配置插件，[可运行的具体的示例](../examples/features/plugin)

### 实现插件

1. 该插件实现 `plugin.Factory` 接口。
```go
// Factory 插件工厂统一抽象 外部插件需要实现该接口，通过该工厂接口生成具体的插件并注册到具体的插件类型里面
type Factory interface {
    // Type 插件的类型 如 selector log config tracing
    Type() string
    // Setup 根据配置项节点装载插件，用户自己先定义好具体插件的配置数据结构
    Setup(name string, configDec Decoder) error
}
```
2. 该插件调用 `plugin.Register` 把自己插件注册到 `plugin` 包。

```
// Register 注册插件工厂 可自己指定插件名，支持相同的实现 不同的配置注册不同的工厂实例
func Register(name string, p Factory)
```

### 配置插件

1. 在 `main` 包中 import 该插件对应的包。
2. 在配置文件中的 `plugins` 字段下面配置该插件。
   配置文件格式为：
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
    local-file:
      # 插件详细配置，具体请参考各个插件的说明
      ....
    remote-file:
      # 插件详细配置，具体请参考各个插件的说明
```

上面定义了 2 个插件类型和 4 个插件。
log 类型的插件下有 logger1 和 logger2 插件。
config 类型的插件下有 local-file 和 remote-file 插件。

### 插件的初始化顺序

tRPC-GO server 调用 `trpc.NewServer()` 函数之后，会读取框架配置文件中“plugins”字段下所有插件配置，然后调用每个插件的“Setup()”函数来完成插件配置的初始化。
一般情况下，插件之间是相互独立的，框架会按随机顺序逐个初始化插件（比如 A 插件依赖 B 插件）。
如果某个插件依赖其它插件可以实现以下方法来声明依赖关系。

```go
// Depender 是 "强依赖" 的接口。 
// 如果插件 a "强烈" 依赖插件 b，那么 b 必须存在， 
// a 将在 b 初始化之后进行初始化。 
type Depender interface { 
    // DependsOn 返回依赖的插件列表。 
    // 列表元素的格式为 "类型-名称"，例如 [ "selector-polaris" ]。 
    DependsOn() []string
}

// FlexDepender 是 "弱依赖" 的接口。 
// 如果插件 a "弱" 依赖插件 b，并且 b 确实存在， 
// 那么 a 将在 b 初始化之后进行初始化。 
type FlexDepender interface { 
    FlexDependsOn() []string
}
```

依赖关系分为强依赖和弱依赖。
强依赖要求被依赖的插件必须存在，不存在框架会 panic。
弱依赖则不会 panic。
框架会先确保所有强依赖都被满足，然后再检查弱依赖。

例如，在下面的例子中，插件的初始化强依赖于 selector 类型的插件 a，弱依赖于 config 类型的插件 b。

```go
func (p *Plugin) DependsOn() []string {
    return []string{"selector-a"}
}
func (p *Plugin) FlexDependsOn() []string {
    return []string{"config-b"}
}
```