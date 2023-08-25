# 前言

**trpc** 命令 是 tRPC 框架配套的一个命令行工具，命令名就叫 trpc，辅助开发人员开发、测试，提高研发效率，主要是为了帮助大家快速生成服务工程，后期为了方便大家测试、沟通等，又陆续集成了 mock 测试、trpc-cli 的 rpc 测试功能、swagger api 文档生成功能。

# 功能介绍

主要包含如下功能：
1.  `trpc create` , 指定 pb 文件，快速生成对应的服务模板、rpc 相关 client stub、*.pb.go 等等；
2.  `trpc version` ，显示命令版本号；
3.  `trpc issue` ，快速创建一个 issue 或者 issue 页面；
4. 其他能力，欢迎提 issue，鼓励大家共建。
5. 支持`go、python、java`等多种语言生成 rpc 服务桩代码
- 模板文件，可以在模板目录下任意添加，trpc 会遍历、处理每一个模板文件；
- 生成服务目录结构与模板中目录结构保持一致；

# 安装说明

## 安装 trpc

### go get 安装

如果您已经设置好 go 开发环境，可以通过执行命令 `go get -u trpc.tech/trpc-go/trpc-go-cmdline/v2/trpc` 完成安装，后续升级可以通过 `trpc upgrade` 来完成。

**注：** 从 go 1.17 版本开始，通过 `go get` 来安装包的方式被废弃（见 [https://go.dev/doc/go-get-install-deprecation](https://go.dev/doc/go-get-install-deprecation)），所以建议使用 `go install trpc.tech/trpc-go/trpc-go-cmdline/v2/trpc@latest`

如果出现类似于证书错误或 `code.oa` 相关错误的字眼，请依次检查以下步骤：

1. 参考 [https://goproxy.woa.com/](https://goproxy.woa.com/) 配置 goproxy
2. 执行 `go env`, 检查 `GOPROXY` 是否为上一步的设置值，如果不是，则说明有系统环境变量 `GOPROXY` 在覆盖该值，需要移除 `GOPROXY` 系统变量
3. 执行 `go env`, 检查 `GONOPROXY,GOPRIVATE` 中是否包含 `git.code.oa.com`, 如果包含，则通过 `go env -w` 进行重新设置使其不包含
4. 保证 `~/.gitconfig` 中有如下内容：
```
[url "git@git.code.oa.com:"]
        insteadOf = https://git.code.oa.com/
        insteadOf = http://git.code.oa.com/
[url "git@git.woa.com:"]
        insteadOf = https://git.woa.com/
        insteadOf = http://git.woa.com/
```
5. 确定你执行 `go get` `go install` 的 package path 是正确并存在的（没有 typo 等）

### 从腾讯云软件源安装 (推荐)

- (**推荐**) 腾讯云软件源：https://mirrors.tencent.com/repository/generic/trpc-go-cmdline/ 下载安装，点击进入链接后，选择最新的 tag 文件夹，进入后选择对应的操作系统直接点击下载 (或者右键复制链接地址进行下载), mac/linux 在命令行的操作如下：

```shell
# 把 v2.0.8 换成最新的 tag 版本即可
$ export TAG=v2.0.8
$ wget -O trpc https://mirrors.tencent.com/repository/generic/trpc-go-cmdline/$TAG/trpc_linux # mac 则把 trpc_linux 改为 trpc_darwin
# 将其放置于 ~/go/bin 目录下, 并把这个目录添加到 $PATH 中
$ mkdir -p ~/go/bin && chmod +x trpc && mv trpc ~/go/bin
$ export PATH=~/go/bin:$PATH # 或者把这一行放到 ~/.bashrc 中, 执行 source ~/.bashrc
```

### 直接从 tag 附件下载

如果您没有设置好 go 开发环境，可以直接从 tags 附件中下载构件好的可执行程序，目前提供了 windows、macOS、Linux 3 个平台下的可执行程序。可通过两种方式下载安装
#### 下载二进制安装
- 在[trpc 工具的发布记录](https://mirrors.tencent.com/repository/generic/trpc-go-cmdline/)上选择需要的平台二进制文件下载
- 解压并查看版本
```shell
chmod +x trpc
cp trpc /usr/local/bin/
trpc version
```

#### 基于 RPM 包安装
- 配置 RPM 源，可参考[腾讯软件源内部 RPM 仓库功能使用指引](https://mirrors.tencent.com/#/document/rpm)
```shell
在机器上添加yum repo配置，编辑或新建/etc/yum.repos.d/artifactory.repo，添加如下配置：
[Artifactory]
name=Artifactory
baseurl=http://username:password@mirrors.tencent.com/repository/rpm/tencent_rpm/x86_64
enabled=1
```
其中，`username`和`password`请点击页面下的[获取访问令牌](https://mirrors.tencent.com/#/private/rpm)
- 通过 yum 安装使用`trpc`工具
```shell
# 清理yum缓存
yum clean metadata
# 查看可用版本
yum list trpc --showduplicates
# 安装基于TAG构建的版本
yum install trpc-0.x.x-1 --nogpgcheck
# 安装基于master构建的最新版本
yum install trpc --nogpgcheck
```
**注意：1、查看[查看已发布的 TAG](https://git.woa.com/trpc-go/trpc-go-cmdline/-/tags)；2、由于 RPM 包没有签名，所以安装时需要添加--nogpgcheck 标记**

### feflow 安装方式

已经构建为 feflow 插件，通过 feflow 安装不需要 protoc、protoc-gen-go 等环境准备，已经集成在一起进行分发，同时也具备自动更新能力。

如果已经安装过 feflow，可以直接执行 `fef install feflow-plugin-trpc` ，未安装过则可以使用下面的一键脚本进行安装。

#### linux/macos

命令行中执行下面的命令

```bash
PRE_INSTALL_PLUGINS="trpc" /bin/bash -c "$(curl -fsSL https://mirrors.tencent.com/repository/generic/cli-market/env/unix-like/env-latest.sh)"
```

#### windows

以管理员权限打开 **powershell** 并执行下面的命令即可安装 trpc

```bash
$env:PRE_INSTALL_PLUGINS="trpc"; iwr https://mirrors.tencent.com/repository/generic/cli-market/env/windows/env-latest.ps1 -useb | iex
```

跑完脚本之后打开新的窗口执行 `trpc -h` 即可。安装过程中的任何问题请前往 [https://docs.qq.com/doc/DWEpxdnhoVk9XdEdO](https://docs.qq.com/doc/DWEpxdnhoVk9XdEdO)

## 安装第三方依赖工具

### trpc setup

trpc 支持一键完成依赖工具的安装：`trpc setup`，自动完成依赖工具是否安装、版本检查、安装符合要求版本。

```bash
运行命令：trpc setup

# 假如系统中缺乏下面的工具，会自动安装
zhangjie@knight ~ $ rm go/bin/protoc-gen-go
zhangjie@knight ~ $ rm go/bin/protoc-gen-secv
zhangjie@knight ~ $ rm go/bin/mockgen 

zhangjie@knight ~ $ trpc setup
[setup] 初始化设置 && 安装依赖工具
[setup] check dependency protoc: passed
[setup] check dependency protoc-gen-go failed: not installed, try installing
[setup] check dependency protoc-gen-go failed: not installed, try installing done
[setup] check dependency protoc-gen-go: passed
[setup] check dependency protoc-gen-secv failed: not installed, try installing
[setup] check dependency protoc-gen-secv failed: not installed, try installing done
[setup] check dependency protoc-gen-secv: passed
[setup] check dependency goimports: passed
[setup] check dependency mockgen failed: not installed, try installing
[setup] check dependency mockgen failed: not installed, try installing done
[setup] check dependency mockgen: passed
[setup] 设置完成
```

trpc 命令本身并不强求开发者使用 proto3 或者 proto2，开发者需自己保证 protoc 能支持 pb 文件中指定的 syntax。

### 升级 protoc
建议将 protoc 升级到 v3.6+以上

```bash
git clone https://github.com/protocolbuffers/protobuf
git checkout v3.6.1.3
./autogen.sh
./configure
make -j8
make install
```

安装完成后请确认下 `protoc --version` 是否正常，有部分同学遇到“***共享库 libprotobuf.so.17 无法找到***”的问题，原因是因为共享库没有刷新。
这种情况下可以执行 `sudo ldconfig` 或者 `export LD_LIBRARY=/usr/local/lib` 来解决。

>使用 proto3 之前，请先了解下与 proto2 的区别

### 升级 protoc-gen-go
建议将 protoc-gen-go 升级到最新版

```bash
git clone https://github.com/golang/protobuf golang-protobuf
cd golang-protobuf
make install
```

默认会将 protoc-gen-go 安装到$GOPATH/bin 下面，记得将该路径加到$PATH。

>旧版 protoc-gen-go（发布时间<2018.3) 不支持 source_relative 选项，可能出现生成的 stub 下路径多级嵌套问题

### 安装 mockgen

>go install github.com/golang/mock/mockgen

### 3.2.4. 安全校验 protoc-gen-secv
如果您有需要使用 message 字段校验的相关特性，您需要`import "validate.proto"`并对 message 字段添加 fieldoption 说明。
该特性依赖插件 [protoc-gen-secv](https://git.woa.com/devsec/protoc-gen-secv)，您需要自行安装该插件。

## 升级 trpc

升级 trpc 可以采用以下方法：

- trpc upgrade，v0.3.19 之后版本均支持；
- go get -u trpc.tech/trpc-go/trpc-go-cmdline/v2/trpc (或 go install)
- 从腾讯云软件源下载：https://mirrors.tencent.com/repository/generic/trpc-go-cmdline/ 
- 从最新 tag 列表里找到最新的 tag，下载对应操作系统版本的附件 trpc_${os}到本地覆盖可执行程序
- 如果是 feflow 安装的，请参考 feflow 升级工具的说明

## 卸载 trpc

- 通过`go get` 、tag 附件下载方式安装的，您可以直接删除对应可执行文件、模板文件目录（/etc/trpc、~/.trpc）
- 通过 feflow 安装的，请参考 feflow 卸载工具的方式来卸载。

# 创建工程

可以通过 `trpc help <create>` 来查看具体的帮助选项。

1. 编写一个 pb 文件，如 helloworld.proto
2. 执行命令 `trpc create -p helloworld.proto`支持多种语言

`trpc <create>`, 这里介绍下创建工程时一些操作，可能会有助于改善不同开发者的使用体验：
- 指定语言：
 - `trpc create --lang=xxx ...`语言可以是 go、python 等语言，下面用法选用 go 语言为例，其他语言可以咨询本文 owner
- 指定输出目录：
    - 如果已经`go mod init`，则默认在 go.mod 所在路径生成工程；
    - 反之，如果指定了`-o`输出路径，以选项值为准；
    - 其他情况下与 pb 文件的 basename 一致；
- 初始化 go.mod:
    - 您可以先手动初始化 go.mod by `go mod init`，再执行`trpc create`完成代码生成；
    - 您也可以将上述两步操作合并成一步，`trpc create ... --mod $module`，指定`-mod`选项即可；
- 指定使用 http 协议：
    - 框架生成的服务工程，配置文件 trpc_go.yaml 中默认使用 trpc 协议；
    - 如果您希望使用 http 协议，或者其他协议，可以：
        - 手动修改该配置文件；
        - 也可以修改团队的代码模板；
- pb 存在 import 场景：
    - 请添加选项 `trpc create --protodir=$dir1 --protodir=$dir2 ...`指定依赖的 pb 文件的搜索路径
      工具需要定位清楚每一个依赖的 pb 文件对应的路径，以方便传递给`protoc`去处理；
      遇到 pbimport 的问题，您也可以查看`trpc create -v`来查看`protoc` 相关的操作，有助于您定位问题
- 指定体系：
    - 不同部门、组织使用开发服务时，依赖的第三方组件列表也不一样，某个体系下依赖的第三方组件不一定是其他体系所需要的
    - 因此，提供参数 `--opsys=xxx` 指定自己依赖的体系，可支持的体系见 trpc 安装目录下的** opsys.json **配置
    - 默认值** pcg123**，如果值不存在** opsys.json **里，生成的代码为裸框架

一个 demo：

```bash
curl https://git.woa.com/trpc-go/trpc-go/tree/master/examples/helloworld/helloworld.proto
trpc create -p helloworld.proto
cd Greeter
go build -v
```

# 接口别名

编写完了 pb 文件，想定义 rpc 请求时的别名，这种情况适用于两种场景：

- http 请求，希望自定义请求 URI；
- 存量协议，不支持字符串命令字的，只支持数值型命令字的，这里需要一个转换；

我们以如下 service 为例进行说明：
```
package hello;
service HelloSvr {
    rpc Hello(Req) returns(Rsp);
}
```

希望将 rpcname 从`/hello.HelloSvr/Hello`修改为`/cgi-bin/hello`，有两种方法：

- 方法一：为 rpc 指定 methodoption `option (trpc.alias) = "/cgi-bin/hello"`
```
package hello;
import "trpc.proto";
service HelloSvr {
    rpc Hello(Req) returns(Rsp) {option (trpc.alias) = "/cgi-bin/hello"; };
}
```
然后执行命令 `trpc create --protofile hello.proto`.

>上述 trpc.proto，以及后面提到的 validate.proto 在项目 [trpc-go/trpc](https://git.woa.com/trpc-go/trpc) 下面维护。

- 方法二：为 rpc 添加注释 `//@alias=/cgi-bin/hello`，leadingComments、trailingComments 均可
```
package hello;
service HelloSvr {
    //@alias=/cgi-bin/hello
    rpc Hello(Req) returns(Rsp);
}
```
然后执行命令 `trpc create --protofile hello.proto -alias`.

# 更新工程

假如存在如下依赖关系的 proto 文件，并且我们用 a.proto 来生成服务工程。

```bash
a.proto
   |- b.proto
         |- c.proto
```

如果 a.proto 有变化：
- 如果涉及到 rpc，trpc create ... -rpconly，在 stub 目录下修改执行即可，会替换掉本地目录下的 stub,*.pb.go
- 如果是 message 有修改，直接 protoc 处理就行了，不用执行 trpc

如果 b.proto/c.proto 有修改：
- 如果涉及到 rpc，处理方式同上
- 如果只是 message 修改，处理方式同上

# 自定义服务模板

trpc 代码生成利用了 go template，如果您想自定义模板，请先了解下 go template 的使用方式，可以参考 [go template 文档](https://golang.org/pkg/text/template/)。

以 go 语言为例，服务模板文件位于工程 install/asset_go 目录下，对于安装了 trpc 的开发机位于~/.trpc 目录下，您可以根据需要定制模板。

# Swagger API 文档

下面是一个 pb 示例，通过 option 来描述，trpc 工具通过这里的 option 提取信息用来构建 swagger api 文档：

```
syntax = "proto3";
package helloworld;

import "trpc.proto";
import "swagger.proto";

message HelloReq{
    string msg = 1;
}
message HelloRsp{
    int32 err_code = 1;
    string err_msg = 2;
}

service helloworld_svr {
    rpc Hello(HelloReq) returns(HelloRsp) {
        option(trpc.alias) = "/api/v1/helloworld";

        option(trpc.swagger) = {
            title : "你好世界"
            method: "get"
            description:
                "入参：msg\n"
                "作用：用于演示 helloword\n"
        };
    };
}
```

假如上述文件名为 helloworld.proto，则执行`trpc create --protofile helloworld.proto --swagger`即可在创建工程的同时，也生成一份 swagger 的 api 文档。

会生成一个 json 文件：apidocs.swagger.json

导入 swagger editor 中，展示如下：
![swagger1](/.resources/user_guide/cmdline_tool/apidocs_swagger_json.1.png)
![swagger2](/.resources/user_guide/cmdline_tool/apidocs_swagger_json.2.png)

# 遇到问题

trpc 工具本身可能存在 bug，一方面是代码本身质量问题，另一方面也有不同开发者不同习惯，测试用例没能覆盖全所有可能的情况。

当您遇到问题时，请不要着急，请尝试把问题反馈给我们。为了更好更快地帮助我们定位问题、还原问题场景，请：
- 尽可能提 issue，描述清楚 trpc 工具版本，可以通过`trpc version`来获取；
- 执行 `trpc issue` 快速打开工程的问题反馈页面；
- 并将 `trpc create -v` （加个选项-v 输出详细 log 信息）粘贴到 issue；
  根据这里的 log 信息，有助于帮开发者定位问题，谢谢合作；

请提 issue 来反馈、跟进问题。
