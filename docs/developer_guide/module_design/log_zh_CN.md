[TOC]




# 背景

服务中通过日志来定位服务流水、异常、错误是必不可少的，log package 用来支持写服务 log。

log 通常需要输出到标准输出、日志文件或者其他日志收集系统，如 syslog 或者公司的一些远程日志平台等。
log 中通过插件化设计对日志信息的多端输出予以了支持，针对常见的开源日志库也进行了对比、选择，最终选择了使用广泛、性能最高的 zapcore 作为基础来予以封装。

# 原理

先看下 log 整体设计：
![log 整体设计](/.resources/developer_guide/module_design/log/log.png)

log 定义了日志输出的通用接口，支持 Trace、Debug、Info、Warning、Error、Fatal 等多种级别。
对应的在 123 的配置字段值和字段位置为：trace error info debug。
plugins:
-log:
--default:
---level: info
trpc-go 提供了日志输出的默认实现，基于 [uber-go/zap](https://github.com/uber-go/zap) 包，支持应用运行时日志级别动态调整。具体实现可参考：https://git.woa.com/trpc-go/trpc-go/blob/master/log/zaplogger.go

# 配置

log 的配置默认支持输出到控制台和输出到文件。

``` go
type Config struct {
    Outputs map[string]OutputConfig // writer name => output config
}
```
## OutputConfig

OutputConfig 定义了日志输出端、日志输出格式、日志输出级别等信息，框架默认加载 trpc-go.yaml 配置文件的信息。

``` go
type OutputConfig struct {
    // Writer 日志输出端 (console, file)
    Writer      string
    WriteConfig WriteConfig `yaml:"writer_config"`
    // Formatter 日志输出格式 (console, json)
    Formatter    string
    FormatConfig FormatConfig `yaml:"formatter_config"`
    // Level 控制日志级别
    Level string
}
```
# 使用方式

WithFields 支持设置一些业务自定义数据到每条 log 里，比如 uid 等。
``` go
m := make(map[string]string)
m["uid"] = "10012"
log.WithFields(m)
```
输出日志
``` go
log.Trace("helloworld")
log.Debug("helloworld")
log.Info("helloworld")
log.Warning("helloworld")
log.Error("helloworld")
```
## 染色日志

利用框架本身的消息染色机制，可以用来打印染色日志。

``` go
msg := trpc.Message(ctx)
msg.WithDyeing(true) // 给请求设置染色标记，此标记会自动下游传递
msg.WithDyeingKey(key) // 设置染色key，设置的染色key可以作为关键字用于染色日志查询
```
下面介绍一下目前在 pcg 用的比较广的两个染色日志系统，天机阁和 metis。
### 天机阁日志
123 平台上部署的服务，默认已经开启天机阁功能，如果需要打印天机阁染色日志，可以使用[tlog 库](https://git.woa.com/trpc-go/trpc-opentracing-tjg/tree/master/tlog/example)。

``` go
// 设置查询key，对于染色或者采样的请求，用户可以在天机阁通过查询key查询到trace数据
uin := "10000"
tlog.SetDyeingKey(ctx, uin)
// 白名单用户，开启染色
// 对于染色采样请求相关的日志，无视日志级别都上报到天机阁
// 对于概率采样请求相关的日志，按照日志级别上报到天机阁
if uin == "10000" {
    tlog.SetDyeing(ctx)
}
tlog.Tracef(ctx, "helloworrld")
tlog.Debugf(ctx, "helloworrld")
tlog.Infof(ctx, "helloworrld")
tlog.Warningf(ctx, "helloworrld")
tlog.Errorf(ctx, "helloworrld")
```

天机阁查询页面：
http://tjg.oa.com

### metis 染色日志

metis 染色日志系统在 pcg 内部用的也是比较广，如果需要接入 metis 日志，可以参考一下[nlog](https://git.woa.com/nfa/nfalib/tree/master/nlog)库，这个库集成了天机阁/metis/本地日志三种日志上报，如果只需要 metis 日志上报，可以自行剥离出来。

metis 染色日志系统地址：
http://metis.pcg.com
![metis 染色日志](/.resources/developer_guide/module_design/log/metis.png)
