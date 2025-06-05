# tRPC-Go framework

[![BK Pipelines Status](https://api.bkdevops.qq.com/process/api/external/pipelines/projects/pcgtrpcproject/p-20167ab337e04866b254949853c75b60/badge?X-DEVOPS-PROJECT-ID=pcgtrpcproject)](https://devops.woa.com/console/pipeline/pcgtrpcproject/p-20167ab337e04866b254949853c75b60/detail/b-f047bb8a601645af8b8b415c5ced86bc)
[![TCoverage](https://tcoverage.woa.com/openapi/v1/single/badge?token=dc1dd705-d8a9-466b-8f00-c39fa35d2190&repository=trpc-go/trpc-go)](https://tcoverage.woa.com/projects/detail/coverage-trend?repository=trpc-go%2Ftrpc-go&projectID=13acd04f-ff0f-4853-9a26-b28b2c31)
[![Go Reference](https://img.shields.io/badge/API_Docs-Go_Doc-green)](https://godoc.woa.com/git.woa.com/trpc-go/trpc-go)
[![iwiki](https://img.shields.io/badge/Wiki-iwiki-green)](https://iwiki.woa.com/p/89292279)
tRPC-Go 框架是公司统一微服务框架的 golang 版本，主要是以高性能，可插拔，易测试为出发点而设计的 rpc 框架。

## 文档地址：[iwiki](https://iwiki.woa.com/p/89292279)

## 需求管理：[tapd](https://tapd.woa.com/trpc_go/prong/stories/stories_list)

## TRY IT

## 整体架构

![架构图](https://git.woa.com/trpc-go/trpc-go/uploads/76DF446E40304476B8E12903E78B5EC4/2FE60489777F72A5901D36F114CFF331.png)

- 一个 server 进程内支持启动多个 service 服务，监听多个地址。
- 所有部件全都可插拔，内置 transport 等基本功能默认实现，可替换，其他组件需由第三方业务自己实现并注册到框架中。
- 所有接口全都可 mock，使用 gomock&mockgen 生成 mock 代码，方便测试。
- 支持任意的第三方业务协议，只需实现业务协议打解包接口即可。默认支持 trpc 和 http 协议，随时切换，无差别开发 cgi 与后台 server。
- 提供生成代码模板的 trpc 命令行工具。

## 插件管理

- 框架插件化管理设计只提供标准接口及接口注册能力。
- 外部组件由第三方业务作为桥梁把系统组件按框架接口包装起来，并注册到框架中。
- 业务使用时，只需要 import 包装桥梁路径。
- 具体插件原理可参考[plugin](plugin) 。

## 生成工具

- 安装

```bash
# 初次安装，请确保环境变量PATH已配置$GOBIN或者$GOPATH/bin
go get -u trpc.tech/trpc-go/trpc-go-cmdline/v2/trpc

# 配置依赖工具，如protoc、protoc-gen-go、mockgen等等
trpc setup

# 后续更新、回退版本
trpc version                            # 检查版本
trpc upgrade -l                         # 检查版本更新
trpc upgrade [--version <version>]      # 更新到指定版本
```

- 使用

```bash
trpc help create
```

```bash
指定pb文件快速创建工程或rpcstub，

'trpc create' 有两种模式:
- 生成一个完整的服务工程
- 生成被调服务的rpcstub，需指定'-rpconly'选项.

Usage:
  trpc create [flags]

Flags:
      --alias                  enable alias mode of rpc name
      --assetdir string        path of project template
  -f, --force                  enable overwritten existed code forcibly
  -h, --help                   help for create
      --lang string            programming language, including go, java, python (default "go")
  -m, --mod string             go module, default: ${pb.package}
  -o, --output string          output directory
      --protocol string        protocol to use, trpc, http, etc (default "trpc")
      --protodir stringArray   include path of the target protofile (default [.])
  -p, --protofile string       protofile used as IDL of target service
      --rpconly                generate rpc stub only
      --swagger                enable swagger to gen swagger api document.
  -v, --verbose                show verbose logging info
```

## 服务协议

- trpc 框架支持任意的第三方协议，同时默认支持了 trpc 和 http 协议
- 只需在配置文件里面指定 protocol 字段等于 http 即可启动一个 cgi 服务
- 使用同样的服务描述协议，完全一模一样的代码，可以随时切换 trpc 和 http，达到真正意义上无差别开发 cgi 和后台服务的效果
- 请求数据使用 http post 方法携带，并解析到 method 里面的 request 结构体，通过 http header content-type(application/json or application/pb)指定使用pb还是json
- 第三方自定义业务协议可以参考[codec](codec)

## 相关文档

- [框架设计文档](https://iwiki.woa.com/p/89292279)
- [trpc 工具详细说明](https://git.woa.com/trpc-go/trpc-go-cmdline)
- [helloworld 开发指南](examples/helloworld)
- [第三方插件 cl5 实现 demo](https://git.woa.com/trpc-go/trpc-selector-cl5)
- [第三方协议实现 demo](https://git.woa.com/trpc-go/trpc-codec)

## 如何贡献

tRPC-Go 项目组有专门的[tapd 需求管理](https://tapd.woa.com/trpc_go/prong/stories/stories_list)，里面包括了各个具体功能点以及负责人和排期时间，
有兴趣的同学可以先看一下[贡献指南](https://iwiki.woa.com/p/1941990862)和[贡献规范](https://iwiki.woa.com/p/655869831)，再看看 tapd 里面 <font color=#DC143C>需求状态为规划中</font> 的功能，自己认领任务，一起为 tRPC-Go 做贡献。
认领时将状态流转为：需求已确认
开始投入将状态流转为：开发中
开发完成将状态流转为：已发布
开发中 和 已发布 之间时间不要超过两周。需求比较大的单可以拆分成多个子需求。

## 联系人

有问题可以优先提 issue 和[码客](https://mk.woa.com/coterie/420)，紧急问题或者讨论联系：jessemjchen;wineguo;leoxhyang;amdahliu