# 怎么为 log 类型的插件开发一个 Writer 插件 

`log` 包提供了一个名为 “default” 的 log 插件， 该插件支持以插件的形式配置多个 Writer。
本指南将在 “default” log 插件的基础上，介绍如何开发一个依赖配置进行加载的 Writer 插件。
下面以 `log` 包提供的名为 “console” 的 Writer 为例，来介绍相关开发步骤。

## 1. 确定插件的配置

下面是在 "trpc_go.yaml" 配置文件中为名字是 “default” 的 log 插件设置名字为 “console” 的 Writer 插件的配置示例：

```yaml
plugins:
  log:
    default:
      - writer: console
        level: debug
        formatter: console
```

完整的配置如下：

```go
// Config is the log config. Each log may have multiple outputs.
type Config []OutputConfig

// OutputConfig is the output config, includes console, file and remote.
type OutputConfig struct {
    // Writer is the output of log, such as console or file.
    Writer      string      `yaml:"writer"`
    WriteConfig WriteConfig `yaml:"writer_config"`

    // Formatter is the format of log, such as console or json.
    Formatter    string       `yaml:"formatter"`
    FormatConfig FormatConfig `yaml:"formatter_config"`

    // RemoteConfig is the remote config. It's defined by business and should be registered by
    // third-party modules.
    RemoteConfig yaml.Node `yaml:"remote_config"`

    // Level controls the log level, like debug, info or error.
    Level string `yaml:"level"`
    
    // CallerSkip controls the nesting depth of log function.
    CallerSkip int `yaml:"caller_skip"`
    
    // EnableColor determines if the output is colored. The default value is false.
    EnableColor bool `yaml:"enable_color"`
}

// WriteConfig is the local file config.
type WriteConfig struct {
    // LogPath is the log path like /usr/local/trpc/log/.
    LogPath string `yaml:"log_path"`
    // Filename is the file name like trpc.log.
    Filename string `yaml:"filename"`
    // WriteMode is the log write mod. 1: sync, 2: async, 3: fast(maybe dropped), default as 3.
    WriteMode int `yaml:"write_mode"`
    // RollType is the log rolling type. Split files by size/time, default by size.
    RollType string `yaml:"roll_type"`
    // MaxAge is the max expire times(day).
    MaxAge int `yaml:"max_age"`
    // MaxBackups is the max backup files.
    MaxBackups int `yaml:"max_backups"`
    // Compress defines whether log should be compressed.
    Compress bool `yaml:"compress"`
    // MaxSize is the max size of log file(MB).
    MaxSize int `yaml:"max_size"`
    
    // TimeUnit splits files by time unit, like year/month/hour/minute, default day.
    // It takes effect only when split by time.
    TimeUnit TimeUnit `yaml:"time_unit"`
}

// FormatConfig is the log format config.
type FormatConfig struct {
    // TimeFmt is the time format of log output, default as "2006-01-02 15:04:05.000" on empty.
    TimeFmt string `yaml:"time_fmt"`
    
    // TimeKey is the time key of log output, default as "T".
    TimeKey string `yaml:"time_key"`
    // LevelKey is the level key of log output, default as "L".
    LevelKey string `yaml:"level_key"`
    // NameKey is the name key of log output, default as "N".
    NameKey string `yaml:"name_key"`
    // CallerKey is the caller key of log output, default as "C".
    CallerKey string `yaml:"caller_key"`
    // FunctionKey is the function key of log output, default as "", which means not to print
    // function name.
    FunctionKey string `yaml:"function_key"`
    // MessageKey is the message key of log output, default as "M".
    MessageKey string `yaml:"message_key"`
    // StackTraceKey is the stack trace key of log output, default as "S".
    StacktraceKey string `yaml:"stacktrace_key"`
}
```

```go
const (
    pluginType        = "log"
    OutputConsole = "console"
)
```

## 2. 实现 `plugin.Factory` 接口

```go
// ConsoleWriterFactory is the console writer instance.
type ConsoleWriterFactory struct {
}

// Type returns the log plugin type.
func (f *ConsoleWriterFactory) Type() string {
    return pluginType
}

// Setup starts, loads and registers console output writer.
func (f *ConsoleWriterFactory) Setup(name string, dec plugin.Decoder) error {
    if dec == nil {
        return errors.New("console writer decoder empty")
    }
    decoder, ok := dec.(*Decoder)
    if !ok {
        return errors.New("console writer log decoder type invalid")
    }
    cfg := &OutputConfig{}
    if err := decoder.Decode(&cfg); err != nil {
        return err
    }
    decoder.Core, decoder.ZapLevel = newConsoleCore(cfg)
    return nil
}

func newConsoleCore(c *OutputConfig) (zapcore.Core, zap.AtomicLevel) {
    lvl := zap.NewAtomicLevelAt(Levels[c.Level])
    return zapcore.NewCore(
        newEncoder(c),
        zapcore.Lock(os.Stdout),
        lvl), lvl
}
```

## 3. 调用 `log.RegisterWriter` 把插件自己注册到 `log` 包

```go
DefaultConsoleWriterFactory = &ConsoleWriterFactory{}
RegisterWriter(OutputConsole, DefaultConsoleWriterFactory)
```