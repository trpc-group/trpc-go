# tRPC-Go 日志功能及实现 

## 日志配置
```yaml
plugins:
  log:                                      #所有日志配置
    default:                                  #默认日志配置，log.Debug("xxx")
      - writer: console                         #控制台标准输出 默认
        level: debug                            #标准输出日志的级别
      - writer: file                              #本地文件日志
        level: debug                              #本地文件滚动日志的级别
        formatter: json                           #标准输出日志的格式
        formatter_config:
          time_fmt: 2006-01-02 15:04:05           #日志时间格式。"2006-01-02 15:04:05"为常规时间格式，"seconds"为秒级时间戳，"milliseconds"为毫秒时间戳，"nanoseconds"为纳秒时间戳
          time_key: Time                          #日志时间字段名称，不填默认"T"
          level_key: Level                        #日志级别字段名称，不填默认"L"
          name_key: Name                          #日志名称字段名称，不填默认"N"
          caller_key: Caller                      #日志调用方字段名称，不填默认"C"
          function_key: Function                  #日志调用方字段名称，不填默认不打印函数名
          message_key: Message                    #日志消息体字段名称，不填默认"M"
          stacktrace_key: StackTrace              #日志堆栈字段名称，不填默认"S"
        writer_config:                            #本地文件输出具体配置
          filename: ../log/trpc_size.log          #本地文件滚动日志存放的路径
          write_mode: 3                           #日志写入模式，1-同步，2-异步，3-极速 (异步丢弃), 不配置默认极速模式
          roll_type: size                         #文件滚动类型，size 为按大小滚动
          max_age: 7                              #最大日志保留天数
          max_backups: 10                         #最大日志文件数
          compress:  false                        #日志文件是否压缩
          max_size: 10                            #本地文件滚动日志的大小 单位 MB
      - writer: file                              #本地文件日志
        level: debug                              #本地文件滚动日志的级别
        formatter: json                           #标准输出日志的格式
        writer_config:                            #本地文件输出具体配置
          filename: ../log/trpc_time.log          #本地文件滚动日志存放的路径
          write_mode: 3                           #日志写入模式，1-同步，2-异步，3-极速 (异步丢弃), 不配置默认极速模式
          roll_type: time                         #文件滚动类型，time 为按时间滚动
          max_age: 7                              #最大日志保留天数
          max_backups: 10                         #最大日志文件数
          compress:  false                        #日志文件是否压缩
          max_size: 10                            #本地文件滚动日志的大小 单位 MB
          time_unit: day                          #滚动时间间隔，支持：minute/hour/day/month/year
      - writer: atta                                #atta 远程日志输出
        remote_config:                              #远程日志配置，业务自定义结构，每一种远程日志都有自己独立的配置
          atta_id: '05e00006180'                    #atta id 每个业务自己申请
          atta_token: '6851146865'                  #atta token 业务自己申请
          message_key: msg                          #日志打印包体的对应 atta 的 field
          field:                                    #申请 atta id 时，业务自己定义的表结构字段，顺序必须一致
            - msg
            - uid
            - cmd
    custom:                                   #业务自定义的 logger 配置，名字随便定，每个服务可以有多个 logger，可使用 log.Get("custom").Debug("xxx") 打日志
      - writer: file                              #业务自定义的 core 配置，名字随便定
        caller_skip: 1                            #用于定位日志的调用处
        level: debug                              #业务自定义 core 输出的级别
        writer_config:                            #本地文件输出具体配置
          filename: ../log/trpc1.log              #本地文件滚动日志存放的路径
      - writer: file                              #本地文件日志
        caller_skip: 1                            #用于定位日志的调用处
        level: debug                              #本地文件滚动日志的级别
        writer_config:                            #本地文件输出具体配置
          filename: ../log/trpc2.log              #本地文件滚动日志存放的路径
```

## 相关概念解析
- logger: 具体打日志的对外接口，每个日志都可以有多个输出，如上配置，log.Debug 可以同时输出到 console 终端和 file 本地文件，可以任意多个
- log factory: 日志插件工厂，每个日志都是一个插件，一个服务可以有多个日志插件，需要通过日志工厂读取配置信息实例化具体 logger 并注册到框架中，没有使用日志工厂，默认只输出到终端
- writer factory: 日志输出插件工厂，每个输出流都是一个插件，一个日志可以有多个输出，需要通过输出工厂读取具体配置实例化具体 core
- core: zap 的具体日志输出实例，有终端，本地日志，远程日志等等
- zap: uber 的一个 log 开源实现，trpc 框架直接使用的 zap 的实现
- with fields: 设置一些业务自定义数据到每条 log 里：比如 uid，imei 等，每个请求入口设置一次


## 日志插件实例化流程
- 1. 首先框架会提前注册好 default log factory, console writer factory, file writer factory
- 2. log factory 解析 logger 配置，遍历 writer 配置
- 3. 逐个 writer 调用 writer factory 加载 writer 配置
- 4. writer factory 实例化 core
- 5. 多个 core 组合成一个 logger
- 6. 注册 logger 到框架中

## 支持多 logger
1. 首先需要在 main 函数入口注册插件
```go
	import (
		"trpc.group/trpc-go/trpc-go/log"
		"trpc.group/trpc-go/trpc-go/plugin"	
	)
	func main() {
		plugin.Register("custom", log.DefaultLogFactory) 
		
		s := trpc.NewServer()
	}
```
2. 配置文件定义自己的 logger，如上 custom 
3. 业务代码具体场景 Get 使用不同的 logger
```go
	log.Get("custom").Debug("message")
```
4. 由于一个 context 只能保存一个 logger，所以 DebugContext 等接口只能打印 default logger，需要使用 XxxContext 接口打印自定义 logger 时，可以在请求入口 get logger 后设置到 ctx 里面，如
```go
    trpc.Message(ctx).WithLogger(log.Get("custom"))
    log.DebugContext(ctx, "custom log msg")
```
## 框架日志
1. 框架以尽量不打日志为原则，将错误一直往上抛交给用户自己处理
2. 底层严重问题才会打印 trace 日志，需要设置环境变量才会开启：export TRPC_LOG_TRACE=1

## 关于 `caller_skip` 的说明

首先捋一下使用 `logger` 的几种方式：

1. Default Logger:

```go
log.Debug("default logger") // 使用默认的 logger
```

此时该 log 使用的配置为 `default`:

```yaml
    default:                              #默认日志配置，log.Debug("xxx")
      - writer: console                   #控制台标准输出 默认
        level: debug                      #标准输出日志的级别
      - writer: file                      #本地文件日志
        level: debug                      #本地文件滚动日志的级别
        formatter: json                   #标准输出日志的格式
        writer_config:                    #本地文件输出具体配置
          filename: ../log/trpc_time.log  #本地文件滚动日志存放的路径
          write_mode: 3                   #日志写入模式，1-同步，2-异步，3-极速 (异步丢弃), 不配置默认极速模式
          roll_type: time                 #文件滚动类型，time 为按时间滚动
          max_age: 7                      #最大日志保留天数
          max_backups: 10                 #最大日志文件数
          compress:  false                #日志文件是否压缩
          max_size: 10                    #本地文件滚动日志的大小 单位 MB
          time_unit: day                  #滚动时间间隔，支持：minute/hour/day/month/year
```

此时不需要关注或者去设置 `caller_skip` 的值，该值默认为 2，意思是在 `zap.Logger.Debug` 上套了两层（`trpc.log.Debug -> trpc.log.zapLog.Debug -> zap.Logger.Debug`）

2. 将自定义的 logger 放到 context 中进行使用：

```go
    trpc.Message(ctx).WithLogger(log.Get("custom"))
    log.DebugContext(ctx, "custom log msg")
```

此时也不需要关注或者去设置 `caller_skip` 的值，该值默认为 2，意思是在 `zap.Logger.Debug` 上套了两层（`trpc.log.DebugContext -> trpc.log.zapLog.Debug -> zap.Logger.Debug`）

配置例子如下：

```yaml
    custom:                           #业务自定义的 logger 配置，名字随便定，每个服务可以有多个 logger，可使用 log.Get("custom").Debug("xxx") 打日志
      - writer: file                  #业务自定义的 core 配置，名字随便定
        level: debug                  #业务自定义 core 输出的级别
        writer_config:                #本地文件输出具体配置
          filename: ../log/trpc1.log  #本地文件滚动日志存放的路径
```


3. 不在 context 中使用自定义的 logger：

```go
	log.Get("custom").Debug("message")
```

此时需要将 `custom` logger 的 `caller_skip` 值设置为 1，因为 `log.Get("custom")` 直接返回的是 `trpc.log.zapLog`，调用 `trpc.log.zapLog.Debug` 只在 `zap.Logger.Debug` 上套了一层（`trpc.log.zapLog.Debug -> zap.Logger.Debug`）

配置例子如下：

```yaml
    custom:                           #业务自定义的 logger 配置，名字随便定，每个服务可以有多个 logger，可使用 log.Get("custom").Debug("xxx") 打日志
      - writer: file                  #业务自定义的 core 配置，名字随便定
        caller_skip: 1                #用于定位日志的调用处
        level: debug                  #业务自定义 core 输出的级别
        writer_config:                #本地文件输出具体配置
          filename: ../log/trpc1.log  #本地文件滚动日志存放的路径
```

要注意 `caller_skip` 放置的位置（不要放在 `writer_config` 里面），并且对于多个 `writer` 都有 `caller_skip` 时，该 logger 的 `caller_skip` 的值以最后一个为准，比如：

```yaml
    custom:                           #业务自定义的 logger 配置，名字随便定，每个服务可以有多个 logger，可使用 log.Get("custom").Debug("xxx") 打日志
      - writer: file                  #业务自定义的 core 配置，名字随便定
        caller_skip: 1                #用于定位日志的调用处
        level: debug                  #业务自定义 core 输出的级别
        writer_config:                #本地文件输出具体配置
          filename: ../log/trpc1.log  #本地文件滚动日志存放的路径
      - writer: file                  #本地文件日志
        caller_skip: 2                #用于定位日志的调用处
        level: debug                  #本地文件滚动日志的级别
        writer_config:                #本地文件输出具体配置
          filename: ../log/trpc2.log  #本地文件滚动日志存放的路径
``` 

最终 `custom` 这个 logger 的 `caller_skip` 值会被设置为 2

__注意：__ 上述用法 2 和用法 3 是冲突的，只能同时用其中的一种

