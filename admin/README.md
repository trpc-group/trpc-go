English | [中文](README.zh_CN.md)

# Introduction

The management command (admin) is the internal management background of the service. It is an additional http service provided by the framework in addition to the normal service port. Through this http interface, instructions can be sent to the service, such as viewing the log level, dynamically setting the log level, etc. The specific command See the list of commands below.

Admin is generally used to query the internal status information of the service, and users can also define arbitrary commands.

Admin provides HTTP services to the outside world using the standard RESTful protocol.

By default, the framework does not enable the admin capability and needs to be configured to start it. When generating the configuration, admin can be configured by default, so that admin can be opened by default.
```yaml
server:
  app: app              # The application name of the business, be sure to change it to your own business application name
  server: server        # The process service name, be sure to change it to your own service process name
  admin:  
    ip: 127.0.0.1        # The IP address of admin, you can also configure the network card NIC
    port: 11014          # The port of admin, admin will only start when the IP and port are configured here at the same time
    read_timeout: 3000   # ms. Set the timeout for accepting requests and reading the request information completely to prevent slow clients
    write_timeout: 60000 # ms. Timeout for processing
```

# List of management commands

The following commands are already built into the framework. Note: the `IP:port` in the commands is the address configured in the admin, not the address configured in the service.

## View all management commands

```shell
curl http://ip:port/cmds
```
Return results

```shell
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

```shell
curl http://ip:port/version
```
Return results

```shell
{
  "errorcode": 0,
  "message": "",
  "version": "v0.1.0-dev"
}
```

## View framework log level

```shell
curl -XGET http://ip:port/cmds/loglevel?logger=xxx&output=0
```
Note: logger is used to support multiple logs. If not specified, it will use the default log of the framework. Output refers to different outputs under the same logger, with the array index starting at 0. If not specified, it will use the first output.

Return results

```shell
{
  "errorcode":0,
  "loglevel":"info",
  "message":""
}
```

## Set framework log level

(The value is the log level, and the possible values are: trace debug info warn error fatal)

```shell
curl http://ip:port/cmds/loglevel?logger=xxx -XPUT -d value="debug"
```
Note: The logger is used to support multiple logs. If not specified, the default log of the framework will be used. The 'output' parameter refers to different outputs under the same logger, with the array index starting at 0. If not specified, the first output will be used.

Note: This sets the internal memory data of the service and will not update the configuration file. It will become invalid after a restart.

Return results

```shell
{
  "errorcode":0,
  "level":"debug",
  "message":"",
  "prelevel":"info"
}
```

## View framework configuration file

```shell
curl http://ip:port/cmds/config
```
Return results

The 'content' parameter refers to the JSON-formatted content of the configuration file.

```shell
{
  "content":{

  },
  "errorcode":0,
  "message":""
}
```

# Customize management commands

## Define a function

First, define your own HTTP interface processing function, which can be defined in any file location.

```go
// load Trigger loading a local file to update a specific value in memory
func load(w http.ResponseWriter, r *http.Request) {
  reader, err := ioutil.ReadFile("xxx.txt")
  if err != nil {
    w.Write([]byte(`{"errorcode":1000, "message":"read file fail"}`))  // Define error codes and error messages by yourself
    return
  }
  
  // Business logic...
  
  // Return a success error code
  w.Write([]byte(`{"errorcode":0, "message":"ok"}`))
}
```

## Register a route

Register admin handle function after `trpc.NewServer`:

```go
import (
  "trpc.group/trpc-go/trpc-go"
  "trpc.group/trpc-go/trpc-go/admin"
)
func init() {
  s := trpc.NewServer()
  adminServer, err := trpc.GetAdminService(s)
  if err != nil { .. }
  adminServer.HandleFunc("/cmds/load", load)  // Define the path yourself, usually under /cmds. Be careful not to duplicate, otherwise they will overwrite each other.
}
```

## Trigger a command

Trigger the execution of a custom command

```shell
curl http://ip:port/cmds/load
```

# Pprof performance analysis

Pprof is a built-in performance analysis tool in Go language, which shares the same port number with the admin service by default. As long as the admin service is enabled, the pprof of the service can be used.

After configuring admin, there are a few ways to use pprof:

## To use pprof on a machine with a configured Go environment and network connectivity to the server

```shell
go tool pprof http://{$ip}:${port}/debug/pprof/profile?seconds=20
```

## To download pprof files to the local machine and analyze them using local Go tools

```shell
curl http://${ip}:{$port}/debug/pprof/profile?seconds=20 > profile.out
go tool pprof profile.out

curl http://${ip}:{$port}/debug/pprof/trace?seconds=20 > trace.out
go tool trace trace.out
```

# Memory management commands debug/pprof/heap

In addition, trpc-go will automatically remove the pprof route registered on the golang http package DefaultServeMux for you, avoiding the security issues of the golang net/http/pprof package (this is a problem of go itself).

Therefore, the service built using the trpc-go framework can directly use the pprof command, but the service started with the `http.ListenAndServe("xxx", xxx)` method will not be able to use the pprof command.

Perform memory analysis on intranet machines with the go command installed:

```shell
go tool pprof -inuse_space http://xxx:11029/debug/pprof/heap
go tool pprof -alloc_space http://xxx:11029/debug/pprof/heap
```
