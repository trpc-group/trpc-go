[TOC]

# 前言

日志是软件开发中的一个重要组成部分，好的日志是我们分析软件运行状态的一个重要信息来源。tRPC-Go 框架提供为业务提供了一套的日志打印，收集和上报的解决方案。通过本文的介绍，旨在为用户提供以下信息：

- 框架提供了哪些日志功能？
- 框架如何实现这些功能的？
- 如何使用标准日志接口？
- 日志配置该如何配？
- 如何和常见的日志系统对接？不配置默认异步模式

# 功能介绍

## 功能概述

如图所示，tRPC-Go 日志模块包括两方面的实现：日志打印编程接口 和 与日志输出端对接。在实现中引入了 Logger 和 Writer 两个概念，提供了 3 套日志打印函数。

![ 'plug-in Logger'](/.resources/user_guide/log/logger.png)

Logger 和 Writer 分别负责日志打印和于日志服务系统对接，它们都采用了插件化编程，可定制化开发。关于如何开发日志插件，请参考[这里](https://git.woa.com/trpc-go/trpc-wiki/blob/main/developer_guide/develop_plugins/log_CN.md) 。Logger 和 Writer 的定义如下：

- Logger 用于负责实现日志打印相关的通用接口，Logger 采用插件化编程，支持多 Logger 功能，用户可以自行选择 logger 来实现差异化日志打印和收集。通过 writer 插件和日志后端进行对接。
- Writer 又称为“日志输出端”，用于负责日志的收集上报功能，包括日志上报格式和与后端日志系统对接等。每个 logger 都可以有多个 Writer，例如：log.Debug 可以同时输出到 console 和本地日志文件。Writer 采用插件化编程，可灵活扩展。

框架提供了 3 套日志打印函数：

- 使用默认 Logger 打印日志：使用最频繁的一种方式。直接使用默认 Logger 进行日志打印，方便简单
- 使用基于 Context Logger 打印日志：为特定场景提供指定 logger，并保存在 context 里，后续使用当前 context logger 进行日志打印。这种方式特别适合 RPC 调用模式：当服务收到 RPC 请求时，为 ctx 设置 logger，并附加此次请求相关的 field 信息，此 RPC 调用的后续日志上报都会带上之前设定的 field 信息
- 指定 Logger 日志打印：可以用于用户自行选择 logger，并调用 logger 的接口函数实现日志打印

## 统一日志打印接口

首先框架参考业界标准对日志打印级别按由低到高的顺序进行了如下划分：

|   级别    | 描述                                                                                                                                                               |
| :-------: | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **trace** | 该级别日志的主要作用是对系统特定事件的每一步的运行状态进行精确的记录                                                                                               |
| **debug** | 指出细粒度信息事件对调试应用程序是非常有帮助的，主要用于开发过程中打印一些运行信息                                                                                 |
| **info**  | 消息在粗粒度级别上突出强调应用程序的运行过程。打印一些你感兴趣的或者重要的信息，这个可以用于生产环境中输出程序运行的一些重要信息，但是不能滥用，避免打印过多的日志 |
| **warn**  | 表明会出现潜在错误的情形，有些信息不是错误信息，但是也要给程序员一些提示                                                                                           |
| **error** | 指出虽然发生错误事件，但仍然不影响系统的继续运行。打印错误和异常信息，如果不想输出太多的日志，可以使用这个级别                                                     |
| **fatal** | 指出每个严重的错误事件将会导致应用程序的退出。这个级别比较高了。重大错误，这种级别你可以直接停止程序了                                                             |

框架为每个级别的日志函数提供了类似于`fmt.Print()`和`fmt.Printf()`两种风格的打印函数。函数命名方式为：

```go
// fmt.Print 函数风格
Debug(args ...interface{})
// fmt.Printf 函数风格
Debugf(format string, args ...interface{})
```

框架提供了为每条日志增加自定义字段的能力。增加的字段以 kv 的方式成对出现。比如：

```go
logger := log.WithFields("id","x100-1","city","sz")
logger.Debug("hello, world")
// 输出端设定以 json 格式存储的话，打印结果为：
// {"L":"INFO","T":"2020-12-10 11:19:52.104","C":"hello/main.go:29","M":"hello, world","id":"x100-1","city":"sz"}
```

## 多 Logger 支持

tRPC-Go 支持同时存在多个 logger，每个 logger 设置不同的日志级别，打印格式，上报输出后端，业务可以为不同的业务场景设置不同的 logger。比如

- 不同的业务模块使用不同的日志文件存储
- 不同的事件，基于事件的关注度不同，采用不同的日志级别收集日志

多 Logger 功能大大增加了日志使用的灵活性。

### 注册 logger 插件

首先需要在 main 函数入口注册插件

```go
import (
    "git.code.oa.com/trpc-go/trpc-go/log"
    "git.code.oa.com/trpc-go/trpc-go/plugin"
)
func main() {
    // 注意：plugin.Register 要在 trpc.NewServer 之前执行
    plugin.Register("custom", log.DefaultLogFactory)
    s := trpc.NewServer()
}
```

### 配置 logger

配置文件定义自己的 logger，比如配置 custom：

```go
plugins:
  log:  # 所有日志配置
    default:  # 默认日志配置，log.Debug("xxx")
      - writer: console  # 控制台标准输出 默认
        level: debug  # 标准输出日志的级别
    custom:  # 业务自定义的logger配置，名字随便定，每个服务可以有多个logger，可使用 log.Get("custom").Debug("xxx") 打日志
      - writer: file  # 业务自定义的core配置，名字随便定
        caller_skip: 1  # 用于定位日志的调用处
        level: debug  # 业务自定义core输出的级别
        writer_config:  # 本地文件输出具体配置
          filename: ../log/trpc1.log  # 本地文件滚动日志存放的路径
```

配置文件中对于 `caller_skip` 的疑问见「关于 `caller_skip` 的说明」这一节。

### 获取 logger

框架提供了以下两种方式供业务来选择 logger：

#### 方法一：直接指定 logger

```go
// 使用名称为“custom”的 logger
// 注意：log.Get 得在 trpc.NewServer 之后执行，因为插件的加载是在 trpc.NewServer 中进行的
log.Get("custom").Debug("message")
```

#### 方法二：为 context 指定 logger，使用 context 类型日志接口

```go
// 设置 ctx 上下文的 logger 为 custom
trpc.Message(ctx).WithLogger(log.Get("custom"))
// 使用“Context”类型的日志接口
log.DebugContext(ctx, "custom log msg")
```

### 多 Writer 支持

对于同一个 logger，框架也提供了日志同时输出到多个日志输出后端（简称“writer”）的能力。系统默认支持 **"console"** 和 **"file"** 两种后端。框架支持通过插件的方式来支持远程日志系统，比如 鹰眼，智研日志等。

多日志输出后端的功能具体包括：

- 日志同时上报到多个输出后端，比如同时打印到 console 和本地日志文件
- 为每个输出后端单独设置日志级别：比如 debug 级别以上日志打 console，warn 级别以上日志保存到日志文件
- 为每个输出后端单独设置日志格式（console，json 等），日志字段名称
- 为“file”类型后端提供滚动日志功能，包括按文件大小或者时间来分割日志文件

这里需要强调的是：**日志的打印设置是以 Writer 粒度来配置的**。用户需要为每个输出端做配置，比如：debug 级别以上日志打 console，warn 级别以上日志保存到日志文件，那就必须要分别配置 console 和 file 这两个 Writer。

## 日志配置

### 配置结构

首先我们来看看日志配置的总体结构。由于日志模块对于 logger 和 writer 的实现都是采用插件的方式来实现的，所以日志配置都必须要放到“plugins”区域内。

```yaml
plugins:
  log:
    default:
      - writer: console
        ...
      - writer: file
        ...
    logger1:
      - writer: console
        ...
      - writer: atta
        ...
    logger2:
      - writer: file
        ...
      - writer: file
        ...
```

上面的配置体现了多 logger 和多 writer 功能。“default”，“logger1”，“logger2”表示 logger 的名称。每个 logger 的配置是一个 writer 的数组。“default”是系统默认的 logger，在不做任何日志配置时，日志默认打 console，日志级别为 debug，打印方式为文本格式，其对应的配置文件为：

```yaml
plugins:
  log:
    default:
      - writer: console
        level: debug
        formatter: console
```

对于 Writer 的配置参数，设计如下：

| 配置项                  | 配置项           |  类型  | 默认值 | 是否废弃 | 配置解释                                                                                             |
| ----------------------- | ---------------- | :----: | :----: | :------: | ---------------------------------------------------------------------------------------------------- |
| writer                  | writer           | string |        |    否    | 必填 日志 Writer 插件的名字，框架默认支持“file，console”                                             |
| writer                  | writer_config    |  对象  |  nil   |    否    | 当日志 Writer 为“file”时才需要设置                                                                   |
| writer                  | formatter        | string |   “”   |    否    | 日志打印格式，支持“console”和“json”，为空时默认设置为“console”                                       |
| writer                  | formatter_config |  对象  |  nil   |    否    | 日志输出时 zapcore Encoder 配置，为空时为参考 formatter_config 的默认值                              |
| writer                  | remote_config    |  对象  |  nil   |    否    | 远程日志格式 配置格式业务随便定 由第三方组件自己注册，具体配置参考各日志插件文档                     |
| writer                  | level            | string |        |    否    | 必填 日志级别大于等于设置级别时，输出到 writer 后端 取值范围：trace，debug，info，warn，error，fatal |
| writer                  | caller_skip      |  int   |   2    |    否    | 用于控制 log 函数嵌套深度，不填或者输入 0 时，默认为 2                                              |
| writer.formatter_config | time_fmt         | string |   ""   |    否    | 日志输出时间格式，空默认为"2006-01-02 15:04:05.000"                                                  |
| writer.formatter_config | time_key         | string |   ""   |    否    | 日志输出时间在以 Json 输出时的 key 的名称，默认为"T", 用 "none" 来禁用这一字段                                                |
| writer.formatter_config | level_key        | string |   ""   |    否    | 日志级别在以 Json 输出时的 key 的名称，默认为"L", 用 "none" 来禁用这一字段                                                    |
| writer.formatter_config | name_key         | string |   ""   |    否    | 日志名称在以 Json 输出时的 key 的名称，默认为"N", 用 "none" 来禁用这一字段                                                    |
| writer.formatter_config | caller_key       | string |   ""   |    否    | 日志输出调用者在以 Json 输出时的 key 的名称，默认为"C", 用 "none" 来禁用这一字段                                                |
| writer.formatter_config | message_key      | string |   ""   |    否    | 日志输出消息体在以 Json 输出时的 key 的名称，默认为"M", 用 "none" 来禁用这一字段                                                 |
| writer.formatter_config | stacktrace_key   | string |   ""   |    否    | 日志输出堆栈在以 Json 输出时的 key 的名称，默认为"S", 用 "none" 来禁用这一字段                                                  |
| writer.writer_config    | log_path         | string |        |    否    | 必填 日志路径名，例如：/usr/local/trpc/log/                                                          |
| writer.writer_config    | filename         | string |        |    否    | 必填 日志文件名，例如：trpc.log                                                                      |
| writer.writer_config    | write_mode       |  int   |   0    |    否    | 日志写入模式，1-同步，2-异步，3-极速 (异步丢弃), 不配置默认极速模式，即异步丢弃                       |
| writer.writer_config    | roll_type        | string |   ""   |    否    | 文件滚动类型，"size"按大小分割文件，"time"按时间分割文件，为空时默认按大小分割                       |
| writer.writer_config    | max_age          |  int   |   0    |    否    | 日志最大保留时间，为 0 表示不清理旧文件                                                              |
| writer.writer_config    | max_backups      |  int   |   0    |    否    | 日志最大文件数，为 0 表示不删除多余文件                                                              |
| writer.writer_config    | compress         |  bool  | false  |    否    | 日志文件是否压缩，默认不压缩                                                                         |
| writer.writer_config    | max_size         | string |   ""   |    否    | 按大小分割时才有效，日志文件最大大小（单位 MB），为 0 表示不按大小滚动                               |
| writer.writer_config    | time_unit        | string |   ""   |    否    | 按时间分割时才有效，按时间分割文件的时间单位，支持 year/month/day/hour/minute, 默认值为 day          |

## 使用 console

当 writer 设置为“console”时，代表日志打终端。配置参考示例如下：

```yaml
plugins:
  log:  # 所有日志配置
    default:  # 默认日志配置，log.Debug("xxx")
      - writer: console  # 控制台标准输出 默认
        level: debug  # 标准输出日志的级别
        formatter: json  # 标准输出日志的格式
        formatter_config:
          time_fmt: 2006-01-02 15:04:05  # 日志时间格式。"2006-01-02 15:04:05"为常规时间格式，"seconds"为秒级时间戳，"milliseconds"为毫秒时间戳，"nanoseconds"为纳秒时间戳
          time_key: Time  # 日志时间字段名称，不填默认"T"，填 "none" 可禁用此字段
          level_key: Level  # 日志级别字段名称，不填默认"L"，填 "none" 可禁用此字段
          name_key: Name  # 日志名称字段名称，不填默认"N"，填 "none" 可禁用此字段
          caller_key: Caller  # 日志调用方字段名称，不填默认"C"，填 "none" 可禁用此字段
          message_key: Message  # 日志消息体字段名称，不填默认"M"，填 "none" 可禁用此字段
          stacktrace_key: StackTrace  # 日志堆栈字段名称，不填默认"S"，填 "none" 可禁用此字段
```

## 使用本地日志

当 writer 设置为“file”时，代表日志打本地日志文件。日志按时间进行文件滚动存放的配置示例如下：

```yaml
plugins:
  log:  # 所有日志配置
    default:  # 默认日志配置，log.Debug("xxx")
      - writer: file  # 本地文件日志
        level: info  # 本地文件滚动日志的级别
        formatter: json  # 标准输出日志的格式
        formatter_config:
          time_fmt: 2006-01-02 15:04:05  # 日志时间格式。"2006-01-02 15:04:05"为常规时间格式，"seconds"为秒级时间戳，"milliseconds"为毫秒时间戳，"nanoseconds"为纳秒时间戳
          time_key: Time  # 日志时间字段名称，不填默认"T"，填 "none" 可禁用此字段
          level_key: Level  # 日志级别字段名称，不填默认"L"，填 "none" 可禁用此字段
          name_key: Name  # 日志名称字段名称，不填默认"N"，填 "none" 可禁用此字段
          caller_key: Caller  # 日志调用方字段名称，不填默认"C"，填 "none" 可禁用此字段
          message_key: Message  # 日志消息体字段名称，不填默认"M"，填 "none" 可禁用此字段
          stacktrace_key: StackTrace  # 日志堆栈字段名称，不填默认"S"，填 "none" 可禁用此字段
        writer_config:
          log_path: /tmp/log/
          filename: trpc_size.log  # 本地文件滚动日志存放的路径
          write_mode: 2  # 日志写入模式，1-同步，2-异步，3-极速 (异步丢弃), 不配置默认异步模式
          roll_type: time  # 文件滚动类型，time 为按时间滚动
          max_age: 7  # 最大日志保留天数
          max_backups: 10  # 最大日志文件数
          time_unit: day  # 滚动时间间隔，支持：minute/hour/day/month/year
```

日志按文件大小进行文件滚动存放的配置示例如下：

```yaml
plugins:
  log:  # 所有日志配置
    default:  # 默认日志配置，log.Debug("xxx")
      - writer: file  # 本地文件日志
        level: info  # 本地文件滚动日志的级别
        formatter: json  # 标准输出日志的格式
        formatter_config:
          time_fmt: 2006-01-02 15:04:05  # 日志时间格式。"2006-01-02 15:04:05"为常规时间格式，"seconds"为秒级时间戳，"milliseconds"为毫秒时间戳，"nanoseconds"为纳秒时间戳
          time_key: Time  # 日志时间字段名称，不填默认"T"，填 "none" 可禁用此字段
          level_key: Level  # 日志级别字段名称，不填默认"L"，填 "none" 可禁用此字段
          name_key: Name  # 日志名称字段名称，不填默认"N"，填 "none" 可禁用此字段
          caller_key: Caller  # 日志调用方字段名称，不填默认"C"，填 "none" 可禁用此字段
          message_key: Message  # 日志消息体字段名称，不填默认"M"，填 "none" 可禁用此字段
          stacktrace_key: StackTrace  # 日志堆栈字段名称，不填默认"S"，填 "none" 可禁用此字段
        writer_config:
          log_path: /tmp/log/
          filename: trpc_size.log  # 本地文件滚动日志存放的路径
          write_mode: 2  # 日志写入模式，1-同步，2-异步，3-极速 (异步丢弃), 不配置默认异步模式
          roll_type: size  # 文件滚动类型，size 为按大小滚动
          max_age: 7  # 最大日志保留天数
          max_backups: 10  # 最大日志文件数
          compress: false  # 日志文件是否压缩
          max_size: 10  # 本地文件滚动日志的大小 单位 MB
```

## 使用远程日志

对于远程日志，需要配置“remote_config”，其配置格式由各个 writer 插件自行设计。请先在 [插件生态](todo) 中选择需要对接的日志插件，并参考插件链接进行配置。下面对常见的几种远程日志的配置为例进行配置示例。

[鹰眼](https://git.woa.com/trpc-go/trpc-log-atta) 插件的日志配置示例如下：

```yaml
plugins:
  log:  # 日志配置 支持多个日志 可通过 log.Get("xxx").Debug 打日志
    default:  # 默认日志的配置，每个日志可支持多输出
      - writer: atta  # atta 远程日志输出
        level: debug  # 远程日志的级别
        remote_config:  # 远程日志配置，业务自定义结构，每一种远程日志都有自己独立的配置
          atta_id: '05e00006180'  # atta id 每个业务自己申请。使用之前需要保证机器有安装 atta agent
          atta_token: '6851146865'  # atta token 业务自己申请
          auto_escape: false  # [可选，默认 false] atta 上报内容是否转义，性能原因默认关闭，true 开启
          attaobj_size: 3  # [可选，默认 3] 日志异步上报，消费 log 的 goroutinue 数，一个 goroutinue 一个 att obj，避免竞态
          channel_block: false  # [可选，默认 false] 日志异步上报，写管道是否阻塞。
          channel_capacity: 10000  # [可选，默认 10000] 日志管道容量
          enable_batch: false  # [可选，默认 false] 是否缓存批量发送日志，
          send_internal: 1000  # [可选，默认 1000] enable_batch 为 true 时有效，缓存批量发送时间间隔，单位 ms
          message_key: msg  # 日志打印包体的对应 atta 的 field
          level_key: level  # [可选，默认为空] 日志级别对应的 atta 字段，不需可不配置
          atta_warning: false  # [可选，默认 false] 日志级别映射，鹰眼告警开启
          field:  # [可选，不配置从远端拉取] 申请 atta id 时，业务自己定义的表结构字段，顺序必须一致
            - msg
            - uid
            - cmd
            - level
```

[智研日志](https://git.woa.com/trpc-go/trpc-log-zhiyan) 的日志配置示例如下：

```yaml
plugins:
  log:  # 日志配置 支持多个日志 可通过 log.Get("xxx").Debug 打日志
    default:  # 默认日志的配置，每个日志可支持多输出
      - writer: zhiyan  # zhiyan 远程日志输出
        level: debug  # 远程日志的级别
        remote_config:  # 远程日志配置，业务自定义结构，每一种远程日志都有自己独立的配置
          report_topic: 'fb-1ecfc3gefdca1f8'  # 智研日志 topic，需要每个业务自己申请
          report_proto: 'tcp'  # [可选，默认 tcp]   日志上报协议，支持 tcp 和 udp
          host: '1.1.1.1'  # [可选，默认为本端上报 IP] 日志上报的 Host 字段
          zhiyanobj_size: 3  # [可选，默认 3] 日志异步上报，消费 log 的 goroutinue 数，一个 goroutinue 一个 obj，避免竞态
          channel_block: false  # [可选，默认 false] 日志异步上报，写管道是否阻塞。
          channel_capacity: 10000  # [可选，默认 10000] 日志管道容量
```

# 接口使用

## 默认 Logger 日志函数

默认 logger 的名称固定为“default”，默认打印级别为 debug，打印 console，打印格式为文本格式。配置可以通过在框架文件中 plugins.log 中修改。使用默认 Logger 打印日志的函数如下，函数风格与“fmt.Print()”和"fmt.Printf()"保持一致。

```go
// 对默认 Logger 提供“fmt.Print()”风格的函数接口
func Trace(args ...interface{})
func Debug(args ...interface{})
func Info(args ...interface{})
func Warn(args ...interface{})
func Error(args ...interface{})
func Fatal(args ...interface{})
// 对默认 Logger 提供“fmt.Printf()”风格的函数接口
// 格式化打印遵循 fmt.Printf() 标准
func Tracef(format string, args ...interface{})
func Debugf(format string, args ...interface{})
func Infof(format string, args ...interface{})
func Warnf(format string, args ...interface{})
func Errorf(format string, args ...interface{})
func Fatalf(format string, args ...interface{})

```

同时系统也提供了默认 Logger 的管理函数，包括获取，设置默认 logger，设置 logger 的打印级别。

```go
// 通过名称获取 Logger
func Get(name string) Logger
// 设置指定 Logger 为默认 Logger
func SetLogger(logger Logger)
// 设置默认 Logger 下指定 writer 的日志级别，output 为 writer 数组下标 "0" "1" "2"
func SetLevel(output string, level Level)
// 获取默认 Logger 下指定 writer 的日志级别，output 为 writer 数组下标 "0" "1" "2"
func GetLevel(output string) Level
```

## Context 类型日志函数

对于 context 类型日志的使用，每个 context logger 必须独占一个 logger，确保 context logger 的配置不被串改。框架提供了`WithFields()` 和 `WithFieldsContext()`来继承父 logger 配置，并生成新的 logger。

```go
// 以默认 Logger 为基础，创建新 Logger，并且在新 Logger 的日志打印中添加 fields。
// fields 字段必须 kv 成对出现，比如：logger := log.WithFields("key1","value1")
func WithFields(fields ...string) Logger
// 以当前 context 下 Logger 为基础，创建新 Logger，并且在新 Logger 的日志打印中添加 fields。
// fields 字段必须 kv 成对出现，比如：logger := log.WithFields("key1","value1")
func WithFieldsContext(ctx context.Context, fields ...string) Logger {
```

然后通过下面函数为 context 设置 Logger：

```go
logger := ...
trpc.Message(ctx).WithLogger(logger)
```

context 级别的日志打印函数和默认 logger 打印日志的函数类似：

```go
// 对 Context Logger 提供“fmt.Print()”风格的函数
func TraceContext(args ...interface{})
func DebugContext(args ...interface{})
func InfoContext(args ...interface{})
func WarnContext(args ...interface{})
func ErrorContext(args ...interface{})
func FatalContext(args ...interface{})
// 对 Context Logger 提供“fmt.Printf()”风格的函数
// 格式化打印遵循 fmt.Printf() 标准
func TraceContextf(format string, args ...interface{})
func DebugContextf(format string, args ...interface{})
func InfoContextf(format string, args ...interface{})
func WarnContextf(format string, args ...interface{})
func ErrorContextf(format string, args ...interface{})
func FatalContextf(format string, args ...interface{})
```

## 指定 Logger 日志接口

同时，框架也提供了函数供用户自行选择 logger。对于每一个 Logger 都实现了 Logger Interface。接口定义为：

```go
type Logger interface {
    // 接口提供“fmt.Print()”风格的函数
    Trace(args ...interface{})
    Debug(args ...interface{})
    Info(args ...interface{})
    Warn(args ...interface{})
    Error(args ...interface{})
    Fatal(args ...interface{})
    // 接口提供“fmt.Printf()”风格的函数
    Tracef(format string, args ...interface{})
    Debugf(format string, args ...interface{})
    Infof(format string, args ...interface{})
    Warnf(format string, args ...interface{})
    Errorf(format string, args ...interface{})
    Fatalf(format string, args ...interface{})
    // SetLevel 设置输出端日志级别
    SetLevel(output string, level Level)
    // GetLevel 获取输出端日志级别
    GetLevel(output string) Level
    // WithFields 设置一些业务自定义数据到每条 log 里：比如 uid，imei 等 fields 必须 kv 成对出现
    WithFields(fields ...string) Logger
}
```

用户可以直接使用上面接口函数进行日志打印，例如

```go
log.Get("custom").Debug("message")
log.Get("custom").Debugf("hello %s", "terry")
```

## 框架日志
1. 框架以尽量不打日志为原则，将错误一直往上抛交给用户自己处理
2. 底层严重问题才会打印 trace 日志，需要设置环境变量才会开启：export TRPC_LOG_TRACE=1

## `trace` 级别日志的开启

要开启日志的 `trace` 级别，首先要保证配置上的日志级别设置为 `debug` 或 `trace`，然后通过环境变量或代码来开启 `trace`:

1. 通过环境变量设置

在执行服务二进制的脚本中添加

```shell
export TRPC_LOG_TRACE=1
./server -conf path/to/trpc_go.yaml
```

2. 通过代码进行设置

添加以下代码即可：

```go
import "git.code.oa.com/trpc-go/trpc-go/log"

func init() {
    log.EnableTrace()
}
```

推荐使用环境变量的方式，更灵活一点

# 关于 `caller_skip` 的说明

使用 `logger` 的方式不同，`caller_skip` 的设置也有所不同：

## Default Logger:

```go
log.Debug("default logger") // 使用默认的 logger
```

此时该 log 使用的配置为 `default`:

```yaml
default:  # 默认日志配置，log.Debug("xxx")
  - writer: console  # 控制台标准输出 默认
    level: debug  # 标准输出日志的级别
  - writer: file  # 本地文件日志
    level: debug  # 本地文件滚动日志的级别
    formatter: json  # 标准输出日志的格式
    writer_config:  # 本地文件输出具体配置
      filename: ../log/trpc_time.log  # 本地文件滚动日志存放的路径
      write_mode: 2  # 日志写入模式，1-同步，2-异步，3-极速 (异步丢弃), 不配置默认异步模式
      roll_type: time  # 文件滚动类型，time 为按时间滚动
      max_age: 7  # 最大日志保留天数
      max_backups: 10  # 最大日志文件数
      compress: false  # 日志文件是否压缩
      max_size: 10  # 本地文件滚动日志的大小 单位 MB
      time_unit: day  # 滚动时间间隔，支持：minute/hour/day/month/year
```

此时不需要关注或者去设置 `caller_skip` 的值，该值默认为 2，意思是在 `zap.Logger.Debug` 上套了两层（`trpc.log.Debug -> trpc.log.zapLog.Debug -> zap.Logger.Debug`）

## 将自定义的 logger 放到 context 中进行使用：

```go
trpc.Message(ctx).WithLogger(log.Get("custom"))
log.DebugContext(ctx, "custom log msg")
```

此时也不需要关注或者去设置 `caller_skip` 的值，该值默认为 2，意思是在 `zap.Logger.Debug` 上套了两层（`trpc.log.DebugContext -> trpc.log.zapLog.Debug -> zap.Logger.Debug`）

配置例子如下：

```yaml
custom:  # 业务自定义的 logger 配置，名字随便定，每个服务可以有多个 logger，可使用 log.Get("custom").Debug("xxx") 打日志
  - writer: file  # 业务自定义的 core 配置，名字随便定
    level: debug  # 业务自定义 core 输出的级别
    writer_config:  # 本地文件输出具体配置
      filename: ../log/trpc1.log  # 本地文件滚动日志存放的路径
```

## 不在 context 中使用自定义的 logger：

```go
log.Get("custom").Debug("message")
```

此时需要将 `custom` logger 的 `caller_skip` 值设置为 1，因为 `log.Get("custom")` 直接返回的是 `trpc.log.zapLog`，调用 `trpc.log.zapLog.Debug` 只在 `zap.Logger.Debug` 上套了一层（`trpc.log.zapLog.Debug -> zap.Logger.Debug`）

配置例子如下：

```yaml
custom:  # 业务自定义的 logger 配置，名字随便定，每个服务可以有多个 logger，可使用 log.Get("custom").Debug("xxx") 打日志
  - writer: file  # 业务自定义的 core 配置，名字随便定
    caller_skip: 1  # 用于定位日志的调用处
    level: debug  # 业务自定义 core 输出的级别
    writer_config:  # 本地文件输出具体配置
      filename: ../log/trpc1.log  # 本地文件滚动日志存放的路径
```

要注意 `caller_skip` 放置的位置（不要放在 `writer_config` 里面），并且对于多个 `writer` 都有 `caller_skip` 时，该 logger 的 `caller_skip` 的值以最后一个为准，比如：

```yaml
custom:  # 业务自定义的 logger 配置，名字随便定，每个服务可以有多个 logger，可使用 log.Get("custom").Debug("xxx") 打日志
  - writer: file  # 业务自定义的 core 配置，名字随便定
    caller_skip: 1  # 用于定位日志的调用处
    level: debug  # 业务自定义 core 输出的级别
    writer_config:  # 本地文件输出具体配置
      filename: ../log/trpc1.log  # 本地文件滚动日志存放的路径
  - writer: file  # 本地文件日志
    caller_skip: 2  # 用于定位日志的调用处
    level: debug  # 本地文件滚动日志的级别
    writer_config:  # 本地文件输出具体配置
      filename: ../log/trpc2.log  # 本地文件滚动日志存放的路径
```

最终 `custom` 这个 logger 的 `caller_skip` 值会被设置为 2

**注意：** 上述用法 2 和用法 3 是冲突的，只能同时用其中的一种
