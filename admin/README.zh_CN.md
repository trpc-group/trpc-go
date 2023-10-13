[English](README.md) | 中文

# 前言

管理命令（admin）是服务内部的管理后台，它是框架在普通服务端口之外额外提供的 http 服务，通过这个 http 接口可以给服务发送指令，如查看日志等级，动态设置日志等级等，具体命令可以看下面的命令列表。

admin 一般用于查询服务内部状态信息，用户也可以自己定义任意的命令。 

admin 内部使用标准 restful 协议对外提供 http 服务。

框架默认不会开启 admin 能力，需要配置才会启动（生成配置时，可以默认配置好 admin，这样就能默认打开 admin 了）：

```yaml
server:
  app: app       # 业务的应用名，注意要改成你自己的业务应用名
  server: server # 进程服务名，注意要改成你自己的服务进程名
  admin:
    ip: 127.0.0.1 # admin 的 ip，配置网卡 nic 也可以
    port: 11014   # admin 的 port，必须同时配置这里的 ip port 才会启动 admin
    read_timeout: 3000 # ms. 请求被接受到请求信息被完全读取的超时时间设置，防止慢客户端
    write_timeout: 60000 # ms. 处理的超时时间
```

# 管理命令列表

框架已经内置以下命令，注意：命令中的`ip:port`是上述 admin 配置的地址，不是 service 配置的地址。

## 查看所有管理命令

```shell
curl http://ip:port/cmds
```
返回结果
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

## 查看框架版本信息

```shell
curl http://ip:port/version
```
返回结果
```shell
{
  "errorcode": 0,
  "message": "",
  "version": "v0.1.0-dev"
}
```

## 查看框架日志级别

```shell
curl -XGET http://ip:port/cmds/loglevel?logger=xxx&output=0
```
说明：logger 是为了支持多日志，不填即为框架的 default 日志，output 同一个 logger 下的不同输出，数组下标，不填即为 0，第一个 output。

返回结果
```shell
{
  "errorcode":0,
  "loglevel":"info",
  "message":""
}
```

## 设置框架日志级别

(value 为日志级别，值为：trace debug info warn error fatal)
```shell
curl http://ip:port/cmds/loglevel?logger=xxx -XPUT -d value="debug"
```
说明：logger 是为了支持多日志，不填即为框架的 default 日志，output 同一个 logger 下的不同输出，数组下标，不填即为 0，第一个 output。

注意：这里是设置的服务内部的内存数据，不会更新到配置文件中，重启即失效。

返回结果
```shell
{
  "errorcode":0,
  "level":"debug",
  "message":"",
  "prelevel":"info"
}
```

## 查看框架配置文件

```shell
curl http://ip:port/cmds/config
```
返回结果

content 为 json 化的配置文件内容
```shell
{
  "content":{
  
  },
  "errorcode":0,
  "message":""
}
```

# 自定义管理命令

## 定义函数

首先自己定义一个 http 接口形式的处理函数，你可以在任何文件位置自己定义：
```go
// load 触发加载本地文件更新内存特定值
func load(w http.ResponseWriter, r *http.Request) {
  reader, err := ioutil.ReadFile("xxx.txt")
  if err != nil {
    w.Write([]byte(`{"errorcode":1000, "message":"read file fail"}`))  // 错误码，错误信息自己定义
    return
  }
  
  // 业务逻辑。..
  
  // 返回成功错误码
  w.Write([]byte(`{"errorcode":0, "message":"ok"}`))
}
```

## 注册路由

需要在 `trpc.NewServer` 加载完服务端之后再注册自定义的 admin：
```go
import (
  "trpc.group/trpc-go/trpc-go"
  "trpc.group/trpc-go/trpc-go/admin"
)
func main() {
  s := trpc.NewServer()
  adminServer, err := trpc.GetAdminService(s)
  if err != nil { .. }
  adminServer.HandleFunc("/cmds/load", load)  // 路径自己定义，一般在/cmds 下面，注意不要重复，不然会相互覆盖
}
```

## 触发命令

触发执行自定义命令
```shell
curl http://ip:port/cmds/load
```

# pprof 性能分析

pprof 是 go 语言自带的性能分析工具，默认跟 admin 服务同一个端口号，只要开启了 admin 服务，即可以使用服务的 pprof。 

配置好 admin 配置以后，有以下几种方式使用 pprof：

## 使用配置有 go 环境并且与服务网络连通的机器

```shell
go tool pprof http://{$ip}:${port}/debug/pprof/profile?seconds=20
```

## 将 pprof 文件下载到本地，本地 go 工具进行分析

```shell
curl http://${ip}:{$port}/debug/pprof/profile?seconds=20 > profile.out
go tool pprof profile.out

curl http://${ip}:{$port}/debug/pprof/trace?seconds=20 > trace.out
go tool trace trace.out
```

# 内存管理命令 debug/pprof/heap

另外，trpc-go 会自动帮你去掉 golang http 包 DefaultServeMux 上注册的 pprof 路由，规避掉 golang net/http/pprof 包的安全问题（这是 go 本身的问题）。

所以，使用 trpc-go 框架搭建的服务直接可以用 pprof 命令，但用`http.ListenAndServe("xxx", xxx)`方式起的服务会无法用 pprof 命令。

在安装了 go 命令的内网机器进行内存分析：

```shell
go tool pprof -inuse_space http://xxx:11029/debug/pprof/heap
go tool pprof -alloc_space http://xxx:11029/debug/pprof/heap
```
