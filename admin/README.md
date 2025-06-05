English | [中文](README.zh_CN.md)

- [Overview of Management Commands](#overview-of-management-commands)
- [Command List](#command-list)
  - [View all management commands](#view-all-management-commands)
  - [View framework version information](#view-framework-version-information)
  - [View framework log level](#view-framework-log-level)
  - [Set framework log level](#set-framework-log-level)
  - [View framework configuration file](#view-framework-configuration-file)
- [Custom Management Commands](#custom-management-commands)
  - [Define a function](#define-a-function)
  - [Register a route](#register-a-route)
  - [Trigger a command](#trigger-a-command)
- [pprof Performance Analysis](#pprof-performance-analysis)
  - [Use a machine with a configured Go environment and connected to the server's network](#use-a-machine-with-a-configured-go-environment-and-connected-to-the-servers-network)
  - [Use Official Flame Graph Proxy Service](#use-official-flame-graph-proxy-service)
  - [Download pprof files to the local machine and analyzing them with local Go tools](#download-pprof-files-to-the-local-machine-and-analyzing-them-with-local-go-tools)
  - [Memory management command debug/pprof/heap](#memory-management-command-debugpprofheap)
  - [View Flame Graphs on PCG 123 Release Platform](#view-flame-graphs-on-pcg-123-release-platform)
  - [Request Cost Measurement](#request-cost-measurement)
    - [Common RPC service request cost metrics](#common-rpc-service-request-cost-metrics)
    - [Streaming RPC Service Request Cost Metrics](#streaming-rpc-service-request-cost-metrics)
    - [ProfilerTagger Performance Tuning Example](#profilertagger-performance-tuning-example)

# Overview of Management Commands

Management commands (admin) are an internal management backend within a service. It is an additional HTTP service provided by the framework outside of the regular service ports. Through this HTTP interface, commands can be sent to the service, such as viewing log levels, dynamically setting log levels, and more. The specific commands can be found in the command list below.
Admin is generally used to query internal status information of the service, and users can also define custom commands.
Admin internally provides HTTP services to the outside world using the standard RESTful protocol.

By default, the framework does not enable the admin capability and requires configuration to start it (when generating the configuration, you can default to configuring admin so that admin is enabled by default):

```yaml
server:
    app: app       # The application name for the business, make sure to change it to your own application name
    server: server # The process service name, make sure to change it to your own service process name
    admin:
        ip: 127.0.0.1 # The IP address of the admin, can also configure the network card NIC
        port: 11014   # The port of the admin, both the IP and port need to be configured here to start admin
        read_timeout: 3000 # ms. The timeout for reading the complete request information after the request is accepted, to prevent slow clients
        write_timeout: 60000 # ms. The timeout for processing
```

# Command List

The framework has built-in the following commands. Note: The IP:port in the commands is the address configured in the admin configuration, not the address configured in the service configuration.

## View all management commands

```bash
curl "http://ip:port/cmds"
```

Response:

```json
{
    "cmds":[
        "/cmds",
        "/version",
        "/cmds/loglevel",
        "/cmds/config"
    ],
    "errorcode":0,
    "message":""
}
```

## View framework version information

```bash
curl "http://ip:port/version"
```

Response:

```json
{
    "errorcode": 0,
    "message": "",
    "version": "v0.1.0-dev"
}
```

## View framework log level

```bash
curl -XGET "http://ip:port/cmds/loglevel?logger=xxx&output=0"
```

Note: The logger is used to support multiple logs. If not provided, it refers to the default log of the framework. The output parameter refers to different outputs under the same logger, with array indices. If not provided, it refers

 to index 0, the first output.

Response:

```json
{
    "errorcode":0,
    "loglevel":"info",
    "message":""
}
```

**Note:** This method cannot determine whether the `trace` level is truly enabled, as the activation of the `trace` level also depends on the settings of environment variables or code.

## Set framework log level

(value is the log level, with values: trace, debug, info, warn, error, fatal)

```bash
curl "http://ip:port/cmds/loglevel?logger=xxx" -XPUT -d value="debug"
```

Note: The logger is used to support multiple logs. If not provided, it refers to the default log of the framework. The output parameter refers to different outputs under the same logger, with array indices. If not provided, it refers to index 0, the first output.
Note: This sets the internal in-memory data of the service, which will not be updated in the configuration file and will become invalid upon restart.

Response:

```json
{
    "errorcode":0,
    "level":"debug",
    "message":"",
    "prelevel":"info"
}
```

**Note:** In addition to setting the log level to `trace` or `debug` here, enabling `trace` level also requires setting the environment variable `export TRPC_LOG_TRACE=1` or adding the code `log.EnableTrace()`.

## View framework configuration file

```bash
curl "http://ip:port/cmds/config"
```

Response:

The content is the JSON representation of the configuration file content.

```json
{
    "content":{ 
        
    },
    "errorcode":0,
    "message":""
}
```

# Custom Management Commands

## Define a function

First, define your own processing function in the form of an HTTP interface. You can define it anywhere in your files:

```go
// load triggers loading specific values into memory from a local file
func load(w http.ResponseWriter, r *http.Request) {
    reader, err := ioutil.ReadFile("xxx.txt")
    if err != nil {
        w.Write([]byte(`{"errorcode":1000, "message":"read file fail"}`))  // Custom error code and message
        return
    }

    // Business logic...

    // Return a success error code
    w.Write([]byte(`{"errorcode":0, "message":"ok"}`))
}
```

## Register a route

Register the admin in the init function or your own internal function:

```go
import (
    "git.code.oa.com/trpc-go/trpc-go/admin"
)

func init() {
    admin.HandleFunc("/cmds/load", load)  // Define your own path, generally under /cmds, be careful not to overlap, otherwise they will override each other
}
```

## Trigger a command

Trigger the execution of a custom command:

```bash
curl "http://ip:port/cmds/load"
```

# pprof Performance Analysis

> v0.5.2~0.v0.18.3: trpc-go will automatically remove the pprof route registered on the golang http package DefaultServeMux to avoid the security issue of the golang net/http/pprof package (this is a problem with Go itself).
Therefore, services built using the trpc-go framework can directly use the pprof command, but services started using `http.ListenAndServe("xxx", xxx)` will not be able to use the pprof command.
If you must start the native HTTP service and want to use pprof to analyze memory, you can use `mux := http.NewServeMux()` instead of using the `http.DefaultServeMux`.

> v0.19.0: The pprof functionality supported by the admin package relies on the imported net/http/pprof package.However, the imported net/http/pprof package implicitly registers HTTP handlers for"/debug/pprof/", "/debug/pprof/cmdline", "/debug/pprof/profile", "/debug/pprof/symbol", "/debug/pprof/trace" in `http.DefaultServeMux` in its init function. This implicit behavior is too subtle and may contribute to people inadvertently leaving such endpoints open, and may cause security problems:<https://github.com/golang/go/issues/22085> if people use `http.DefaultServeMux`. So we decide to reset default serve mux to remove pprof registration. This requires making sure that people are not using `http.DefaultServeMux` before we reset it. In most cases, this works, which is guaranteed by the execution order of the init function. If you need to enable pprof on `http.DefaultServeMux` you need to register it explicitly after importing the admin package. Simply importing the net/http/pprof package anonymously will not work. More details see: <https://git.woa.com/trpc-go/trpc-go/issues/912>, and <https://github.com/golang/go/issues/42834>.

```go
http.DefaultServeMux.HandleFunc("/debug/pprof/", pprof.Index)
http.DefaultServeMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
http.DefaultServeMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
http.DefaultServeMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
http.DefaultServeMux.HandleFunc("/debug/pprof/trace", pprof.Trace)
```

pprof is a built-in performance analysis tool in the Go language. It shares the same port number with the admin service by default. As long as the admin service is enabled, the service's pprof can be used.

There are three ways to use pprof on an IDC machine that is configured with a Go environment and connected to the IDC network:

## Use a machine with a configured Go environment and connected to the server's network

```bash
go tool pprof http://{$ip}:${port}/debug/pprof/profile?seconds=20
```

## Use Official Flame Graph Proxy Service

The tRPC-Go team has set up an official flame graph proxy service. Simply enter the following address in your office network browser to view the flame graph of your own service, where the ip:port parameters are the admin address of your service:

```text
https://trpcgo.debug.woa.com/debug/proxy/profile?ip=${ip}&port=${port}
https://trpcgo.debug.woa.com/debug/proxy/heap?ip=${ip}&port=${port}
```

In addition, tRPC-Go team  has also set up the official go tool pprof web service from golang (owner: terrydang) with a UI interface:

```text
https://qqops.woa.com/pprof/
```

Note: If you have already connected to other flame graph proxy platforms (such as Galileo), using this flame graph proxy service may result in a `500 Internal Server Error`.
Please use the platform you have already connected to view the flame graph.

## Download pprof files to the local machine and analyzing them with local Go tools

```bash
curl "http://${ip}:{$port}/debug/pprof/profile?seconds=20" > profile.out
go tool pprof profile.out

curl "http://${ip}:{$port}/debug/pprof/trace?seconds=20" > trace.out
go tool trace trace.out
```

## Memory management command debug/pprof/heap

Perform memory analysis on a machine with the go command installed:

```bash
go tool pprof -inuse_space http://xxx:11029/debug/pprof/heap
go tool pprof -alloc_space http://xxx:11029/debug/pprof/heap
```

## View Flame Graphs on PCG 123 Release Platform

You can search for the ["View Flame Graphs"]( https://123.woa.com/v2/formal#/plugins-platform/detail?pluginID=10025) plugin in the [Plugin Market](https://123.woa.com/v2/formal#/plugins-platform/index?_tab_=pluginsMarket) on the PCG 123 release platform.

## Request Cost Measurement

When building and optimizing RPC services, understanding the runtime overhead is crucial. This can help us identify performance bottlenecks, optimize code, and improve the overall performance of the service. To achieve this, the framework provides `WithProfilerTagger` and `WithStreamProfilerTagger` to measure the cost of RPC service requests based on the type of RPC service.

`ProfilerTagger` provides a method for performance analysis at the Goroutine level. By adding labels to Goroutines, we can filter out more detailed information based on different labels when viewing the pprof graph. For example, when you find that the service response time is longer than expected, or the CPU usage is too high, you can use `ProfilerTagger` to add labels to Goroutines, understand the runtime overhead of each RPC request in more detail, and better optimize service performance.

### Common RPC service request cost metrics

Use the `server.WithProfilerTagger` option to specify a ProfilerTagger for normal RPC services.

The following example code specifies a ProfilerTagger for a normal RPC service.

```go
type tagger struct{}

func (t *tagger) Tag(ctx context.Context, req interface{}) (*server.ProfileLabel, error) {
    profileLabel := server.NewProfileLabel()
    profileLabel.Store("serviceName", "trpc.test.helloworld.Greeter")
    if helloRsp, ok := req.(*pb.HelloRequest); ok {
        profileLabel.Store("msg", helloRsp.GetMsg())
    }
    return profileLabel, nil
}

s := trpc.NewServer(server.WithProfilerTagger(&tagger{}))
```

We have implemented the ProfilerTagger interface for the tagger. Every RPC call, the Goroutine of the server-side processing function will carry two pairs of labels, with key value pairs as shown in the table below.

| label key     | label value                    |
| :------------ | ------------------------------ |
| `serviceName` | `trpc.test.helloworld.Greeter` |
| `msg`         | Message sent by the server     |

### Streaming RPC Service Request Cost Metrics

Use the `server.WithStreamProfilerTagger` option to specify a StreamProfilerTagger for a streaming RPC service.

The following example code specifies a StreamProfilerTagger for a streaming RPC service.

```go
type tagger struct {
}

// Tag a streaming RPC service call.
func (t *tagger) Tag(ctx context.Context, info *server.StreamServerInfo) (*server.ProfileLabel, error) {
    profileLabel := server.NewProfileLabel()
    profileLabel.Store("serviceName", "trpc.test.helloworld.Greeter")
    return profileLabel, nil
}

// Tag every RecvMsg call.
func (t *tagger) TagRecvMsg(ctx context.Context) (*server.ProfileLabel, error) {
    profileLabel := server.NewProfileLabel()
    profileLabel.Store("RecvMsg", "RecvMsgValue")
    return profileLabel, nil
}

// Tag every SendMsg call.
func (t *tagger) TagSendMsg(ctx context.Context, m interface{}) (*server.ProfileLabel, error) {
    profileLabel := server.NewProfileLabel()
    if rsp, ok := m.(*pb.HelloReply); ok {
        profileLabel.Store("SendMsg", rsp.GetMsg())
    }
    return profileLabel, nil
}

s := trpc.NewServer(server.WithStreamProfilerTagger(&tagger{}))
```

We have implemented the StreamProfilerTagger interface for the tagger, and each RPC call will carry three pairs of labels in the Goroutine of the server-side processing function. The key value pairs of the labels are shown in the table below.

| label key     | label value                    |
| :------------ | ------------------------------ |
| `serviceName` | `trpc.test.helloworld.Greeter` |
| `RecvMsg`     | `RecvMsgValue`                 |
| `SendMsg`     | Message sent by the server     |

### ProfilerTagger Performance Tuning Example

Suppose there is an RPC service method `Say`, the implementation logic is as follows.

```go
func (g *Greeter) Say(ctx context.Context, req *pb.SayRequest) (*pb.SayReply, error) {
    if req.GetMsg() == "hello" {
        // Redundant operation
        for i := 0; i < 1_000_000_000; i++ {
        }
    }
    // Normal operation
    for i := 0; i < 1_000_000_000; i++ {
    }
    return &pb.SayReply{}, nil
}
```

Use the `WithProfilerTagger` option to specify `ProfilerTagger` for the service.

```go
type tagger struct{}

func (t *tagger) Tag(ctx context.Context, req interface{}) (*server.ProfileLabel, error) {
    profileLabel := server.NewProfileLabel()
    if helloRsp, ok := req.(*pb.HelloRequest); ok {
        profileLabel.Store("msg", helloRsp.GetMsg())
    }
    return profileLabel, nil
}

s := trpc.NewServer(server.WithProfilerTagger(&tagger{}))
```

We have implemented the `ProfilerTagger` interface for the tagger. Every time an RPC call is made, the Goroutine of the server-side processing function will carry a label with the label key being `"msg"` and the value being the client request message.

After the server starts, the client continuously calls the server's `Say` method, and the request message randomly switches between `"hello"` and `"hi"`.

View the pprof information, you can observe that msg has two values `"hello"` and `"hi"`, where `"hello"` takes longer.

![pprof before optimize](../.resources/admin/pprof-before-optimize.png)

Analyzing the server's `Say` method, you can find that the implementation logic is redundant. The optimized code is as follows.

```go
func (g *Greeter) Say(ctx context.Context, req *pb.SayRequest) (*pb.SayReply, error) {
    // Normal operation
    for i := 0; i < 1_000_000_000; i++ {
    }
    return &pb.SayReply{}, nil
}
```

View the pprof information again, you can observe that the time consumption of `"hello"` and `"hi"` is basically the same, and the performance is optimized.

![pprof after optimize](../.resources/admin/pprof-after-optimize.png)
