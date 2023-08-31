# tRPC-Go framework

[English](README.md) | 中文

tRPC-Go 框架是公司统一微服务框架的 golang 版本，主要是以高性能，可插拔，易测试为出发点而设计的 rpc 框架。

## 整体架构

![架构图](.resources/overall_zh_CN.png)

- 一个 server 进程内支持启动多个 service 服务，监听多个地址。
- 所有部件全都可插拔，内置 transport 等基本功能默认实现，可替换，其他组件需由第三方业务自己实现并注册到框架中。
- 所有接口全都可 mock，使用 gomock&mockgen 生成 mock 代码，方便测试。
- 支持任意的第三方业务协议，只需实现业务协议打解包接口即可。默认支持 trpc 和 http 协议，随时切换，无差别开发 cgi 与后台 server。
- 提供生成代码模板的 trpc 命令行工具。

## 插件管理

- 框架插件化管理设计只提供标准接口及接口注册能力。
- 外部组件由第三方业务作为桥梁把系统组件按框架接口包装起来，并注册到框架中。
- 业务使用时，只需要 import 包装桥梁路径。
- 具体插件原理可参考 [plugin](plugin) 。

## 生成工具

参考 [trpc-group/trpc-go-cmdline](https://github.com/trpc-group/trpc-go-cmdline) 进行安装及使用。

## 快速上手

在安装好 [trpc-go-cmdline](https://github.com/trpc-group/trpc-go-cmdline) 工具后，阅读并执行[快速上手](https://github.com/trpc-group/trpc-go-cmdline#quick-start)，对应的实现可参考 [examples/helloworld](/examples/helloworld)

## 服务协议

- trpc 框架支持任意的第三方协议，同时默认支持了 trpc 和 http 协议
- 只需在配置文件里面指定 protocol 字段等于 http 即可启动一个 cgi 服务
- 使用同样的服务描述协议，完全一模一样的代码，可以随时切换 trpc 和 http，达到真正意义上无差别开发 cgi 和后台服务的效果
- 请求数据使用 http post 方法携带，并解析到 method 里面的 request 结构体，通过 http header content-type(application/json or application/pb) 指定使用 pb 还是 json
- 第三方自定义业务协议可以参考 [codec](codec)

## 相关文档

- [框架设计文档](https://trpc.group/trpc-go/trpc-wiki)
- [trpc 工具详细说明](https://trpc.group/trpc-go/trpc-go-cmdline)
- [helloworld 开发指南](examples/helloworld)
- [第三方协议实现 demo](https://trpc.group/trpc-go/trpc-codec)

## 如何贡献

有兴趣的同学可以先看一下 [贡献指南](CONTRIBUTING.md)，再看看 Issue 里尚未认领的问题，自己认领任务，一起为 tRPC-Go 做贡献。
