English | [中文](log.zh_CN.md)

# How to Develop a Writer Plugin for Log plugin

The log package provides a log plugin named "default", which supports configuring multiple Writers in a pluginized manner. 
This guide will introduce how to develop a Writer plugin that relies on configuration for loading, based on the "default" log plugin provided by the log package.
The following steps will use the "console" Writer provided by the log package as an example to illustrate the relevant development steps.

## 1. Determine the Configuration of the Plugin

The following is an example of the configuration for setting the "console" Writer plugin to the "default" log plugin with the name "console" in the "trpc_go.yaml" configuration file:

```yaml
plugins:
  log:
    default:
      - writer: console
        level: debug
        formatter: console
```

The complete configuration is as follows:

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

## 2. Implement the plugin.Factory Interface

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

## 3. Call `log.RegisterWriter` to Register the plugin with the log Package

```go
DefaultConsoleWriterFactory = &ConsoleWriterFactory{}
RegisterWriter(OutputConsole, DefaultConsoleWriterFactory)
```