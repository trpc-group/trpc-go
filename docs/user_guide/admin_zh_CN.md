[TOC]

<!-- TOC -->

- [1 前言](#1-前言)
- [2 管理命令列表](#2-管理命令列表)
    - [2.1 查看所有管理命令](#21-查看所有管理命令)
    - [2.2 查看框架版本信息](#22-查看框架版本信息)
    - [2.3 查看框架日志级别](#23-查看框架日志级别)
    - [2.4 设置框架日志级别](#24-设置框架日志级别)
    - [2.5 查看框架配置文件](#25-查看框架配置文件)
- [3 自定义管理命令](#3-自定义管理命令)
    - [3.1 定义函数](#31-定义函数)
    - [3.2 注册路由](#32-注册路由)
    - [3.3 触发命令](#33-触发命令)
- [4 pprof 性能分析](#4-pprof性能分析)
    - [4.1 使用配置有 go 环境并且与 idc 网络连通的机器](#41-使用配置有go环境并且与idc网络连通的机器)
    - [4.2 将 pprof 文件下载到本地，本地 go 工具进行分析](#42-将pprof文件下载到本地，本地go工具进行分析)
    - [4.3 官方火焰图代理服务](#43-官方火焰图代理服务)
    - [4.4 内存管理命令debug/pprof/heap](#44-内存管理命令debugpprofheap)
    - [4.5 PCG 123 发布平台查看火焰图](#45-pcg-123发布平台查看火焰图)
- [5 FAQ](#5-faq)
- [6 OWNER](#6-owner)
  - [nickzydeng](#nickzydeng)
  - [leoxhyang（PCG 123 平台火焰图问题请联系 leoxhyang）](#leoxhyangpcg-123leoxhyang)
<!-- /TOC -->

# 前言

管理命令（admin）是服务内部的管理后台，它是框架在普通服务端口之外额外提供的 http 服务，通过这个 http 接口可以给服务发送指令，如查看日志等级，动态设置日志等级等，具体命令可以看下面的命令列表。

admin 一般用于查询服务内部状态信息，用户也可以自己定义任意的命令。 

admin 内部使用标准 restful 协议对外提供 http 服务。

框架默认不会开启 admin 能力，需要配置才会启动（生成配置时，可以默认配置好 admin，这样就能默认打开 admin 了）：

```
server:
  app: app       # 业务的应用名，注意要改成你自己的业务应用名
  server: server # 进程服务名，注意要改成你自己的服务进程名
  admin:
    ip: 127.0.0.1 # admin的ip，配置网卡nic也可以
    port: 11014   # admin的port，必须同时配置这里的ip port才会启动admin
    read_timeout: 3000 # ms. 请求被接受到请求信息被完全读取的超时时间设置，防止慢客户端
    write_timeout: 60000 # ms. 处理的超时时间
```
PCG 的 123 发布平台的配置为例如下：

```
server:                                   # 服务端配置
  app: ${app}                             # 业务的应用名
  server: ${server}                       # 进程服务名
  bin_path: /usr/local/trpc/bin/          # 二进制可执行文件和框架配置文件所在路径
  conf_path: /usr/local/trpc/conf/        # 业务配置文件所在路径
  data_path: /usr/local/trpc/data/        # 业务数据文件所在路径
  admin:
    ip: ${local_ip}      # ip  local_ip  trpc_admin_ip 都可以
    port: ${ADMIN_PORT}  #
    read_timeout: 3000   # ms. 请求被接受到请求信息被完全读取的超时时间设置，防止慢客户端
    write_timeout: 60000 # ms. 处理的超时时间
    enable_tls: false    # 是否启用tls，目前不支持
```

# 管理命令列表

框架已经内置以下命令，注意：命令中的`ip:port`是上述 admin 配置的地址，不是 service 配置的地址。

## 查看所有管理命令

```
curl http://ip:port/cmds
```
返回结果
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

## 查看框架版本信息

```
curl http://ip:port/version
```
返回结果
```
{
  "errorcode": 0,
  "message": "",
  "version": "v0.1.0-dev"
}
```

## 查看框架日志级别

```
curl -XGET http://ip:port/cmds/loglevel?logger=xxx&output=0
```
说明：logger 是为了支持多日志，不填即为框架的 default 日志，output 同一个 logger 下的不同输出，数组下标，不填即为 0，第一个 output。

返回结果
```
{
  "errorcode":0,
  "loglevel":"info",
  "message":""
}
```

## 设置框架日志级别

(value 为日志级别，值为：trace debug info warn error fatal)
```
curl http://ip:port/cmds/loglevel?logger=xxx -XPUT -d value="debug"
```
说明：logger 是为了支持多日志，不填即为框架的 default 日志，output 同一个 logger 下的不同输出，数组下标，不填即为 0，第一个 output。

注意：这里是设置的服务内部的内存数据，不会更新到配置文件中，重启即失效。

返回结果
```
{
  "errorcode":0,
  "level":"debug",
  "message":"",
  "prelevel":"info"
}
```

## 查看框架配置文件

```
curl http://ip:port/cmds/config
```
返回结果

content 为 json 化的配置文件内容
```
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
```
// load 触发加载本地文件更新内存特定值
func load(w http.ResponseWriter, r *http.Request) {
  reader, err := ioutil.ReadFile("xxx.txt")
  if err != nil {
    w.Write([]byte(`{"errorcode":1000, "message":"read file fail"}`))  // 错误码，错误信息自己定义
    return
  }
  
  // 业务逻辑...
  
  // 返回成功错误码
  w.Write([]byte(`{"errorcode":0, "message":"ok"}`))
}
```

## 注册路由

init 函数或者自己的内部函数注册 admin：
```
import (
  "git.code.oa.com/trpc-go/trpc-go/admin"
)
func init() {
  admin.HandleFunc("/cmds/load", load)  // 路径自己定义，一般在/cmds下面，注意不要重复，不然会相互覆盖
}
```

## 触发命令

触发执行自定义命令
```
curl http://ip:port/cmds/load
```

# pprof 性能分析

pprof 是 go 语言自带的性能分析工具，默认跟 admin 服务同一个端口号，只要开启了 admin 服务，即可以使用服务的 pprof。 

idc 机器配置好 admin 配置以后，有以下三种方式使用 pprof：

## 使用配置有 go 环境并且与 idc 网络连通的机器

```
go tool pprof http://{$ip}:${port}/debug/pprof/profile?seconds=20
```

## 将 pprof 文件下载到本地，本地 go 工具进行分析

```
curl http://${ip}:{$port}/debug/pprof/profile?seconds=20 > profile.out
go tool pprof profile.out

curl http://${ip}:{$port}/debug/pprof/trace?seconds=20 > trace.out
go tool trace trace.out
```

## 官方火焰图代理服务

tRPC-Go 官方已经搭建好火焰图代理服务，只需要在办公网浏览器输入以下地址即可查看自己服务的火焰图，其中 ipport 参数为你的服务的 admin 地址
```
https://trpcgo.debug.woa.com/debug/proxy/profile?ip=${ip}&port=${port}
https://trpcgo.debug.woa.com/debug/proxy/heap?ip=${ip}&port=${port}
```
另外，还搭建了 golang 官方的 go tool pprof web 服务，（owner: terrydang）有 ui 界面
```
https://qqops.woa.com/pprof/
```

## 内存管理命令debug/pprof/heap

trpc 从 0.4.0 版本开始，直到 0.5.1 版本，由于安全问题 admin 没有集成 pprof 内存分析的命令，0.5.2 版本解决了安全问题，重新集成了 pprof 内存分析的命令。

所以，如果要使用/debug/pprof/heap, 请确认 trpc-go 已更新到最新版本。

另外，trpc-go 会自动帮你去掉 golang http 包 DefaultServeMux 上注册的 pprof 路由，规避掉 golang net/http/pprof包的安全问题（这是go本身的问题）。

所以，使用 trpc-go 框架搭建的服务直接可以用 pprof 命令，但用`http.ListenAndServe("xxx", xxx)`方式起的服务会无法用 pprof 命令。

如果一定要起原生 http 服务，并且要用 pprof 分析内存，可以通过`mux := http.NewServeMux()`的方式，不要用`http.DefaultServeMux`。

在安装了 go 命令的内网机器进行内存分析：

```
go tool pprof -inuse_space http://xxx:11029/debug/pprof/heap
go tool pprof -alloc_space http://xxx:11029/debug/pprof/heap
```

## PCG 123 发布平台查看火焰图

对于 123 平台的用户，可以直接集成到页面按钮上，更方便使用，首先需要用户安装火焰图插件，请按以下步骤简单操作一下，只需安装一次，后面每个服务自己根据实际需求开启或者禁用即可。

注意：只有 tRPC-Go 有火焰图，trpc 其他语言没有！！

1、打开 123 平台，鼠标移动到右上角的个人头像，点击“插件中心”
!['install_plugin_step_1.png'](/.resources/user_guide/admin/install_plugin_step_1.png)

2、点击“安装服务插件”，会弹出选择插件的界面，普通插件搜索查询“火焰图”，点击安装
!['install_plugin_step_2.png'](/.resources/user_guide/admin/instll_plugin_step_2.png)

3、安装成功后，点击“添加”按钮，会弹出界面给用户选择需要安装插件的服务，可以选择多个服务一起安装（注意这里是区分环境的，同个服务在不同环境下需要分别选择）
!['install_plugin_step_3.png'](/.resources/user_guide/admin/instll_plugin_step_3.png)

4、添加好服务后，点击上图具体的服务名，可以查看安装后的效果，此时鼠标移动到节点列表每一行右边“更多操作”，会出现刚刚安装的“查看火焰图”操作项，然后点击按钮即可生成火焰图
!['install_plugin_step_4.png'](/.resources/user_guide/admin/instll_plugin_step_4.png)

# FAQ

<todo>

# OWNER

## nickzydeng

## leoxhyang（PCG 123 平台火焰图问题请联系 leoxhyang）