[TOC]



# Introduction

Logs are an important part of software development, and a good log is an important source of information for us to analyze the running status of the software.

The Trpc-go framework provides a set of log printing, collection and reporting solutions for the business. Through the introduction of this article, it is intended to provide users with the following information:

- What logging capabilities does the framework provide?
- How does the framework achieve these functions?
- How to use the standard logging interface?
- How to configure the log configuration?
- How to connect with common log systems? Use asynchronous mod as default on missing config

# Features

## Functional Overview

As shown in the figure, the Trpc-go log module includes two implementations: log printing programming interface and docking with log output. Two concepts of Logger and Writer are introduced in the implementation, and three sets of log printing functions are provided.

![ 'logger.png'](/.resources/user_guide/log/logger.png)

Logger and Writer are respectively responsible for log printing and docking with the log service system. They both adopt plug-in programming and can be customized for development. For how to develop log plugins, please refer to [here](https://git.woa.com/trpc-go/trpc-wiki/blob/main/developer_guide/develop_plugins/log.md). Logger and Writer are defined as follows:

- Logger is used to realize the general interface related to log printing. Logger adopts plug-in programming and supports multiple Logger functions. Users can choose loggers to realize differentiated log printing and collection. Interface with the log backend through the writer plug-in.
- Writer, also known as "log output terminal", is responsible for the log collection and reporting function, including log reporting format and docking with the back-end log system, etc. Each logger can have multiple Writers, for example: log.Debug can output to the console and local log files at the same time. Writer adopts plug-in programming and can be flexibly expanded.

The framework provides 3 sets of log printing functions:

- Use the default Logger to print logs: the most frequently used method. Directly use the default Logger for log printing, which is convenient and simple
- Print logs based on Context Logger: Provide a specified logger for a specific scenario and save it in the context, and then use the current context logger for log printing. This method is especially suitable for the RPC call mode: when the service receives the RPC request, set the logger for ctx and attach the field information related to this request, and the subsequent log report of this RPC call will bring the previously set field information
- Specify Logger log printing: it can be used for users to select logger by themselves, and call the interface function of logger to realize log printing

## Unified log printing interface

First, the framework divides the log printing levels from low to high with reference to industry standards as follows:

- **trace**: The main function of this level of log is to accurately record the running status of each step of the system-specific event
- **debug**: Point out that fine-grained information events are very helpful for debugging applications, mainly used to print some running information during development
- **info**: Messages highlight the progress of the application at a coarse-grained level. Print some information you are interested in or important. This can be used to output some important information about the running of the program in the production environment, but it cannot be abused to avoid printing too many logs
- **warn**: Indicates that there will be a potential error situation. Some information is not an error message, but some hints should also be given to the programmer
- **error**: Point out that although an error event occurs, it still does not affect the continuous operation of the system. Print error and exception information, if you don't want to output too many logs, you can use this level
- **fatal**: Indicates that each fatal error event will cause the application to exit. This level is relatively high. Major error, you can stop the program directly at this level

The framework provides two styles of print functions similar to `fmt.Print()` and `fmt.Printf()` for each level of log function. The function names are:

```go
// fmt.Print function style
Debug(args ...interface{})
// fmt.Printf function style
Debugf(format string, args ...interface{})
```

The framework provides the ability to add custom fields to each log. Additional fields appear in pairs in kv. for example:

```go
logger := log.WithFields("id","x100-1","city","sz")
logger.Debug("hello, world")
// If the output terminal is set to be stored in json format, the printed result is:
// {"L":"INFO","T":"2020-12-10 11:19:52.104","C":"hello/main.go:29","M":"hello, world","id":"x100-1","city":"sz"}
```

## Multiple Logger Support

TRPC-Go supports multiple loggers at the same time. Each logger can set different log levels, print formats, and report output backends. Businesses can set different loggers for different business scenarios. for example

- Different business modules use different log file storage
- Different events, based on different attention levels of events, use different log levels to collect logs

The multi-logger function greatly increases the flexibility of log usage.

### Register the logger plugin

First, you need to register the plug-in at the main function entry

```go
import (
    "git.code.oa.com/trpc-go/trpc-go/log"
    "git.code.oa.com/trpc-go/trpc-go/plugin"
)
func main() {
    // Note: plugin.Register should be executed before trpc.NewServer.
    plugin.Register("custom", log.DefaultLogFactory)
    s := trpc.NewServer()
}
```

### Configure logger

The configuration file defines its own logger, such as configuring custom:

```go
plugins:
  log:  # All log configuration
    default:  # Default log configuration ,log.Debug("xxx")
      - writer: console  # console stdout default
        level: debug  # The level of standard output logging
    custom:  # Business-defined logger configuration, the name can be set at will, each service can have multiple loggers, you can use log.Get("custom").Debug("xxx") to log
      - writer: file                       # Business-defined core configuration, the name can be set at will
        caller_skip: 1  # The call site used to locate the log
        level: debug  # Business custom core output level
        writer_config:  # Local file output specific configuration
          filename: ../log/trpc1.log  # The path where the local file rolling log is stored
```

For questions about `caller_skip` in the configuration file, see Chapter Explanation about `caller_skip`.

### Get logger

The framework provides the following two ways for businesses to select loggers:

#### Method 1: Specify logger directly

```go
// use a logger named "custom"
// Note: log.Get should be executed after trpc.NewServer, as plugin loading happens within trpc.NewServer.
log. Get("custom"). Debug("message")
```

#### Method 2: Specify a logger for the context and use the context type log interface

```go
// Set the logger of the ctx context to custom
trpc.Message(ctx).WithLogger(log.Get("custom"))
// Use the logging interface of type "Context"
log.DebugContext(ctx, "custom log msg")
```

### Multi-Writer support

For the same logger, the framework also provides the ability to output logs to multiple log output backends (referred to as "writers") at the same time. The system supports **"console"** and **"file"** two backends by default. The framework supports remote log systems through plug-ins, such as Eagle Eye, Zhiyan Log, etc.

The functions of the multi-log output backend specifically include:

- Logs are reported to multiple output backends at the same time, such as printing to the console and local log files at the same time
- Set the log level separately for each output backend: For example, the log above the debug level is displayed in the console, and the log above the warn level is saved to the log file
- Set log format (console, json, etc.), log field name separately for each output backend
- Provides rolling log functionality for "file" type backends, including splitting log files by file size or time

What needs to be emphasized here is: **The log printing settings are configured at the Writer granularity**. The user needs to configure each output terminal, for example: logs above the debug level are typed into the console, and logs above the warn level are saved to log files, then the two Writers console and file must be configured separately.

## log configuration

### Configuration structure

First, let's take a look at the overall structure of the logging configuration. Since the log module implements logger and writer in the form of plug-ins, all log configurations must be placed in the "plugins" area.

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

The above configuration reflects the multi-logger and multi-writer functions. "default", "logger1", "logger2" indicate the name of the logger. Each logger is configured with an array of writers. "default" is the default logger of the system. When no log configuration is made, the log will default to console, the log level is debug, and the printing method is in text format. The corresponding configuration file is:

```yaml
plugins:
  log:
    default:
      - writer: console
        level: debug
        formatter: console
```

For the configuration parameters of Writer, the design is as follows:

| configuration item      | configuration item |  type  | default value | obsolete | configuration explanation                                                                                                                                               |
| ----------------------- | ------------------ | :----: | :-----------: | :------: | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| writer                  | writer             | string |               |    No    | Mandatory Log Writer plug-in name, the framework supports "file, console" by default                                                                                    |
| writer                  | writer_config      | object |      nil      |    no    | only need to be set when the log Writer is "file"                                                                                                                       |
| writer                  | formatter          | string |      ""       |    No    | Log printing format, supports "console" and "json", and defaults to "console" when it is empty                                                                          |
| writer                  | formatter_config   | object |      nil      |    no    | zapcore Encoder configuration when log output, when it is empty, refer to the default value of formatter_config                                                         |
| writer                  | remote_config      | Object |      nil      |    No    | Remote log format The configuration format can be set at will by the third-party component.                                                                             |
| writer                  | level              | string |               |    No    | Mandatory When the log level is greater than or equal to the set level, output to the writer backend Value range: trace, debug, info, warn, error, fatal                |
| writer                  | caller_skip        |  int   |       2       |    No    | Used to control the nesting depth of the log function, if not filled or 0 is entered, the default is 2                                                                  |
| writer.formatter_config | time_fmt           | string |      ""       |    No    | Log output time format, empty default is "2006-01-02 15:04:05.000"                                                                                                      |
| writer.formatter_config | time_key           | string |      ""       |    No    | The name of the key when the log output time is output in Json, the default is "T", use "none" to disable this field                                                                                      |
| writer.formatter_config | level_key          | string |      ""       |    No    | The name of the key when the log level is output in Json, the default is "L", use "none" to disable this field                                                                                            |
| writer.formatter_config | name_key           | string |      ""       |    No    | The name of the key when the log name is output in Json, the default is "N", use "none" to disable this field                                                                                             |
| writer.formatter_config | caller_key         | string |      ""       |    No    | The name of the log output caller's key when outputting in Json, default is "C", use "none" to disable this field                                                                                            |
| writer.formatter_config | message_key        | string |      ""       |    No    | The name of the key when the log output message body is output in Json, the default is "M", use "none" to disable this field                                                                              |
| writer.formatter_config | stacktrace_key     | string |      ""       |    No    | The name of the key when the log output stack is output in Json, default is "S", use "none" to disable this field                                                                                            |
| writer.writer_config    | log_path           | string |               |    No    | Mandatory Log path name, for example: /usr/local/trpc/log/                                                                                                              |
| writer.writer_config    | filename           | string |               |    No    | Mandatory Log file name, for example: trpc.log                                                                                                                          |
| writer.writer_config    | write_mode         |  int   |       0       |    No    | Log writing mode, 1-synchronous, 2-asynchronous, 3-extreme speed (asynchronous discard), do not configure the default extreme speed mode, that is, asynchronous discard |
| writer.writer_config    | roll_type          | string |      ""       |    No    | File roll type, "size" splits files by size, "time" splits files by time, and defaults to split by size when it is empty                                                |
| writer.writer_config    | max_age            |  int   |       0       |    No    | The maximum log retention time, 0 means do not clean up old files                                                                                                       |
| writer.writer_config    | max_backups        |  int   |       0       |    No    | The maximum number of files in the log, 0 means not to delete redundant files                                                                                           |
| writer.writer_config    | compress           |  bool  |     false     |    no    | Whether to compress the log file, default is not compressed                                                                                                             |
| writer.writer_config    | max_size           | string |      ""       |    No    | Only valid when splitting by size, the maximum size of the log file (in MB), 0 means not rolling by size                                                                |
| writer.writer_config    | time_unit          | string |      ""       |    No    | Only valid when split by time, the time unit of split files by time, support year/month/day/hour/minute, the default value is day                                       |

## Using the console

When the writer is set to "console", it means that the log will hit the terminal. The configuration reference example is as follows:

```yaml
plugins:
  log:  # All log configuration
    default:  # default log configuration, log.Debug("xxx")
      - writer: console  # Console standard output default
        level: debug  # Standard output log level
        formatter: json  # Standard output log format
        formatter_config:
          time_fmt: 2006-01-02 15:04:05  # Log time format. "2006-01-02 15:04:05" is the regular time format, "seconds" is the second-level timestamp, "milliseconds" is the millisecond timestamp, "nanoseconds" is the nanosecond timestamp
          time_key: Time  # Log time field name, default "T" if not filled, use "none" to disable this field
          level_key: Level  # Log level field name, default "L" if not filled, use "none" to disable this field
          name_key: Name  # log name field name, default "N" if not filled, use "none" to disable this field
          caller_key: Caller  # log caller field name, default "C" if not filled, use "none" to disable this field
          message_key: Message  # Log message body field name, default "M" if not filled, use "none" to disable this field
          stacktrace_key: StackTrace  # Log stack field name, default "S" if not filled, use "none" to disable this field
```

## Using local logs

When the writer is set to "file", it means that the log is written to a local log file. The configuration example of log file rolling storage according to time is as follows:

```yaml
plugins:
  log:  # All log configuration
    default:  # default log configuration, log.Debug("xxx")
      - writer: file  # Local file log
        level: info  # Local file rolling log level
        formatter: json  # Standard output log format
        formatter_config:
          time_fmt: 2006-01-02 15:04:05  # Log time format. "2006-01-02 15:04:05" is the regular time format, "seconds" is the second-level timestamp, "milliseconds" is the millisecond timestamp, "nanoseconds" is the nanosecond timestamp
          time_key: Time  # Log time field name, default "T" if not filled, use "none" to disable this field
          level_key: Level  # Log level field name, default "L" if not filled, use "none" to disable this field
          name_key: Name  # log name field name, default "N" if not filled, use "none" to disable this field
          caller_key: Caller  # log caller field name, default "C" if not filled, use "none" to disable this field
          message_key: Message  # Log message body field name, default "M" if not filled, use "none" to disable this field
          stacktrace_key: StackTrace  # Log stack field name, default "S" if not filled, use "none" to disable this field
        writer_config:
          log_path: /tmp/log/
          filename: trpc_size.log  # Local file rolling log storage path
          write_mode: 2  # log write mode, 1-synchronous, 2-asynchronous, 3-extreme speed (asynchronous discard), do not configure the default asynchronous mode
          roll_type: time  # File roll type, time is roll by time
          max_age: 7  # Maximum log retention days
          max_backups: 10  # Maximum number of log files
          time_unit: day  # Rolling time interval, support: minute/hour/day/month/year
```

An example configuration of rolling logs based on file size is as follows:

```yaml
plugins:
  log:  # All log configuration
    default:  # default log configuration, log.Debug("xxx")
      - writer: file  # Local file log
        level: info  # Local file rolling log level
        formatter: json  # Standard output log format
        formatter_config:
          time_fmt: 2006-01-02 15:04:05  # Log time format. "2006-01-02 15:04:05" is the regular time format, "seconds" is the second-level timestamp, "milliseconds" is the millisecond timestamp, "nanoseconds" is the nanosecond timestamp
          time_key: Time  # Log time field name, default "T" if not filled, use "none" to disable this field
          level_key: Level  # Log level field name, default "L" if not filled, use "none" to disable this field
          name_key: Name  # log name field name, default "N" if not filled, use "none" to disable this field
          caller_key: Caller  # log caller field name, default "C" if not filled, use "none" to disable this field
          message_key: Message  # Log message body field name, default "M" if not filled, use "none" to disable this field
          stacktrace_key: StackTrace  # Log stack field name, default "S" if not filled, use "none" to disable this field
        writer_config:
          log_path: /tmp/log/
          filename: trpc_size.log  # Local file rolling log storage path
          write_mode: 2  # log write mode, 1-synchronous, 2-asynchronous, 3-extreme speed (asynchronous discard), do not configure the default asynchronous mode
          roll_type: size  # File roll type, size is roll by size
          max_age: 7  # Maximum log retention days
          max_backups: 10  # Maximum number of log files
          compress: false  # Whether the log file is compressed
          max_size: 10  # The size of the local file rolling log, in MB
```

## Using remote logs

For remote logs, "remote_config" needs to be configured, and its configuration format is designed by each writer plug-in. Please first select the log plugin to be connected in [Plugin Ecology](todo), and refer to the plugin link for configuration. The configuration examples of several common remote log configurations are given below.

[Eagle Eye](https://git.woa.com/trpc-go/trpc-log-atta) plugin log configuration example is as follows:

```yaml
plugins:
   log:  # log configuration supports multiple logs, you can log through log.Get("xxx").Debug
     default:  # Default log configuration, each log can support multiple outputs
       - writer: atta  # atta remote log output
         level: debug  # Remote log level
         remote_config:  # Remote log configuration, business custom structure, each remote log has its own independent configuration
           atta_id: '05e00006180'  # atta id Each business applies for itself. Before using it, you need to ensure that the machine has atta agent installed
           atta_token: '6851146865'  # atta token business application
           auto_escape: false  # [Optional, default false] Whether atta report content is escaped, performance reason is disabled by default, true is enabled
           attaobj_size: 3  # [Optional, default 3] The log is reported asynchronously, the number of goroutinues consuming the log, one goroutinue and one att obj, to avoid race conditions
           channel_block: false  # [Optional, default false] The log is reported asynchronously, whether the writing pipeline is blocked.
           channel_capacity: 10000  # [optional, default 10000] log pipeline capacity
           enable_batch: false  # [optional, default false] Whether to cache batch sending logs,
           send_internal: 1000  # [Optional, default 1000] Valid when enable_batch is true, cache batch sending time interval, unit ms
           message_key: msg  # The field corresponding to atta of the log printing package body
           level_key: level  # [optional, empty by default] the atta field corresponding to the log level, it is not necessary to configure
           atta_warning: false  # [optional, default false] log level mapping, eagle eye warning is enabled
           field:  # [Optional, do not configure to pull from the remote] When applying for atta id, the table structure fields defined by the business itself must be in the same order
             - msg
             -uid
             -cmd
             - level
```

The log configuration example of [Zhiyan Log](https://git.woa.com/trpc-go/trpc-log-zhiyan) is as follows:

```yaml
plugins:
  log:  # log configuration supports multiple logs, you can log through log.Get("xxx").Debug
    default:  # Default log configuration, each log can support multiple outputs
      - writer: zhiyan  # zhiyan remote log output
        level: debug  # Remote log level
        remote_config:  # Remote log configuration, business custom structure, each remote log has its own independent configuration
          report_topic: 'fb-1ecfc3gefdca1f8'  # Zhiyan log topic, each business needs to apply for it
          report_proto: 'tcp'  # [optional, default tcp] log reporting protocol, supports tcp and udp
          host: '1.1.1.1'  # [Optional, the default is the IP reported by the local end] The Host field of the log report
          zhiyanobj_size: 3  # [Optional, default 3] The log is reported asynchronously, the number of goroutines consuming the log, one goroutinue and one obj, to avoid race conditions
          channel_block: false  # [Optional, default false] The log is reported asynchronously, whether the writing pipeline is blocked.
          channel_capacity: 10000  # [optional, default 10000] log pipeline capacity
```

# Interface usage

## Default Logger log function

The name of the default logger is fixed as "default", the default print level is debug, print console, and the print format is text format. Configuration can be modified in the framework file plugins.log. The function to print logs using the default Logger is as follows, and the function style is consistent with "fmt.Print()" and "fmt.Printf()".

```go
// Provide "fmt.Print()" style function interface for the default Logger
func Trace(args ... interface{})
func Debug(args...interface{})
func Info(args...interface{})
func Warn(args...interface{})
func Error(args...interface{})
func Fatal(args ... interface{})
// Provide "fmt.Printf()" style function interface for the default Logger
// Formatted printing follows the fmt.Printf() standard
func Tracef(format string, args ... interface{})
func Debugf(format string, args ... interface{})
func Infof(format string, args...interface{})
func Warnf(format string, args...interface{})
func Errorf(format string, args...interface{})
func Fatalf(format string, args ... interface{})

```

At the same time, the system also provides management functions of the default Logger, including obtaining, setting the default logger, and setting the printing level of the logger.

```go
// Get Logger by name
func Get(name string) Logger
// Set the specified Logger as the default Logger
func SetLogger(logger Logger)
// Set the log level of the specified writer under the default Logger, and the output is the subscript of the writer array "0" "1" "2"
func SetLevel(output string, level Level)
// Get the log level of the specified writer under the default Logger, and the output is the subscript of the writer array "0" "1" "2"
func GetLevel(output string) Level
```

## Context type log function

For the use of context type logs, each context logger must exclusively have one logger to ensure that the configuration of the context logger will not be tampered with. The framework provides `WithFields()` and `WithFieldsContext()` to inherit the parent logger configuration and generate a new logger.

```go
// Based on the default Logger, create a new Logger and add fields to the log printing of the new Logger.
// fields fields must appear in pairs, for example: logger := log.WithFields("key1","value1")
func WithFields(fields ... string) Logger
// Based on the Logger under the current context, create a new Logger and add fields to the log printing of the new Logger.
// fields fields must appear in pairs, for example: logger := log.WithFields("key1","value1")
func WithFieldsContext(ctx context.Context, fields...string) Logger {
```

Then set the Logger for the context by the following function:

```go
logger := ...
trpc.Message(ctx).WithLogger(logger)
```

The log printing function at the context level is similar to the default logger printing function:

```go
// Provide "fmt.Print()" style function to Context Logger
func TraceContext(args...interface{})
func DebugContext(args...interface{})
func InfoContext(args...interface{})
func WarnContext(args ... interface{})
func ErrorContext(args ... interface{})
func FatalContext(args ... interface{})
// Provide "fmt.Printf()" style function to Context Logger
// Formatted printing follows the fmt.Printf() standard
func TraceContextf(format string, args ... interface{})
func DebugContextf(format string, args ... interface{})
func InfoContextf(format string, args...interface{})
func WarnContextf(format string, args...interface{})
func ErrorContextf(format string, args ... interface{})
func FatalContextf(format string, args ... interface{})
```

## Specify Logger log interface

At the same time, the framework also provides functions for users to choose loggers by themselves. For each Logger implements the Logger Interface. The interface is defined as:

```go
type Logger interface {
     // The interface provides "fmt.Print()" style functions
     Trace(args...interface{})
     Debug(args...interface{})
     Info(args...interface{})
     Warn(args ... interface{})
     Error(args...interface{})
     Fatal(args...interface{})
     // The interface provides "fmt.Printf()" style functions
     Tracef(format string, args...interface{})
     Debugf(format string, args...interface{})
     Infof(format string, args...interface{})
     Warnf(format string, args ... interface{})
     Errorf(format string, args...interface{})
     Fatalf(format string, args ... interface{})
     // SetLevel sets the output log level
     SetLevel(output string, level Level)
     // GetLevel to get the output log level
     GetLevel(output string) Level
     // WithFields set some business custom data into each log: such as uid, imei and other fields must appear in pairs of kv
     WithFields(fields...string) Logger
}
```

Users can directly use the above interface functions to print logs, for example

```go
log. Get("custom"). Debug("message")
log.Get("custom").Debugf("hello %s", "terry")
```

## Framework Logs
1. The framework should log as little as possible and throw error up to user for handling.
2. Some underlying severe errors may print trace logs. To enable trace log, you must set ENV:
   ```bash
   export TRPC_LOG_TRACE=1
   ```

## Enabling `trace` level logging

To enable `trace` level logging, you must first ensure that the log level is set to `debug` or `trace` in the configuration. Then, you can enable `trace` through either environment variables or code:

1. Setting through environment variables

Add the following to the script that executes the server binary:

```shell
export TRPC_LOG_TRACE=1
./server -conf path/to/trpc_go.yaml
```

2. Setting through code

Add the following code:

```go
import "git.code.oa.com/trpc-go/trpc-go/log"

func init() {
    log.EnableTrace()
}
```

It is recommended to use the environment variable method as it is more flexible.

# Notes about `caller_skip`

Depending on how the `logger` is used, the `caller_skip` setting is also different:

## Default Logger:

```go
log.Debug("default logger") // use the default logger
```

At this time, the configuration used by the log is `default`:

```yaml
default:  # default log configuration, log.Debug("xxx")
  - writer: console  # Console standard output default
    level: debug  # Standard output log level
  - writer: file  # Local file log
    level: debug  # Local file rolling log level
    formatter: json  # Standard output log format
    writer_config:  # Local file output specific configuration
      filename: ../log/trpc_time.log  # Local file rolling log storage path
      write_mode: 2  # log write mode, 1-synchronous, 2-asynchronous, 3-extreme speed (asynchronous discard), do not configure the default asynchronous mode
      roll_type: time  # File roll type, time is roll by time
      max_age: 7  # Maximum log retention days
      max_backups: 10  # Maximum number of log files
      compress: false  # Whether the log file is compressed
      max_size: 10  # The size of the local file rolling log, in MB
      time_unit: day  # Rolling time interval, support: minute/hour/day/month/year
```

At this time, there is no need to pay attention to or set the value of `caller_skip`, the default value is 2, which means that there are two layers on `zap.Logger.Debug` (`trpc.log.Debug -> trpc.log.zapLog. Debug -> zap.Logger.Debug`)

## Put the custom logger into the context for use:

```go
trpc.Message(ctx).WithLogger(log.Get("custom"))
log.DebugContext(ctx, "custom log msg")
```

At this time, there is no need to pay attention to or set the value of `caller_skip`, the value is 2 by default, which means that there are two layers on `zap.Logger.Debug` (`trpc.log.DebugContext -> trpc.log.zapLog .Debug -> zap.Logger.Debug`)

The configuration example is as follows:

```yaml
custom:  # Business custom logger configuration, the name can be set at will, each service can have multiple loggers, you can use log.Get("custom").Debug("xxx") to log
  - writer: file  # Business-defined core configuration, the name can be set at will
    level: debug  # Business custom core output level
    writer_config:  # Local file output specific configuration
      filename: ../log/trpc1.log  # Local file rolling log storage path
```

## Do not use custom logger in context:

```go
log. Get("custom"). Debug("message")
```

At this time, you need to set the `caller_skip` value of `custom` logger to 1, because `log.Get("custom")` directly returns `trpc.log.zapLog`, call `trpc.log.zapLog.Debug` Only one layer on top of `zap.Logger.Debug` (`trpc.log.zapLog.Debug -> zap.Logger.Debug`)

The configuration example is as follows:

```yaml
custom:  # Business custom logger configuration, the name can be set at will, each service can have multiple loggers, you can use log.Get("custom").Debug("xxx") to log
  - writer: file  # Business-defined core configuration, the name can be set at will
    caller_skip: 1  # Caller for locating the log
    level: debug  # Business custom core output level
    writer_config:  # Local file output specific configuration
      filename: ../log/trpc1.log  # Local file rolling log storage path
```

Pay attention to the location of `caller_skip` (not in `writer_config`), and when there are `caller_skip` for multiple `writer`s, the value of `caller_skip` of the logger is subject to the last one, for example:

```yaml
custom:  # Business custom logger configuration, the name can be set at will, each service can have multiple loggers, you can use log.Get("custom").Debug("xxx") to log
  - writer: file  # Business-defined core configuration, the name can be set at will
    caller_skip: 1  # Caller for locating the log
    level: debug  # Business custom core output level
    writer_config:  # Local file output specific configuration
      filename: ../log/trpc1.log  # Local file rolling log storage path
  - writer: file  # Local file log
    caller_skip: 2  # The calling place used to locate the log
    level: debug  # Local file rolling log level
    writer_config:  # Local file output specific configuration
      filename: ../log/trpc2.log  # Local file rolling log storage path
```

Finally, the `caller_skip` value of the `custom` logger will be set to 2

**Note:** The above usage 2 and usage 3 are in conflict, only one of them can be used at the same time
