[TOC]

<!-- TOC -->

- [1 Introduction](#1-introduction)
- [2 List of management commands](#2-list-of-management-commands)
   - [2.1 View all management commands](#21-view-all-management-commands)
   - [2.2 View framework version information](#22-view-framework-version-information)
   - [2.3 View framework log level](#23-view-framework-log-level)
   - [2.4 Set framework log level](#24-set-framework-log-level)
   - [2.5 View framework configuration file](#25-view-framework-configuration-file)
- [3 Customize management commands](#3-customize-management-commands)
   - [3.1 Define a function](#31-define-a-function)
   - [3.2 Register a route](#32-register-a-route)
   - [3.3 Trigger a command](#33-trigger-a-command)
- [4 Pprof performance analysis](#4-pprof-performance-analysis)
   - [4.1 To use pprof on a machine with a configured Go environment and network connectivity to the IDC](#41-to-use-pprof-on-a-machine-with-a-configured-go-environment-and-network-connectivity-to-the-idc)
   - [4.2 To download pprof files to the local machine and analyze them using local Go tools](#42-to-download-pprof-files-to-the-local-machine-and-analyze-them-using-local-go-tools)
   - [4.3 The official flame graph proxy service](#43-the-official-flame-graph-proxy-service)
   - [4.4 Memory management commands debug/pprof/heap](#44-memory-management-commands-debugpprofheap)
   - [4.5 PCG 123 release platform view flame graph](#45-pcg-123-release-platform-view-flame-graph)
- [5 FAQ](#5-faq)
- [6 OWNER](#6-owner)
  - [nickzydeng](#nickzydeng)
  - [leoxhyang (For PCG 123 platform flame graph issues, please contact leoxhyang)](#leoxhyangfor-pcg-123-platform-flame-graph-issues-please-contact-leoxhyang)
<!-- /TOC -->

# Introduction

The management command (admin) is the internal management background of the service. It is an additional http service provided by the framework in addition to the normal service port. Through this http interface, instructions can be sent to the service, such as viewing the log level, dynamically setting the log level, etc. The specific command See the list of commands below.

Admin is generally used to query the internal status information of the service, and users can also define arbitrary commands.

Admin provides HTTP services to the outside world using the standard RESTful protocol.

By default, the framework does not enable the admin capability and needs to be configured to start it. When generating the configuration, admin can be configured by default, so that admin can be opened by default.
```
server:
  app: app              # The application name of the business, be sure to change it to your own business application name
  server: server        # The process service name, be sure to change it to your own service process name
  admin:  
    ip: 127.0.0.1        # The IP address of admin, you can also configure the network card NIC
    port: 11014          # The port of admin, admin will only start when the IP and port are configured here at the same time
    read_timeout: 3000   # ms. Set the timeout for accepting requests and reading the request information completely to prevent slow clients
    write_timeout: 60000 # ms. Timeout for processing
```
Example configuration for PCG's 123 release platform:

```
server:                                   # Server configuration
  app: ${app}                             # Application name for the business
  server: ${server}                       # Process service name
  bin_path: /usr/local/trpc/bin/          # Path to the binary executable file and framework configuration file
  conf_path: /usr/local/trpc/conf/        # Path to the business configuration file
  data_path: /usr/local/trpc/data/        # Path to the business data file
  admin:
    ip: ${local_ip}         # ip  local_ip  trpc_admin_ip either one is fine
    port: ${ADMIN_PORT}     #
    read_timeout: 3000      # ms. Set the timeout for accepting requests and reading the request information completely to prevent slow clients
    write_timeout: 60000    # ms. Timeout for processing
    enable_tls: false       # Enable TLS or not. Currently not supported
```

# List of management commands

The following commands are already built into the framework. Note: the `IP:port` in the commands is the address configured in the admin, not the address configured in the service.

## View all management commands

```
curl http://ip:port/cmds
```
Return results

```
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

```
curl http://ip:port/version
```
Return results

```
{
  "errorcode": 0,
  "message": "",
  "version": "v0.1.0-dev"
}
```

## View framework log level

```
curl -XGET http://ip:port/cmds/loglevel?logger=xxx&output=0
```
Note: logger is used to support multiple logs. If not specified, it will use the default log of the framework. Output refers to different outputs under the same logger, with the array index starting at 0. If not specified, it will use the first output.

Return results

```
{
  "errorcode":0,
  "loglevel":"info",
  "message":""
}
```

## Set framework log level

(The value is the log level, and the possible values are: trace debug info warn error fatal)

```
curl http://ip:port/cmds/loglevel?logger=xxx -XPUT -d value="debug"
```
Note: The logger is used to support multiple logs. If not specified, the default log of the framework will be used. The 'output' parameter refers to different outputs under the same logger, with the array index starting at 0. If not specified, the first output will be used.

Note: This sets the internal memory data of the service and will not update the configuration file. It will become invalid after a restart.

Return results

```
{
  "errorcode":0,
  "level":"debug",
  "message":"",
  "prelevel":"info"
}
```

## View framework configuration file

```
curl http://ip:port/cmds/config
```
Return results

The 'content' parameter refers to the JSON-formatted content of the configuration file.

```
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

```
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

Register admin in the init function or in your own internal function:

```
import (
  "git.code.oa.com/trpc-go/trpc-go/admin"
)
func init() {
  admin.HandleFunc("/cmds/load", load)  // Define the path yourself, usually under /cmds. Be careful not to duplicate, otherwise they will overwrite each other.
}
```

## Trigger a command

Trigger the execution of a custom command

```
curl http://ip:port/cmds/load
```

# Pprof performance analysis

Pprof is a built-in performance analysis tool in Go language, which shares the same port number with the admin service by default. As long as the admin service is enabled, the pprof of the service can be used.

After configuring admin on the IDC machine, there are three ways to use pprof:

## To use pprof on a machine with a configured Go environment and network connectivity to the IDC

```
go tool pprof http://{$ip}:${port}/debug/pprof/profile?seconds=20
```

## To download pprof files to the local machine and analyze them using local Go tools

```
curl http://${ip}:{$port}/debug/pprof/profile?seconds=20 > profile.out
go tool pprof profile.out

curl http://${ip}:{$port}/debug/pprof/trace?seconds=20 > trace.out
go tool trace trace.out
```

## The official flame graph proxy service

The tRPC-Go official flame graph proxy service is already set up, and you can view your service's flame graph by entering the following address in your office network browser, where the ipport parameter is the admin address of your service.

```
https://trpcgo.debug.woa.com/debug/proxy/profile?ip=${ip}&port=${port}
https://trpcgo.debug.woa.com/debug/proxy/heap?ip=${ip}&port=${port}
```

In addition, the official go tool pprof web service of golang has been built, (owner: terrydang) has a ui interface.
```

https://qqops.woa.com/pprof/
```

## Memory management commands debug/pprof/heap

From version 0.4.0 to version 0.5.1, trpc did not integrate pprof memory analysis commands due to security issues. Version 0.5.2 solves security issues and reintegrates pprof memory analysis commands.

So, if you want to use /debug/pprof/heap, please make sure that trpc-go has been updated to the latest version.

In addition, trpc-go will automatically remove the pprof route registered on the golang http package DefaultServeMux for you, avoiding the security issues of the golang net/http/pprof package (this is a problem of go itself).

Therefore, the service built using the trpc-go framework can directly use the pprof command, but the service started with the `http.ListenAndServe("xxx", xxx)` method will not be able to use the pprof command.
If you must start a native http service and use pprof to analyze memory, you can use `mux := http.NewServeMux()` instead of `http.DefaultServeMux`.

Perform memory analysis on intranet machines with the go command installed:

```
go tool pprof -inuse_space http://xxx:11029/debug/pprof/heap
go tool pprof -alloc_space http://xxx:11029/debug/pprof/heap
```

## PCG 123 release platform view flame graph

For users of the 123 platform, it can be directly integrated into the page button, which is more convenient to use. First, the user needs to install the flame graph plug-in. Please follow the steps below for a simple operation. You only need to install it once, and each service will be enabled or disabled according to actual needs. That's it.

Note: only tRPC-Go has a flame graph, trpc does not have other languages! !

1, Open the 123 platform, move the mouse to the personal avatar in the upper right corner, and click "Plugin Center"
!['install_plugin_step_1.png'](/.resources/user_guide/admin/install_plugin_step_1.png)

2、Click "Install Service Plug-in", and the interface for selecting a plug-in will pop up. Search and query "flame graph" for common plug-ins, and click Install
!['install_plugin_step_2.png'](/.resources/user_guide/admin/instll_plugin_step_2.png)

3, After the installation is successful, click the "Add" button, and an interface will pop up for the user to choose the service that needs to install the plug-in. You can choose multiple services to install together (note that the environment is distinguished here, and the same service needs to be selected separately in different environments)
!['install_plugin_step_3.png'](/.resources/user_guide/admin/instll_plugin_step_3.png)

4, After adding the service, click the specific service name in the above picture to view the effect after installation. At this time, move the mouse to the "More Operations" on the right side of each row of the node list, and the "View Flame Graph" operation item just installed will appear , and then click the button to generate the flame graph
!['install_plugin_step_4.png'](/.resources/user_guide/admin/instll_plugin_step_4.png)

# FAQ

<todo>

# OWNER

## nickzydeng

## leoxhyang (For PCG 123 platform flame graph issues, please contact leoxhyang)