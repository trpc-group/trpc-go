# background

It is essential to leverage logs to identify service transactions, anomalies, and errors during service delivery, and the log package is utilized to facilitate the logging of service activities.

Logs typically need to be output to standard output, log files, or other log collection systems, such as syslog or company-specific remote log platforms.

Through a plugin-based design, the logging system supports the output of log information to multiple endpoints. After comparing and evaluating commonly used open source logging libraries, zapcore was selected as the foundation for encapsulation due to its widespread usage and high performance.

# principle

Let's first examine the overall design of the logging system:
![logging system](/.resources/developer_guide/module_design/log/log.png)

The log defines a common interface for logging output, supporting multiple levels such as Trace, Debug, Info, Warning, Error, and Fatal.
Corresponding configuration field values and positions for 123 platform are: trace, error, info, debug.
plugins:
-log:
--default:
---level: info

Trpc-go provides a default implementation for logging output based on the [uber-go/zap](https://github.com/uber-go/zap) package, which supports dynamic adjustment of runtime log levels. For specific implementation details, please refer to: https://git.woa.com/trpc-go/trpc-go/blob/master/log/zaplogger.go

# configuration

The default configuration of the logging system supports outputting logs to both the console and files.

``` go
type Config struct {
    Outputs map[string]OutputConfig // writer name => output config
}
```
## OutputConfig

OutputConfig defines information such as logging output endpoints, logging output formats, and logging output levels. By default, the framework loads information from the trpc-go.yaml configuration file.

``` go
type OutputConfig struct {
    // Writer Logging output endpoint (console, file)
    Writer      string
    WriteConfig WriteConfig `yaml:"writer_config"`
    // Formatter Logging output format (console, json)
    Formatter    string
    FormatConfig FormatConfig `yaml:"formatter_config"`
    // Level Control logging level
    Level string
}
```
# usage

WithFields supports the setting of some business-specific data in each log, such as UID, etc.

``` go
m := make(map[string]string)
m["uid"] = "10012"
log.WithFields(m)
```
Output logs
``` go
log.Trace("helloworld")
log.Debug("helloworld")
log.Info("helloworld")
log.Warning("helloworld")
log.Error("helloworld")
```
## DyeingLog

Using the framework's own message dyeing mechanism to print dyeing logs.

``` go
msg := trpc.Message(ctx)
msg.WithDyeing(true) // Setting dyeing tags for requests, which will be automatically passed downstream.
msg.WithDyeingKey(key) // Setting dyeing keys, which can be used as keywords for querying dyeing logs.
```
Now, let's introduce two widely used dyeing logging systems in PCG: tpstelemetry and metis.

### tpstelemetryLog
By default, services deployed on the 123 platform have enabled tpstelemetry. To print tpstelemetry dyeing logs, you can use the [tlog library](https://git.woa.com/trpc-go/trpc-opentracing-tjg/tree/master/tlog/example)。

``` go
// Setting the query key allows users to query trace data in tpstelemetry for dyeing or sampling requests.
uin := "10000"
tlog.SetDyeingKey(ctx, uin)
// White-listed users can enable dyeing.
// For dyeing sampling request-related logs, they are reported to tpstelemetry regardless of the log level.
// For probability sampling request-related logs, they are reported to tpstelemetry according to the log level.
if uin == "10000" {
    tlog.SetDyeing(ctx)
}
tlog.Tracef(ctx, "helloworrld")
tlog.Debugf(ctx, "helloworrld")
tlog.Infof(ctx, "helloworrld")
tlog.Warningf(ctx, "helloworrld")
tlog.Errorf(ctx, "helloworrld")
```

The tpstelemetry query page:
http://tjg.oa.com

### metisDyeingLog

Metis dyeing log system is also widely used within pcg, and if you need to integrate with Metis logging, you can refer to the nlog library. This library integrates three types of log reporting: tpstelemetry/Metis/local log. If you only need Metis log reporting, you can extract it on your own.

Address of Metis dyeing log system:
http://metis.pcg.com
![MetisDyeingLog](/.resources/developer_guide/module_design/log/metis.png)
