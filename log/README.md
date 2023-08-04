# tRPC-Go log

## Log Configuration
```yaml
plugins:
  log:                                    # all logs are configured here
    default:                              # the default log cfg, log.Debug("xxx")
      - writer: console                   # default as stdout to console
        level: debug                      # log level
      - writer: file                      # write to local file
        level: debug                      # log level for local file
        formatter: json                   # the format of file log
        formatter_config:                 # cfg of format
          time_fmt: 2006-01-02 15:04:05   # time format of file log
          time_key: Time                  # prefix to log time, default as 'T'
          level_key: Level                # prefix to log level, default as 'L'
          name_key: Name                  # prefix to log name, default as 'N'
          caller_key: Caller              # prefix to log caller, default as 'C'
          function_key: Function          # prefix to log callee, print nothing on omitted.
          message_key: Message            # prefix to log message, default as 'M'
          stacktrace_key: StackTrace      # prefix to log stack trace, default as 'S'
        writer_config:                    # detail cfg of local file
          filename: ../log/trpc_size.log  # path of local log file
          write_mode: 3                   # write mod, 1-sync, 2-async, 3-fast(async drop), default as 3
          roll_type: size                 # roll type of log file, size means log file is rolled by file size
          max_age: 7                      # maximum log retention days
          max_backups: 10                 # maximum number of log files
          compress:  false                # whether the log file is compressed
          max_size: 10                    # log rolling size in MB
      - writer: file                      # write to local file
        level: debug                      # log level
        formatter: json                   # the format of file log
        writer_config:                    # detail cfg of local file
          filename: ../log/trpc_time.log  # path of local log file
          write_mode: 3                   # write mod, 1-sync, 2-async, 3-fast(async drop), default as 3
          roll_type: time                 # roll type of log file, time means log file is rolled by datetime
          max_age: 7                      # maximum log retention days
          max_backups: 10                 # maximum number of log files
          compress:  false                # whether the log file is compressed
          max_size: 10                    # log rolling size in MB
          time_unit: day                  # rolling time interval, support minute/hour/day/month/year
      - writer: atta                      # remote atta log
        remote_config:                    # remote configuration, this may be different for each remote log
          atta_id: '05e00006180'          # atta id for each business
          atta_token: '6851146865'        # atta token for each business
          message_key: msg                # atta field for log body
          field:                          # the user defined fields when apply atta id, the order must be respected
            - msg
            - uid
            - cmd
    # This is a user defined log with name custom.
    # You may have multiple logs, each accessible via log.Get("custom").Debug("xxx").
    custom:
      - writer: file                  # user defined writer for zap core
        caller_skip: 1                # skip caller stacks when print log file line number
        level: debug                  # log level
        writer_config:                # detail cfg of writer
          filename: ../log/trpc1.log  # path of local file log
      - writer: file                  # user defined local file log
        caller_skip: 1                # skip caller stacks when print log file line number
        level: debug                  # log level
        writer_config:                # detail cfg of writer
          filename: ../log/trpc2.log  # path of local file log
```

## Concepts
- logger: the interface to use log. Every log may have multiple output. In previous config, log.Debug is print to
  console and local file.
- log factory: the factory of log. Every log is a plugin, a server may have multiple log plugins. Log factory reads
  config and registers logger instances to framework. Without log factory, the log is print to console by default.
- writer factory: the log output factory. Every output stream is a plugin, a log may have multiple output. Output
  factory reads config and init the core instances.
- core: the output instance of zap log, such as console, local file, remote log, etc.
- zap: an opensource log implementation of uber. tRPC-Go uses zap log.
- with fields: set some user defined data to each log, such as uid, imei. It should be used at the beginning of each
  request.

## Plugin Instantiation Process
1. At first, tRPC-Go has registered default log factory, console writer factory and file writer factory.
2. Log factory parses logger config and traverses writer config.
3. Call writer factory and load writer config one by one.
4. Each writer factory instantiates core.
5. Multiple cores are composed to a logger.
6. Register logger to framework.

## Support Multiple Loggers
1. Register plugin at the beginning of main.
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
2. Define you own logger in config, such as custom.
3. Use Get in different business scenarios.
   ```go
   log.Get("custom").Debug("message")
   ```
4. DebugContext print default logger by default. To use log.XxxContext to print user defined logger, you should get
   logger at the begninning of request and set it to ctx. For example:
   ```go
   trpc.Message(ctx).WithLogger(log.Get("custom"))
   log.DebugContext(ctx, "custom log msg")
   ```

## framework logs
1. The framework should log as little as possible and throw error up to user for handling.
2. Some underlying severe errors may print trace logs. To enable trace log, you must set ENV:
   ```bash
   export TRPC_LOG_TRACE=1
   ```

## About `caller_skip`

These are common usages of `logger`:

1. Default Logger:
   ```go
   log.Debug("default logger") // use default logger
   ```
   It use following `default` config:
   ```yaml
   default:                              # the default log cfg, log.Debug("xxx")
     - writer: console                   # default stdout to console
       level: debug                      # log level
     - writer: file                      # write to local file
       level: debug                      # log level for local file
       formatter: json                   # the format of file log
       writer_config:                    # detail cfg of local file
         filename: ../log/trpc_time.log  # path of local log file
         write_mode: 3                   # write mod, 1-sync, 2-async, 3-fast(async drop), default as 3
         roll_type: time                 # roll type of log file, time means log file is rolled by datetime
         max_age: 7                      # maximum log retention days
         max_backups: 10                 # maximum number of log files
         compress:  false                # whether the log file is compressed
         max_size: 10                    # log rolling size in MB
         time_unit: day                  # rolling time interval, support minute/hour/day/month/year
   ```
   You don't need to set `caller_skip`. It has default value 2, which means `zap.Logger.Debug` is wrapped by two
   function calls: `trpc.log.Debug -> trpc.log.zapLog.Debug -> zap.Logger.Debug`.
2. Put user defined logger to context:
   ```go
   trpc.Message(ctx).WithLogger(log.Get("custom"))
   log.DebugContext(ctx, "custom log msg")
   ```
   You don't need to set `caller_skip` neither. It has default value 2, which means `zap.Logger.Debug` is wrapped by two
   function calls: `trpc.log.DebugContext -> trpc.log.zapLog.Debug -> zap.Logger.Debug`.  
   This is an example config:
   ```yaml
   # This is a user defined log with name custom.
   # You may have multiple logs, each accessible via log.Get("custom").Debug("xxx").
   custom:
     - writer: file                  # user defined local file log
       level: debug                  # log level
       writer_config:                # detail cfg of writer
         filename: ../log/trpc1.log  # path of local file log
   ```
3. Do not use user defined logger in context:
   ```go
   log.Get("custom").Debug("message")
   ```
   You should set `caller_skip` of `custom` logger to 1, since `log.Get("custom")` returns a `trpc.log.zapLog`, calling
   `trpc.log.zapLog.Debug` just wraps only one function call `trpc.log.zapLog.Debug -> zap.Logger.Debug`.  
   This is an example of config:
   ```yaml
   # This is a user defined log with name custom.
   # You may have multiple logs, each accessible via log.Get("custom").Debug("xxx").
   custom:
     - writer: file                  # user defined local file log
       caller_skip: 1                # skip caller stacks when print log file line number
       level: debug                  # log level
       writer_config:                # detail cfg of writer
         filename: ../log/trpc1.log  # path of local file log
   ```
   Take care of the position of `caller_skip`(do not put it under `writer_config`). If there are multiple `caller_skip`
   under each `writer`, only the last one takes effect. For example:
   ```yaml
   # This is a user defined log with name custom.
   # You may have multiple logs, each accessible via log.Get("custom").Debug("xxx").
   custom:
     - writer: file                  # user defined local file log
       caller_skip: 1                # skip caller stacks when print log file line number
       level: debug                  # log level
       writer_config:                # detail cfg of writer
         filename: ../log/trpc1.log  # path of local file log
     - writer: file                  # user defined local file log
       caller_skip: 2                # skip caller stacks when print log file line number
       level: debug                  # log level
       writer_config:                # detail cfg of writer
         filename: ../log/trpc2.log  # path of local file log
   ```
   The `caller_skip` of `custom` logger is 2, not 1.

__Cautious:__ The above two usages 2 and 3 are conflicting, you should use only one of them at the same time.

