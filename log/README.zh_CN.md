[English](README.md) | 中文

# log

## 概述

下面是一个使用 `log` 包的最简单的程序：

```go
 // The code below is located in example/main.go
package main

import "trpc.group/trpc-go/trpc-go/log"

func main() {
    log.Info("hello, world")
}
```

截至撰写本文时，它打印：

```
2023-09-07 11:46:40.905 INFO    example/main.go:6       hello, world
```

`Info` 函数使用 `log` 包中默认的 Logger 打印出一条日志级别为 Info 的消息。
根据输出消息的重要性和紧急性， log 包除了支持上述的 Info 级别的日志外，还有提供其他五种日志级别（Trace、Debug、Warn、Error 和 Fatal）。
该日志除了包含消息-"hello, world"和日志级别-"INFO"外，还包含打印时间-"2023-09-07 11:46:40.905"，调用栈-"example/main.go:6"。

你也可以使用 `Infof` 输出相同的日志级别的日志， `Infof` 更为灵活，允许你以想要的格式打印消息。

```go
log.Infof("hello, %s", "world")
```

除此之外，你可以传递一系列键值对给 `With` 函数从默认的 Logger 中创建出新的 Logger。
新的 Logger 在打印日志时，会把键值对输出到消息的后面。

```go
logger := log.With(log.Field{Key:"user", Value: os.Getenv("USER")})
logger.Info("hello, world")
```

现在的输出如下所示：

```
2023-09-07 15:05:21.168 INFO    example/main.go:12      hello, world    {"user": "goodliu"}
```

正如之前提到的，`Info` 函数使用默认的 Logger，你可以显式地获取这个 Logger，并调用它的方法：

```go
dl := log.GetDefaultLogger()
l := dl.With(log.Field{Key: "user", Value: os.Getenv("USER")})
l.Info("hello, world")
```

## 主要类型

`log` 包包含两种主要类型：

- `Logger` 是前端，提供类似于 `Info`和 `Infof` 的输出方法，你可以使用这些方法产生日志。
- `Writer` 是后端，处理 `Logger` 产生的日志，将日志写入到各种日志服务系统中，如控制台、本地文件和远端。

`log` 包支持设置多个互相独立的 `Logger`，每个 `Logger` 又支持设置多个互相独立的 `Writer`。
如图所示，在这个示例图中包含三个 `Logger`， “Default Logger”，“Other Logger-1“ 和  ”Other Logger-2“，其中的 ”Default Logger“ 是 log 包中内置默认的 `Logger`。
”Default Logger“ 包含三个不同的 `Writer`， "Console Writer"，“File Writer” 和 “Remote Writer”，其中的 "Console Writer" 是 ”Default Logger“ 默认的 `Writer`。
`Logger` 和 `Writer` 都被设计为定制化开发的插件，关于如何开发它们，可以参考[这里](/docs/developer_guide/develop_plugins/log_zh_CN.md) 。


```ascii
                                             +------------------+
                                             | +--------------+ |
                                             | |Console Writer| |
                                             | +--------------+ |
                                             | +-----------+    |
                   +----------------+        | | File Witer|    |
     +-------------> Default Logger +--------> +-----------+    |
     |             +----------------+        | +-------------+  |
     |                                       | |Remote Writer|  |
     |                                       | +-------------+  |
     |                                       +------------------+
     |                                        +-------------+
     |                                        | +--------+  |
     |                                        | |Writer-A|  |
+----+----+        +----------------+         | +--------+  |
| Loggers +--------> Other Logger-1 +-------->| +--------+  |
+----+----+        +----------------+         | |Writer-B|  |
     |                                        | +--------+  |
     |                                        +-------------+
     |             +----------------+          +---------+
     +-------------> Other Logger-2 +----------> Writer-C|
                   +----------------+          +---------+
```

下面将先介绍如何在配置文件中配置 `Logger` 和 `Writer` ，然后再以自下而上地方式分别介绍 `Writer` 和 `Logger`。

## 配置 `Logger` 和 `Writer`

因为 `Logger` 和 `Writer` 都是采用插件的方式来实现的，所以相关配置都需要放到“plugins”字段下面。

```yaml
plugins:
  log:
    default:
      - writer: console
        ...
      - writer: file
        ...
      - writer: remote
        ...
    logger1:
      - writer: writer-a
        ...
      - writer: writer-b
        ...
    logger2:
      - writer: writer-c
        ...
```

上面配置了三个名字为 “default”，“logger1”，和 “logger2” 的 `Logger`。
其中的 “default” 是系统默认的 `Logger`，它配置了名字为 “console”，“file” 和 “remote” 的 `Writer`。
在不做任何日志配置时，日志默认写入到 ”console“，日志级别为 Debug，打印方式为文本格式，其对应的配置文件为：

```yaml
plugins:
  log:
    default:
      - writer: console
        level: debug
        formatter: console
```

对于 `Writer` 的配置参数，设计如下：

| 配置项                  | 配置项           |  类型  | 默认值 | 配置解释                                                                  |
| ----------------------- | ---------------- | :----: | :----: |-----------------------------------------------------------------------|
| writer                  | writer           | string |        | 必填 日志 Writer 插件的名字，框架默认支持“file，console”                               |
| writer                  | writer_config    |  对象  |  nil   | 当日志 Writer 为“file”时才需要设置                                              |
| writer                  | formatter        | string |   “”   | 日志打印格式，支持“console”和“json”，为空时默认设置为“console”                           |
| writer                  | formatter_config |  对象  |  nil   | 日志输出时 zapcore Encoder 配置，为空时为参考 formatter_config 的默认值                 |
| writer                  | remote_config    |  对象  |  nil   | 远程日志格式 配置格式你随便定 由第三方组件自己注册，具体配置参考各日志插件文档                              |
| writer                  | level            | string |        | 必填 日志级别大于等于设置级别时，输出到 writer 后端 取值范围：trace，debug，info，warn，error，fatal |
| writer                  | caller_skip      |  int   |   2    | 用于控制 log 函数嵌套深度，不填或者输入 0 时，默认为 2                                      |
| writer.formatter_config | time_fmt         | string |   ""   | 日志输出时间格式，空默认为"2006-01-02 15:04:05.000"                                |
| writer.formatter_config | time_key         | string |   ""   | 日志输出时间在以 Json 输出时的 key 的名称，默认为"T", 用 "none" 来禁用这一字段                   |
| writer.formatter_config | level_key        | string |   ""   | 日志级别在以 Json 输出时的 key 的名称，默认为"L", 用 "none" 来禁用这一字段                     |
| writer.formatter_config | name_key         | string |   ""   | 日志名称在以 Json 输出时的 key 的名称，默认为"N", 用 "none" 来禁用这一字段                     |
| writer.formatter_config | caller_key       | string |   ""   | 日志输出调用者在以 Json 输出时的 key 的名称，默认为"C", 用 "none" 来禁用这一字段                  |
| writer.formatter_config | message_key      | string |   ""   | 日志输出消息体在以 Json 输出时的 key 的名称，默认为"M", 用 "none" 来禁用这一字段                  |
| writer.formatter_config | stacktrace_key   | string |   ""   | 日志输出堆栈在以 Json 输出时的 key 的名称，默认为"S", 用 "none" 来禁用这一字段                   |
| writer.writer_config    | log_path         | string |        | 必填 日志路径名，例如：/usr/local/trpc/log/                                      |
| writer.writer_config    | filename         | string |        | 必填 日志文件名，例如：trpc.log                                                  |
| writer.writer_config    | write_mode       |  int   |   0    | 日志写入模式，1-同步，2-异步，3-极速 (异步丢弃), 不配置默认极速模式，即异步丢弃                         |
| writer.writer_config    | roll_type        | string |   ""   | 文件滚动类型，"size"按大小分割文件，"time"按时间分割文件，为空时默认按大小分割                         |
| writer.writer_config    | max_age          |  int   |   0    | 日志最大保留时间，为 0 表示不清理旧文件                                                 |
| writer.writer_config    | max_backups      |  int   |   0    | 日志最大文件数，为 0 表示不删除多余文件                                                 |
| writer.writer_config    | compress         |  bool  | false  | 日志文件是否压缩，默认不压缩                                                        |
| writer.writer_config    | max_size         | string |   ""   | 按大小分割时才有效，日志文件最大大小（单位 MB），为 0 表示不按大小滚动                                |
| writer.writer_config    | time_unit        | string |   ""   | 按时间分割时才有效，按时间分割文件的时间单位，支持 year/month/day/hour/minute, 默认值为 day        |

## 多 Writer

多 Writer 可以提供如下功能：

- 支持日志同时上报到多个输出后端，比如同时打印到 console 和本地日志文件
- 支持每个输出后端单独设置日志级别，比如 debug 级别以上日志打 console，warn 级别以上日志保存到日志文件
- 支持每个输出后端单独设置日志格式（console，json 等），日志字段名称
- 支持“file”类型后端滚动日志功能，包括按文件大小或者时间来分割日志文件

这里需要强调的是：**日志的打印设置是以 Writer 粒度来配置的**。
你需要为每个输出端做配置，比如：Debug 级别以上日志打 console，Warn 级别以上日志保存到日志文件，那就必须要分别配置 console 和 file 这两个 Writer。

系统默认支持 **"console"** 和 **"file"** 两种 Writer。

### 将日志写入到控制台

当 writer 设置为 ”console“ 时，日志会被写入到控制台。
配置参考示例如下：

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

### 将日志写入到本地

当 writer 设置为“file”时，日志会被写入到本地日志文件。

日志按时间进行文件滚动存放的配置示例如下：

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

### 将日志写入到远端

将日志写入到远端需要设置“remote_config”字段。
默认 `Logger` 配置两个远端 `Writer` 的配置示例如下：

```yaml
plugins:
  log: #日志配置 支持多个日志 可通过 log.Get("xxx").Debug 打日志
    default: # 默认日志的配置，每个日志可支持多个 Writer
      - writer: remote-writer1  # 名字为 remote-writer1 的 remote writer
        level: debug # 远程日志的级别
        remote_config:  # 远程日志配置，你自定义的结构，每一种远程日志都有自己独立的配置
          # 相关配置
      - writer: remote-writer2 # 名字为 remote-writer2 的 remote writer
        level: info # 远程日志的级别
        remote_config:  # 远程日志配置，你自定义的结构，每一种远程日志都有自己独立的配置
          # 相关配置
```

## 多 Logger

`log` 包支持同时多个 logger，每个 logger 可以设置不同的日志级别，打印格式，和 writers。
你可以为不同的应用场景设置不同的 logger，例如：

- 不同的应用模块使用不同的日志文件存储
- 不同的事件，基于事件的关注度不同，采用不同的日志级别收集日志

多 logger 功能大大增加了日志使用的灵活性。

### 配置 Logger

在配置文件中配置你的 logger，比如配置名为 “custom” 的 logger：

```yaml
plugins:
  log:  # 所有日志配置
    default:  # 默认日志配置，log.Debug("xxx")
      - writer: console  # 控制台标准输出 默认
        level: debug  # 标准输出日志的级别
    custom:  # 你自定义的 Logger 配置，名字随便定，每个服务可以有多个 Logger，可使用 log.Get("custom").Debug("xxx") 打日志
      - writer: file  # 你自定义的core配置，名字随便定
        caller_skip: 1  # 用于定位日志的调用处
        level: debug  # 你自定义core输出的级别
        writer_config:  # 本地文件输出具体配置
          filename: ../log/trpc1.log  # 本地文件滚动日志存放的路径
```

配置文件中对于 `caller_skip` 的疑问见「关于 `caller_skip` 的说明」这一节。

### 注册 Logger 插件

在 `main` 函数入口注册日志插件:

```go
import (
    "trpc.group/trpc-go/trpc-go/log"
    "trpc.group/trpc-go/trpc-go/plugin"
)
func main() {
    // 注意：plugin.Register 要在 trpc.NewServer 之前执行
    plugin.Register("custom", log.DefaultLogFactory)
    s := trpc.NewServer()
}
```

### 获取 Logger

`log` 包提供了以下两种方式供你获取 `Logger`：

#### 方法一：直接指定 Logger

```go
// 使用名称为“custom”的 Logger
// 注意：log.Get 得在 trpc.NewServer 之后执行，因为插件的加载是在 trpc.NewServer 中进行的
log.Get("custom").Debug("message")
```

#### 方法二：为 Context 指定 Logger，使用 Context 类型日志接口

```go
// 设置 ctx 的 Logger 为 custom
trpc.Message(ctx).WithLogger(log.Get("custom"))
// 使用“Context”类型的日志接口
log.DebugContext(ctx, "custom log msg")
```

### 日志级别

根据输出消息的重要性和紧急性， log 包提供六种日志打印级别，并按由低到高的顺序进行了如下划分：

1. Trace：这是最低的级别，通常用于记录程序的所有运行信息，包括一些细节信息和调试信息。
   这个级别的日志通常只在开发和调试阶段使用，因为它可能会生成大量的日志数据。
2. Debug：这个级别主要用于调试过程中，提供程序运行的详细信息，帮助你找出问题的原因。
3. Info： 这个级别用于记录程序的常规操作，比如用户登录、系统状态更新等。
   这些信息对于了解系统的运行状态和性能很有帮助。
4. Warn：警告级别表示可能的问题，这些问题不会立即影响程序的功能，但可能会在未来引发错误。
   这个级别的日志可以帮助你提前发现和预防问题。
5. Error：错误级别表示严重的问题，这些问题可能会阻止程序执行某些功能。
   这个级别的日志需要立即关注和处理。
6. Fatal：致命错误级别表示非常严重的错误，这些错误可能会导致程序崩溃。
   这是最高的日志级别，表示需要立即处理的严重问题。

正确地使用日志级别可以帮助你更好地理解和调试你的应用程序。

### 日志打印接口

`log` 包提供了3组日志打印接口：

- 默认 Logger 的日志函数：使用最频繁的一种方式。直接使用默认 `Logger` 进行日志打印，方便简单。
- 基于 Context Logger 的日志函数：为特定场景提供指定 `Logger`，并保存在 Context 里，后续使用当前 Context `Logger` 进行日志打印。
  这种方式特别适合 RPC 调用模式：当服务收到 RPC 请求时，为 ctx 设置 `Logger`，并附加此次请求相关的 field 信息，此 RPC 调用的后续日志上报都会带上之前设定的 field 信息。
- 指定的 Logger 的日志函数：可以用于用户自行选择 `Logger`，并调用 `Logger` 的接口函数实现日志打印。

#### 默认 Logger 的日志函数

默认 Logger 的名称固定为“default”，默认打印级别为 debug，打印 console，打印格式为文本格式。配置可以通过在框架文件中 plugins.log 中修改。
使用默认 Logger 打印日志的函数如下，函数风格与“fmt.Print()”和"fmt.Printf()"保持一致。

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

同时系统也提供了默认 Logger 的管理函数，包括获取，设置默认 Logger，设置 Logger 的打印级别。

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

#### 基于 Context Logger 的日志函数

基于 Context Logger 的日志函数，每个 Context Logger 必须独占一个 Logger，确保 Context Logger 的配置不被串改。
`log` 提供了 `With` 和 `WithContext` 来继承父 Logger 配置，并生成新的 Logger。

```go
// With adds user defined fields to Logger. Field support multiple values.
func With(fields ...Field) Logger
// WithContext adds user defined fields to the Logger of context.
// Fields support multiple values.
func WithContext(ctx context.Context, fields ...Field) Logger
```

然后通过下面函数为 context 设置 Logger：

```go
logger := ...
trpc.Message(ctx).WithLogger(logger)
```

context 级别的日志打印函数和默认 Logger 打印日志的函数类似：

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

#### 指定的 Logger 的日志函数

同时，`log` 也提供了函数供用户自行选择 Logger。
对于每一个 Logger 都实现了 Logger Interface。
接口定义为：

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

    // WithFields 设置一些你自定义数据到每条 log 里：比如 uid，imei 等 fields 必须 kv 成对出现
    WithFields(fields ...string) Logger
}
```

用户可以直接使用上面接口函数进行日志打印，例如

```go
log.Get("custom").Debug("message")
log.Get("custom").Debugf("hello %s", "world")
```

## 框架日志

1. 框架以尽量不打日志为原则，将错误一直往上抛交给用户自己处理
2. 底层严重问题才会打印 trace 日志，需要设置环境变量才会开启：export TRPC_LOG_TRACE=1

### `trace` 级别日志的开启

要开启日志的 `trace` 级别，首先要保证配置上的日志级别设置为 `debug` 或 `trace`，然后通过环境变量来开启 `trace`:

- 通过环境变量设置

在执行服务二进制的脚本中添加

```shell
export TRPC_LOG_TRACE=1
./server -conf path/to/trpc_go.yaml
```

- 通过代码进行设置

添加以下代码即可：

```go
import "trpc.group/trpc-go/trpc-go/log"

func init() {
    log.EnableTrace()
}
```

推荐使用环境变量的方式，更灵活一点。

## 关于 `caller_skip` 的说明

使用 `Logger` 的方式不同，`caller_skip` 的设置也有所不同：

### 用法1: 使用 Default Logger

```go
log.Debug("default logger") // 使用默认的 logger
```

此时该 Logger 使用的配置为 `default`:

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

### 用法2: 将自定义的 Logger 放到 context

```go
trpc.Message(ctx).WithLogger(log.Get("custom"))
log.DebugContext(ctx, "custom log msg")
```

此时也不需要关注或者去设置 `caller_skip` 的值，该值默认为 2，意思是在 `zap.Logger.Debug` 上套了两层（`trpc.log.DebugContext -> trpc.log.zapLog.Debug -> zap.Logger.Debug`）

配置例子如下：

```yaml
custom:  # 你的 Logger 配置，名字随便定，每个服务可以有多个 Logger，可使用 log.Get("custom").Debug("xxx") 打日志
  - writer: file  # 你的 core 配置，名字随便定
    level: debug  # 你的 core 输出的级别
    writer_config:  # 本地文件输出具体配置
      filename: ../log/trpc1.log  # 本地文件滚动日志存放的路径
```

### 用法3: 不在 context 中使用自定义的 Logger

```go
log.Get("custom").Debug("message")
```

此时需要将 `custom` Logger 的 `caller_skip` 值设置为 1，因为 `log.Get("custom")` 直接返回的是 `trpc.log.zapLog`，调用 `trpc.log.zapLog.Debug` 只在 `zap.Logger.Debug` 上套了一层（`trpc.log.zapLog.Debug -> zap.Logger.Debug`）

配置例子如下：

```yaml
custom:  # 你自定义的 Logger 配置，名字随便定，每个服务可以有多个 Logger，可使用 log.Get("custom").Debug("xxx") 打日志
  - writer: file  # 你自定义的 core 配置，名字随便定
    caller_skip: 1  # 用于定位日志的调用处
    level: debug  # 你自定义 core 输出的级别
    writer_config:  # 本地文件输出具体配置
      filename: ../log/trpc1.log  # 本地文件滚动日志存放的路径
```

要注意 `caller_skip` 放置的位置（不要放在 `writer_config` 里面），并且对于多个 `writer` 都有 `caller_skip` 时，该 Logger 的 `caller_skip` 的值以最后一个为准，比如：

```yaml
custom:  # 你自定义的 Logger 配置，名字随便定，每个服务可以有多个 Logger，可使用 log.Get("custom").Debug("xxx") 打日志
  - writer: file  # 你自定义的 core 配置，名字随便定
    caller_skip: 1  # 用于定位日志的调用处
    level: debug  # 你自定义的 core 输出的级别
    writer_config:  # 本地文件输出具体配置
      filename: ../log/trpc1.log  # 本地文件滚动日志存放的路径
  - writer: file  # 本地文件日志
    caller_skip: 2  # 用于定位日志的调用处
    level: debug  # 本地文件滚动日志的级别
    writer_config:  # 本地文件输出具体配置
      filename: ../log/trpc2.log  # 本地文件滚动日志存放的路径
```

最终 `custom` 这个 Logger 的 `caller_skip` 值会被设置为 2。

**注意：** 上述用法 2 和用法 3 是冲突的，只能同时用其中的一种。
