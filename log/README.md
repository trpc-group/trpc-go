English | [中文](README.zh_CN.md)

# log

## Overview

Here is the simplest program that uses the `log` package:

```go
 // The code below is located in example/main.go
package main

import "trpc.group/trpc-go/trpc-go/log"

func main() {
    log.Info("hello, world")
}
```

As of this writing, it prints:

```
2023-09-07 11:46:40.905 INFO example/main.go:6 hello, world
```

The `Info` function prints out a message with a log level of Info using the default Logger in the `log` package.
Depending on the importance and urgency of the output message, the log package supports five other logging levels (Trace, Debug, Warn, Error, and Fatal) in addition to the Info level mentioned above.
The log contains the message "hello, world" and the log level "INFO", but also the print time "2023-09-07 11:46:40.905" and the call stack "example/main.go:6".

You can also use `Infof` to output the same log level, `Infof` is more flexible and allows you to print messages in the format you want.

`Infof` is more flexible, allowing you to print messages in the format you want.
```go
log.Infof("hello, %s", "world")
```

In addition, you can pass a series of key-value pairs to the `With` function to create a new `Logger` from the default `Logger`.
The new `Logger` will output the key-value pairs to the end of the message when it prints the log.

```go
logger := log.With(log.Field{Key: "user", Value: os.Getenv("USER")})
logger.Info("hello, world")
```

The output now looks like this:

```
2023-09-07 15:05:21.168 INFO example/main.go:12 hello, world {"user": "goodliu"}
```

As mentioned before, the `Info` function uses the default `Logger`.
You can explicitly get this Logger and call its methods:
```go
dl := log.GetDefaultLogger()
l := dl.With(log.Field{Key: "user", Value: os.Getenv("USER")})
l.Info("hello, world")
```

## Main Types

The `log` package contains two main types:

- `Logger` is the front end, providing output methods similar to `Info` and `Infof`, which you can use to generate logs.
- `Writer` is the back end, processing the logs generated by `Logger` and writing them to various logging service systems, such as the console, local files, and remote servers.

The `log` package supports setting up multiple independent Loggers, each of which can be configured with multiple independent Writers.
As shown in the diagram, this example contains three Loggers: "Default Logger", "Other Logger-1", and "Other Logger-2", with "Default Logger" being the default Logger built into the log package.
"Default Logger" contains three different Writers: "Console Writer", "File Writer", and "Remote Writer", with "Console Writer" being the default Writer of "Default Logger".
`Logger` and `Writer` are both designed as customizable plug-ins, and you can refer to [here](https://github.com/trpc-group/trpc-go/blob/main/docs/developer_guide/develop_plugins/log.md) for information on how to develop them.

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

First, we will introduce how to configure `Logger` and `Writer` in the configuration file.
And then we will introduce `Writer` and `Logger` separately in a bottom-up manner.

## Configure `Logger` and `Writer`

Since both Logger and Writer are implemented as plugins, their related configurations need to be placed under the `plugins` field.

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

The above configuration includes three Loggers named "default", "logger1", and "logger2".
The "default" Logger is the system default Logger, which is configured with Writers named "console", "file", and "remote". When no logging configuration is done, the logs are written to the console by default, with a log level of Debug and a text format printing method.
The corresponding configuration file is:

```yaml
plugins:
  log:
    default:
      - writer: console
        level: debug
        formatter: console
```

For the configuration parameters of Writer, the design is as follows:

| configuration item      | configuration item |  type  | default value | configuration explanation                                                                                                                                               |
| ----------------------- | ------------------ | :----: | :-----------: | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| writer                  | writer             | string |               | Mandatory Log Writer plug-in name, the framework supports "file, console" by default                                                                                    |
| writer                  | writer_config      | object |      nil      | only need to be set when the log Writer is "file"                                                                                                                       |
| writer                  | formatter          | string |      ""       | Log printing format, supports "console" and "json", and defaults to "console" when it is empty                                                                          |
| writer                  | formatter_config   | object |      nil      | zapcore Encoder configuration when log output, when it is empty, refer to the default value of formatter_config                                                         |
| writer                  | remote_config      | Object |      nil      | Remote log format The configuration format can be set at will by the third-party component.                                                                             |
| writer                  | level              | string |               | Mandatory When the log level is greater than or equal to the set level, output to the writer backend Value range: trace, debug, info, warn, error, fatal                |
| writer                  | caller_skip        |  int   |       2       | Used to control the nesting depth of the log function, if not filled or 0 is entered, the default is 2                                                                  |
| writer.formatter_config | time_fmt           | string |      ""       | Log output time format, empty default is "2006-01-02 15:04:05.000"                                                                                                      |
| writer.formatter_config | time_key           | string |      ""       | The name of the key when the log output time is output in Json, the default is "T", use "none" to disable this field                                                                                      |
| writer.formatter_config | level_key          | string |      ""       | The name of the key when the log level is output in Json, the default is "L", use "none" to disable this field                                                                                            |
| writer.formatter_config | name_key           | string |      ""       | The name of the key when the log name is output in Json, the default is "N", use "none" to disable this field                                                                                             |
| writer.formatter_config | caller_key         | string |      ""       | The name of the log output caller's key when outputting in Json, default is "C", use "none" to disable this field                                                                                            |
| writer.formatter_config | message_key        | string |      ""       | The name of the key when the log output message body is output in Json, the default is "M", use "none" to disable this field                                                                              |
| writer.formatter_config | stacktrace_key     | string |      ""       | The name of the key when the log output stack is output in Json, default is "S", use "none" to disable this field                                                                                            |
| writer.writer_config    | log_path           | string |               | Mandatory Log path name, for example: /usr/local/trpc/log/                                                                                                              |
| writer.writer_config    | filename           | string |               | Mandatory Log file name, for example: trpc.log                                                                                                                          |
| writer.writer_config    | write_mode         |  int   |       0       | Log writing mode, 1-synchronous, 2-asynchronous, 3-extreme speed (asynchronous discard), do not configure the default extreme speed mode, that is, asynchronous discard |
| writer.writer_config    | roll_type          | string |      ""       | File roll type, "size" splits files by size, "time" splits files by time, and defaults to split by size when it is empty                                                |
| writer.writer_config    | max_age            |  int   |       0       | The maximum log retention time, 0 means do not clean up old files                                                                                                       |
| writer.writer_config    | time_unit          | string |      ""       | Only valid when split by time, the time unit of split files by time, support year/month/day/hour/minute, the default value is day                                       |
| writer.writer_config    | max_backups        |  int   |       0       | The maximum number of files in the log, 0 means not to delete redundant files                                                                                           |
| writer.writer_config    | compress           |  bool  |     false     | Whether to compress the log file, default is not compressed                                                                                                             |
| writer.writer_config    | max_size           | string |      ""       | Only valid when splitting by size, the maximum size of the log file (in MB), 0 means not rolling by size                                                                |

## Multiple Writers

Multiple Writers can provide the following features:

- Support for logging to multiple output backends simultaneously, such as printing to the console and saving to a local log file at the same time.
- Support for setting log levels individually for each output backend, for example, printing debug-level logs to the console and saving warning-level logs to a log file.
- Support for setting log formats (console, JSON, etc.) and log field names individually for each output backend.
- Support for log file rolling, including splitting log files by size or time, for file type backends.

It should be emphasized here that **settings of log are configured at the Writer level**.
You need to configure each output separately.
For example, if you want to print debug-level logs to the console and save warning-level logs to a log file, you must configure both the console and file Writers separately.

The system defaults to supporting two Writers: **"console"** and **"file"**.

### Write logs to the console

When the writer is set to "console", it means that the log will be written to the console.
Here's an example configuration:

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

### Write logs to a local file.

When the writer is set to "file", it means that the log is written to a local log file.
The configuration example of log file rolling storage according to time is as follows:

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

### Write logs to a remote location

To write logs to a remote location, you need to set the `remote_config` field.
The default Logger configuration for two remote Writers is shown below:

```yaml
plugins:
  log: # log configuration supports multiple logs, you can log through log.Get("xxx").Debug
    default: # Default log configuration, each log can support multiple writers
      - writer: remote-writer1  # remote writer named remote-writer1
        level: debug # log level of remote log
        remote_config: # remote log configuration, a custom structure for each independent remote log
          # relevant configuration
      - writer: remote-writer2 # remote writer named remote-writer2
        level: info # log level of remote log
        remote_config:  # remote log configuration, a custom structure for each independent remote log
         # relevant configuration
```

## Multiple Loggers

`log` package supports multiple loggers at the same time.
Each logger can set different log levels, print formats, and writers.
You can set different loggers for different application scenarios, for example:

- Different application modules use different log files for storage.
- Different events, based on the degree of attention of the event, collect logs using different log levels.

The multi-loggers function greatly increases the flexibility of log usage.

### Configure logger

Configure your logger in a configuration file, for example, configuring a logger named "custom":

```yaml
plugins:
  log:  # All log configuration
    default:  # Default log configuration ,log.Debug("xxx")
      - writer: console  # console stdout default
        level: debug  # The level of standard output logging
    custom:  # Your custom logger configuration, the name can be set at will, each service can have multiple loggers, you can use log.Get("custom").Debug("xxx") to log
      - writer: file                       # Your custom core configuration, the name can be set at will
        caller_skip: 1  # The call site used to locate the log
        level: debug  # Your custom core output level
        writer_config:  # Local file output specific configuration
          filename: ../log/trpc1.log  # The path where the local file rolling log is stored
```

For questions about `caller_skip` in the configuration file, see Chapter Explanation about `caller_skip`.


### Register the logger plugin

Register the logging plugin at the main function entry point:

```go
import (
    "trpc.group/trpc-go/trpc-go/log"
    "trpc.group/trpc-go/trpc-go/plugin"
)
func main() {
    // Note: plugin.Register should be executed before trpc.NewServer.
    plugin.Register("custom", log.DefaultLogFactory)
    s := trpc.NewServer()
}
```

### Get logger

The log package provides the following two ways for you to get a Logger:

#### Method 1: Specify logger directly

```go
// use a logger named "custom"
// Note: log.Get should be executed after trpc.NewServer, as plugin loading happens within trpc.NewServer.
log.Get("custom").Debug("message")
```

#### Method 2: Specify a logger for the context and use the context type log interface

```go
// Set the logger of the ctx context to custom
trpc.Message(ctx).WithLogger(log.Get("custom"))
// Use the logging interface of type "Context"
log.DebugContext(ctx, "custom log msg")
```

### Log Levels

According to the importance and urgency of the output messages, the log package provides six levels of logging, which are divided as follows from lowest to highest:

1. Trace: This is the lowest level, usually used to record all running information of the program, including some details and debugging information. This level of logging is usually only used in the development and debugging phase because it may generate a large amount of log data.
2. Debug: This level is mainly used in the debugging process to provide detailed information about program execution, helping you find the cause of problems.
3. Info: This level is used to record the general operation of the program, such as user login, system status updates, etc. These pieces of information are helpful in understanding the system's running status and performance.
4. Warn: The warning level indicates possible problems that will not immediately affect the program's functionality, but may cause errors in the future. This level of logging can help you discover and prevent problems in advance.
5. Error: The error level indicates serious problems that may prevent the program from executing certain functions. This level of logging requires immediate attention and handling.
6. Fatal: The fatal error level indicates very serious errors that may cause the program to crash. This is the highest log level, indicating a serious problem that needs to be addressed immediately.

7. Using log levels correctly can help you better understand and debug your application program.

### Log printing interface

The `log` package provides 3 sets of log printing interfaces:

- Log function of Default Logger: the most frequently used method.
  Directly use the default Logger for log printing, which is convenient and simple.
- Log function based on Context Logger: Provide a specified logger for a specific scenario and save it in the context, and then use the current context logger for log printing. This method is especially suitable for the RPC call mode: when the service receives the RPC request, set the logger for ctx and attach the field information related to this request, and the subsequent log report of this RPC call will bring the previously set field information
- Log function of the specified Logger: it can be used for users to select logger by themselves, and call the interface function of logger to realize log printing.


#### Log function of Default Logger

The name of the default logger is fixed as "default", the default print level is debug, print console, and the print format is text format.
Configuration can be modified in the framework file plugins.log.
The function to print logs using the default Logger is as follows, and the function style is consistent with "fmt.Print()" and "fmt.Printf()".

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

#### Log function based on Context Logger

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

#### Log function of the specified Logger

At the same time, the framework also provides functions for users to choose loggers by themselves.
For each Logger implements the Logger Interface.
The interface is defined as:

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

     // WithFields set some your custom data into each log: such as uid, imei and other fields must appear in pairs of kv
     WithFields(fields...string) Logger
}
```

Users can directly use the above interface functions to print logs, for example

```go
log.Get("custom").Debug("message")
log.Get("custom").Debugf("hello %s", "world")
```

## Framework Logs

1. The framework should log as little as possible and throw error up to user for handling.
2. Some underlying severe errors may print trace logs.
   To enable trace log, you need to set `TRPC_LOG_TRACE` environment variable:

```bash
   export TRPC_LOG_TRACE=1
```

### Enabling `trace` level logging

To enable `trace` level logging, you must first ensure that the log level is set to `debug` or `trace` in the configuration.
Then, you can enable `trace` through either environment variables:

- Setting through environment variables

Add the following to the script that executes the server binary:

```shell
export TRPC_LOG_TRACE=1
./server -conf path/to/trpc_go.yaml
```

- Setting through code

Add the following code:

```go
import "trpc.group/trpc-go/trpc-go/log"

func init() {
    log.EnableTrace()
}
```

It is recommended to use the environment variable method as it is more flexible.

## Notes about `caller_skip`

Depending on how the `logger` is used, the `caller_skip` setting is also different:

### Usage 1: Use Default Logger

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

### Usage 2: Put the custom logger into the context

```go
trpc.Message(ctx).WithLogger(log.Get("custom"))
log.DebugContext(ctx, "custom log msg")
```

At this time, there is no need to pay attention to or set the value of `caller_skip`, the value is 2 by default, which means that there are two layers on `zap.Logger.Debug` (`trpc.log.DebugContext -> trpc.log.zapLog .Debug -> zap.Logger.Debug`).

The configuration example is as follows:

```yaml
custom:  # Your custom logger configuration, the name can be set at will, each service can have multiple loggers, you can use log.Get("custom").Debug("xxx") to log
  - writer: file  # Your core configuration, the name can be set at will
    level: debug  # Your custom core output level
    writer_config:  # Local file output specific configuration
      filename: ../log/trpc1.log  # Local file rolling log storage path
```

### Do not use custom logger in context:

```go
log.Get("custom").Debug("message")
```

At this time, you need to set the `caller_skip` value of `custom` logger to 1, because `log.Get("custom")` directly returns `trpc.log.zapLog`, call `trpc.log.zapLog.Debug` Only one layer on top of `zap.Logger.Debug` (`trpc.log.zapLog.Debug -> zap.Logger.Debug`).

The configuration example is as follows:

```yaml
custom:  # Your custom logger configuration, the name can be set at will, each service can have multiple loggers, you can use log.Get("custom").Debug("xxx") to log
  - writer: file  # Your core configuration, the name can be set at will
    caller_skip: 1  # Caller for locating the log
    level: debug  # Your custom core output level
    writer_config:  # Local file output specific configuration
      filename: ../log/trpc1.log  # Local file rolling log storage path
```

Pay attention to the location of `caller_skip` (not in `writer_config`), and when there are `caller_skip` for multiple `writer`s, the value of `caller_skip` of the logger is subject to the last one, for example:

```yaml
custom:  # Your custom logger configuration, the name can be set at will, each service can have multiple loggers, you can use log.Get("custom").Debug("xxx") to log
  - writer: file  # Your core configuration, the name can be set at will
    caller_skip: 1  # Caller for locating the log
    level: debug  # Your custom core output level
    writer_config:  # Local file output specific configuration
      filename: ../log/trpc1.log  # Local file rolling log storage path
  - writer: file  # Local file log
    caller_skip: 2  # The calling place used to locate the log
    level: debug  # Local file rolling log level
    writer_config:  # Local file output specific configuration
      filename: ../log/trpc2.log  # Local file rolling log storage path
```

Finally, the `caller_skip` value of the `custom` logger will be set to 2.

**Note:** The above usage 2 and usage 3 are in conflict, only one of them can be used at the same time.