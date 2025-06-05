# 前言

tRPC-Go 项目组提供了以下 4 种开发环境的搭建方式，分别对应不同的需求：

- 在 IT 云研发上一键快速创建环境
- 在 Linux 上开发
- 在 MacOS 上开发
- 在 Windows 上开发

对于其他平台的开发习惯，自己选择相对应的平台阅读即可。

# 1. 使用 IT 云研发一键快速创建 tRPC 环境

[无需配置，极速体验，点此即可启动！一键快速创建 tRPC-Go 开发环境](https://devc.woa.com/open/env?templateUid=template-25lqhfagfxe6&bs=devc)。更多用法请参考 [云研发](https://iwiki.woa.com/space/AnyDev)。

# 2. Linux

## 2.1 安装 Go 语言

请参考 [Go 官网](https://go.dev/doc/install) 来安装最新版本 Golang（请选择 Linux 的 tab），建议使用的版本在最新的三个稳定的 Major Release 内（比如当前最新为 `1.22`，那么 `1.22`, `1.21` 和 `1.20` 大概都是 OK 的），新版本 Go 本身会修复很多问题。

如果官网上的安装方式不够清晰，请参考如下步骤：

```shell
# 下载 Go 源码包，请保证下面的版本号是较新的
wget https://go.dev/dl/go1.22.0.linux-amd64.tar.gz
# 移除旧的已有安装，并解压出新的安装（需要 root 权限）
rm -rf /usr/local/go && tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz
```

然后需要设置环境变量。执行 `vim ~/.bashrc` 命令，把以下部分添加到 `~/.bashrc` 中：

```shell
# 这里为什么这么配置，可参考 https://learnku.com/go/t/39086#0b3da8
export GO111MODULE=on

# 把 go 命令以及 GOPATH 添加到 PATH 环境变量
test -d ~/go || mkdir ~/go
export GOPATH=~/go
export PATH=/usr/local/go/bin:$GOPATH/bin:$PATH
```

最后记得要执行 `source ~/.bashrc` 命令。

## 2.2 配置代理

首先请点击以下链接，进行 go proxy 和 go sumdb 的设置（具体操作是把链接中的小眼睛点开，然后复制到 `~/.bashrc` 中，完成后记得执行 `source ~/.bashrc` 命令）:

[Goproxy for Tencent](https://goproxy.woa.com/)

接下来请执行 `go env` 命令查看输出结果，重点看以下 key 的值：

- `GOPROXY`: 要保证这个值是 [Goproxy for Tencent](https://goproxy.woa.com/) 网站上设置的值
- `GOSUMDB`: 要保证这个值是 [Goproxy for Tencent](https://goproxy.woa.com/) 网站上设置的值
- `GONOPROXY`: 要保证这个值里面不含 `git.code.oa.com`
- `GOPRIVATE`: 要保证这个值里面不含 `git.code.oa.com`
- `GONOSUMDB`: 要保证这个值里面不含 `git.code.oa.com`

假如以上有部分不相符，那么说明存在对应的系统环境变量覆盖了 `go env` 本身设置的值，可以考虑在 `~/.bashrc` 中这样写：

```shell
export GOPROXY=""
export GOSUMDB=""
export GONOPROXY=""
export GOPRIVATE=""
export GONOSUMDB=""

# 以下三行换成你自己访问 https://goproxy.woa.com/ 点开小眼睛后看到的三行
go env -w GOPROXY="https://goproxy.woa.com,direct" 
go env -w GOPRIVATE=""
go env -w GOSUMDB="sum.woa.com+643d7a06+Ac5f5VOC4N8NUXdmhbm8pZSXIWfhek5JSmWdWrq7pLX4"

go env -w GONOSUMDB=""
```

执行 `source ~/.bashrc` 后再次执行 `go env` 命令，保证显示的值符合之前提到的要求。

### 一些注意事项

- 为了避免后续再出现 `git.code.oa.com` 相关的错误，可以再在 `~/.gitconfig` 中添加如下内容：

  ```raw
  [url "git@git.code.oa.com:"]
          insteadOf = https://git.code.oa.com/
          insteadOf = http://git.code.oa.com/
  [url "git@git.woa.com:"]
          insteadOf = https://git.woa.com/
          insteadOf = http://git.woa.com/
  ```

- 要注意不同终端环境的区别，比如你在你开的一个 shell 上打开之后执行 `go env` 显示的是符合标准的，但是在 IDE（比如 VS Code 和 Goland 等）上面点一下相关操作，结果还是报 `git.code.oa.com` 相关的错误，这个时候要仔细研究下 VS Code 和 Goland 它们本身环境变量的机制，它们使用的是什么样的 shell 等等，比如对于 Goland 来说，需要把环境变量设置到下述地方：

  ![setting](../../.resources/user_guide/environment_setup/setting.png)

  ![go_module](../../.resources/user_guide/environment_setup/go_module.png)

- 可以再额外尝试下这个 [代码笔记](https://mk.woa.com/note/6760) 中方法一给出的修改 host 法。

## 2.3 配置工蜂

### 2.3.1 支持 go get 工蜂代码

按照 [Goproxy for Tencent](https://goproxy.woa.com/) 配置后，本小节自动支持。

### 2.3.2 配置 ssh 访问工蜂

生成 ssh 公钥：

```shell
# 执行后连续按三次回车
ssh-keygen -t rsa -C "your-rtx-name@tencent.com"
```

在 `~/.ssh` 路径下创建 `config` 文件并写入如下内容，`IdentityFile` 视具体路径情况而定（需要 `root` 权限）：

```shell
Host git.woa.com 
HostName git.woa.com
User git
Port 22
IdentityFile ~/.ssh/id_rsa
```

修改 config 文件权限：

```shell
chmod 600 ~/.ssh/config
```

工蜂平台配置 ssh 公钥：

```shell
打开 https://git.woa.com/profile/keys => 点击 Add SSH Key 按钮
设置 Key => 粘贴 ~/.ssh/id_rsa.pub 内容
设置 Title => 自行命名
```

配置使用 ssh 方式访问工蜂平台：

```shell
git config --global --add url."git@git.code.oa.com:".insteadOf https://git.code.oa.com/
# 现在公司 oa 域名已全面切换到 woa 域名
git config --global --add url."git@git.woa.com:".insteadOf https://git.woa.com/
```

有些场景下 go get 不到代码但也没有任何提示，可以执行以下命令：

```shell
git config --global http.sslverify false
```

## 2.4 安装依赖

### 2.4.1 protoc v3.6.0+

请根据自己使用的操作系统选择对应的包管理工具安装：

- Linux 下用 yum 或 apt 等（如执行 `yum install protobuf-compiler` 或 `apt install -y protobuf-compiler` 命令）安装。
- MacOS 通过 brew 安装。
- Windows 通过下载可执行程序或者其他安装程序来安装。

对于某些缺乏包管理工具的平台，或者软件源提供的版本不满足最低要求的，可以参考下面的内容从源码安装；

> PS：执行下边的操作前，先确保安装了相应的工具链，如 make、autoconf、aclocal 等。

```bash
git clone https://github.com/protocolbuffers/protobuf

# v3.6.0+以上版本支持 map 解析，syntax=2、3 消息序列化后是二进制兼容的，用 root 执行以下命令
cd protobuf
git checkout v3.6.1.3
./autogen.sh
./configure
make -j8
make install
```

> PS：不要只从其他机器拷贝 protoc 命令，完整的 protobuf 开发包除了 protoc 编译器还包含一些 proto 文件。

### 2.4.2 protoc-gen-go

```bash
go get -u github.com/golang/protobuf/protoc-gen-go
```

新版本 Go 已经废弃 go get 安装，使用 install 安装：

```bash
go install github.com/golang/protobuf/protoc-gen-go@latest
```

### 2.4.3 git

低版本 git 可能容易碰上莫名其妙的 go mod 拉取失败问题，可升级到 `git 2.16.5` 后用 `root` 权限执行以下命令：

```bash
yum install curl-devel expat-devel   # 依赖
yum remove git  # 删除老版本

wget https://www.kernel.org/pub/software/scm/git/git-2.16.5.tar.gz
tar xzf git-2.16.5.tar.gz
cd git-2.16.5
make prefix=/usr/local/git all
make prefix=/usr/local/git install
```

然后在 .bashrc 中把 `/usr/local/git/bin/` 加入到 `PATH` 变量：

```shell
export PATH=$PATH:/usr/local/go/bin:/usr/local/git/bin
```

## 2.5 安装常用工具

### 2.5.1 trpc

> PS：如果使用 [rick 平台](http://trpc.rick.oa.com/rick) 进行 pb 管理和代码生成，则 trpc 工具可以暂不用安装。

低版本 Golang 环境（< 1.17）执行如下命令：

```bash
go get -u trpc.tech/trpc-go/trpc-go-cmdline/v2/trpc
```

从 go 1.17 版本开始 [通过 go get 来安装包的方式被废弃](https://go.dev/doc/go-get-install-deprecation)，使用 install 来安装：

```bash
go install trpc.tech/trpc-go/trpc-go-cmdline/v2/trpc@latest
```

> PS：此处 trpc-go-cmdline 工具本身是 v2 的，但是既可以生成 code.oa v1 的代码，也可以生成 trpc.tech v2 的代码。工具的 v2 和项目是否使用 trpc-go v2 无关。此外，[目前不推荐项目使用 v2](http://mk.woa.com/q/291729/answer/116248)。**工具的 v2 和项目是否使用 trpc-go v2 无关！这里 `go install trpc.tech/trpc-go/trpc-go-cmdline/v2/trpc@latest` 不影响你项目使用 trpc-go 的 v1！**

安装之后确保你的安装目录是加到你的 `PATH` 变量中的，否则 `trpc` 命令会找不到（或者找到的是某个其他路径下的旧版本），一般来说是安装到了你的 `$GOPATH/bin` 目录下，你可以执行如下命令：

```bash
echo $GOPATH # 如果为空的话, 可以执行 go env 来确定 GOPATH
ls $GOPATH/bin # 查看 $GOPATH/bin 目录下是否有刚安装的 trpc
export PATH=$GOPATH/bin:$PATH # 在 $PATH 环境变量中添加安装路径, 可以把这一行放到你的 ~/.bashrc 中然后 source ~/.bashrc
```

如果出现类似于证书错误或 `code.oa` 相关错误的字眼，请依次检查以下步骤：

1. 参考 [Goproxy for Tencent](https://goproxy.woa.com/) 配置 goproxy。
2. 执行 `go env` 检查 `GOPROXY`, `GOPRIVATE`, `GOSUMDB` 是否为上一步的设置值。如果不是，则说明有系统环境变量 `GOPROXY`, `GOPRIVATE`, `GOSUMDB` 在覆盖该值，需要移除 `GOPROXY`, `GOPRIVATE`, `GOSUMDB` 系统变量。
3. 执行 `go env` 检查 `GONOPROXY`, `GOPRIVATE` 中是否包含 `git.code.oa.com`。如果包含，则通过 `go env -w` 进行重新设置使其不包含。
4. 保证 `~/.gitconfig` 中有如下内容：

```raw
[url "git@git.code.oa.com:"]
        insteadOf = https://git.code.oa.com/
        insteadOf = http://git.code.oa.com/
[url "git@git.woa.com:"]
        insteadOf = https://git.woa.com/
        insteadOf = http://git.woa.com/
```

5. 确定你执行 `go get` 和 `go install` 的 package path 是正确并存在的（没有 typo 等）。

更多安装方式可以参考 [trpc-go-cmdline 其他安装方式](https://git.woa.com/trpc-go/trpc-go-cmdline#13-%E5%85%B6%E4%BB%96%E5%AE%89%E8%A3%85%E6%96%B9%E5%BC%8F)。

后续版本升级请执行 `trpc upgrade` 命令（遇到错误 `parse domainAllowed git.code.oa.com err` 时，请参考 [这里](https://mk.woa.com/note/6760) 来解决）。

### 2.5.2 trpc-cli

trpc-cli 工具是 trpc 接口测试脚手架，具有简单发包、生成 JSON 测试用例、生成接测代码、执行接口测试等功能。使用方法请参考 [快速上手](https://iwiki.woa.com/pages/viewpage.action?pageId=194215409)。

可以下载直接使用：

```shell
# Linux 版本
wget https://mirrors.tencent.com/repository/generic/trpc-go/trpc-cli/trpc-cli.zip
unzip trpc-cli.zip
# 查看版本
./trpc-cli -version

# Mac 版本
wget https://mirrors.tencent.com/repository/generic/trpc-go/trpc-cli/trpc-cli_mac.zip

# Windows 版本
wget https://mirrors.tencent.com/repository/generic/trpc-go/trpc-cli/trpc-cli_win.zip
```

或者使用 go get 安装：

```bash
go get git.code.oa.com/trpc-go/trpc-cli/v2  # 不推荐，go get 只是拉代码后 go build，但实际上是需要 make build 的，go get 包会缺少正确的版本信息
```

trpc-ui 是与 trpc-cli 配套的本地图形化接口调试工具，在个人电脑/DevCloud 开发机本地启动 Web 页面，即可对本地或远程服务进行接口测试。使用方法请参考 [使用指南](https://iwiki.woa.com/p/377047500#trpc-ui-使用指南)。

下载直接使用：

```shell
# Linux 版本
wget https://mirrors.tencent.com/repository/generic/trpc-go/trpc-ui/trpc-ui.zip
unzip trpc-ui.zip
./trpc-ui version

# Mac 版本
wget https://mirrors.tencent.com/repository/generic/trpc-go/trpc-ui/trpc-ui_mac.zip

# Windows 版本
wget https://mirrors.tencent.com/repository/generic/trpc-go/trpc-ui/trpc-ui_win.zip
```

### 2.5.3 mockgen

```bash
go get github.com/golang/mock/mockgen
```

### 2.5.4 mockserver

同源测试本地 mockserver 工具主要用于开发过程中的后端依赖 mock，具体使用细节可以看 [这里](https://iwiki.oa.tencent.com/pages/viewpage.action?pageId=195763097)。

```bash
go get git.code.oa.com/NGTest/ngtest-mock
```

### 2.5.5 dtools

dtools 工具主要用于开发过程中，直接 push 编译二进制到测试环境的 docker 里面，自动重启，方便自测。dtools 工具帮助文档见 [这里](https://iwiki.woa.com/space/dtools)。
安装：

```bash
wget -N http://mirrors.tencent.com/repository/generic/dtools/linux/release/dtools
```

使用：

```bash
dtools bpatch -env ${env} -app ${app} -server ${server} -user ${username} -lang go
```

> PS：开发网和 idc 网络是不通的，真正开发服务时，开发机只能用来写代码，不要启动调试服务，自测时使用 dtools 更新服务。

# 3. MacOS

## 3.1 安装 Go 语言

```shell
brew install go
```

## 3.2 配置工蜂

### 3.2.1 配置代理

请参考 Linux 中的配置代理一节。

### 3.2.2 配置工蜂

请参考 Linux 中的配置工蜂一节。

## 3.3 安装依赖

首先可以执行以下命令来安装 protoc 等工具的依赖：

```bash
brew install autoconf automake libtool curl make
```

安装 protoc 工具：

```bash
brew install protobuf
```

安装 protoc-gen-go 工具：

```bash
brew install protoc-gen-go
```

> PS：Mac M1 可以通过配置 Rosetta 环境来自动使用 x86_64 的二进制，相关问题见 [这里](http://mk.woa.com/q/289712/answer/112187)。

# 4. Windows

## 4.1 安装 Go 语言

下载好 [Golang](https://golang.org/dl/) 最新版 msi 文件，直接双击安装即可。然后请参照前面 Linux 的环境配置部分，先设置 go proxy，涉及到环境变量的部分替换为 Windows 的环境变量设置方法即可，`.gitconfig` 文件在 Windows 下的路径可以自行搜索。

## 4.2 配置 IDE

### 4.2.1 配置 Goland

Goland 需要支持 go mod 模式，在 `Preferences --> Go --> Go Modules` 选项中需要勾选 `enable go modules` 选项。

### 4.2.2 配置 VS Code

VS Code 的配置比较简单，去官网下载 VS Code 并安装后，VS Code 会自动在右下角提醒你安装相应插件，选择 Install all 并且 Reload 即可。

# 5. FAQ

## 5.1 Golang 环境相关问题

注：tRPC-Go 如果在 Go 1.13 环境下碰到了不在下面这些问题中的问题，可以先参考 [Go 1.13 Release Notes](https://golang.org/doc/go1.13)。

### Q1 - Go 1.13 环境 GOSUMDB 代理问题：410 Gone

![proxy-410-Gone-1](../../.resources/user_guide/environment_setup/proxy-410-Gone-1.png)

go env 输出 go 环境配置

![proxy-410-Gone-2](../../.resources/user_guide/environment_setup/proxy-410-Gone-2.png)

配置 export GOSUMDB=off

![proxy-410-Gone-3](../../.resources/user_guide/environment_setup/proxy-410-Gone-3.png)

### Q2 - 由于更换已用主机的 IP 导致 ssh 报错不识别原有的 host

```sh
uild command-line-arguments: cannot load git.code.oa.com/trpc-go/go_reuseport: git.code.oa.com/trpc-go/go_reuseport@v1.4.1-0.20190918100016-ae3a98fc71ee: invalid version: git fetch -f https://git.code.oa.com/trpc-go/go_reuseport refs/heads/*:refs/heads/* refs/tags/*:refs/tags/* in /Users/delvin/go/pkg/mod/cache/vcs/8ec554d03b667ca5a82ea00bac2b45b8efaaeaaac21fe8c672efcd03bd2db33b: exit status 128:
 @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
 @       WARNING: POSSIBLE DNS SPOOFING DETECTED!          @
 @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
 The RSA host key for git.code.oa.com has changed,
 and the key for the corresponding IP address 10.14.40.17
 is unknown. This could either mean that
 DNS SPOOFING is happening or the IP address for the host
 and its host key have changed at the same time.
 @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
 @    WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED!     @
 @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
 IT IS POSSIBLE THAT SOMEONE IS DOING SOMETHING NASTY!
 Someone could be eavesdropping on you right now (man-in-the-middle attack)!
 It is also possible that a host key has just been changed.
 The fingerprint for the RSA key sent by the remote host is
 SHA256:O/rHOxiTfD6BGBM8iwioUtqx8qHDxxd3uYn1hee4/Rc.
 Please contact your system administrator.
 Add correct host key in /Users/delvin/.ssh/known_hosts to get rid of this message.
 Offending RSA key in /Users/delvin/.ssh/known_hosts:1
 RSA host key for git.code.oa.com has changed and you have requested strict checking.
 Host key verification failed.
 fatal: Could not read from remote repository.

 Please make sure you have the correct access rights
 and the repository exists.
```

需要在连接的目标主机上的~/.ssh/known_hosts 文件，去除过时的认证

```sh
rm ~/.ssh/known_hosts
```

### Q3 - 在 Go 1.13 版本以下出现 assignment mismatch 错误

```sh
# git.woa.com/trpc-go/trpc-go/server
../trpc-go/trpc-go/server/service_timer.go:84:4: assignment mismatch: 1 variable but c.AddFunc returns 2 values
```

在 tRPC-Go 的 go.mod 文件中已经指定了 `github.com/robfig/cron v1.2.0` 但是没有生效，拉的最新 tag，检查当前是否是 go mod 模式，配置

```shell
export GO111MODULE=on
```

### Q4 - go get 报错：unknown revision

![go-get-unknown-revision](../../.resources/user_guide/environment_setup/go-get-unknown-revision.png)

有可能是环境问题，工蜂无法访问导致。请参考本文档正确配置工蜂。

也有可能是缓存问题导致的，删除 `GOPATH/pkg/mod/cache` 目录，重新 go build 即可。

### Q5 - fatal: git fetch-pack: expected shallow list

![git fetch-pack](../../.resources/user_guide/environment_setup/git-fetch-pack.png)

一般是 Git 版本过低导致，升级 Git 到 2.16 以上。建议直接使用 tRPC-Go 提供的 trpc-go-dev 开发镜像，不用自己折腾环境问题。

### Q6 - Goland import 飘红问题

Goland import 飘红，例如：

![go-import-redline-1](../../.resources/user_guide/environment_setup/go-import-redline-1.png)

Goland import 飘红，一般是下面三个问题导致：
（1）go modules 没生效
（2）代理没配置对
（3）goland 没有读取到系统的环境变量

第一步，执行 `echo $GO111MODULE` 检查 go modules 是否生效，假如是空或者是 off，需要改为 auto 或者 on，同时检查 Goland 是否开启了 go modules。

![go-import-redline-2](../../.resources/user_guide/environment_setup/go-import-redline-2.png)

第二步，检查代理是否配置正确：检查 HTTP PROXY 设置，办公网设置为 127.0.0.1:12639，如下。

![go-import-redline-3](../../.resources/user_guide/environment_setup/go-import-redline-3.png)

第三步，检查 Goland 环境变量与系统环境变量是否一致。

### Q7 - disabled by GOPRIVATE/GONOPROXY

1.13 出现。可能是 GOPROXY 设置出错。建议设置

```shell
go env -w GOPROXY=direct
go env -w GOSUMDB=off
go env -w GONOPROXY=""
go env -w GOPRIVATE=""
```

### Q8 - 终端提示 terminal prompts disabled

```shell
export GIT_TERMINAL_PROMPT=1
```

### Q9 - mod 依赖问题：invalid pseudo-version: git -c protocol.version=0

```shell
go: git.code.oa.com/trpc-go/trpc-opentracing-tjg@v0.0.0-20191206063522-55211dffce94 requires
git.code.oa.com/trpc-go/trpc-go@v0.1.0-beta.2.0.20191202064853-18cab5526064: invalid pseudo-version: git -c protocol.version=0 fetch --unshallow -f https://git.code.oa.com/trpc-go/trpc-go
        refs/heads/*:refs/heads/* refs/tags/*:refs/tags/* in/Users/sevencheng/go/pkg/mod/cache/vcs/d4cf83d942beedbbf3dcbbc639063f52f10cbf8212940ff5e7be80ff9329cab6: exit status 1:
        fatal: missing tree object '6ac9dc8576bcbaff41723554f071ea47e4f5ceeb'
        error: remote did not send all necessary objects

```

删除 tjg 的 go.mod 文件后重试。

### Q10 - mod 依赖问题：cannot find module providing package xxxx

一般是 go modules 模式下，依赖模块远程仓库不存在导致，解决方式是建立远程仓库或者使用 replace 本地仓库替代远程仓库。更多 go modules 有关可以参考：[go modules](https://go.dev/wiki/Modules)。

## 5.2 Git 相关问题

### Q1 - go get 拉取失败

拉取失败大概率是没有完全按照以上文档的步骤仔细配置，估计漏了其中某一步，比如 Git 没有升级，证书没有安装，或者拉取地址不对，总之有很多原因，先仔细确认是否完全配置正确了，再查看 [这里](http://km.oa.com/group/29073/articles/show/376902)。

### Q2 - fatal: could not read Username for '<https://git.code.oa.com>': terminal prompts disabled

原因：你的 git 使用了 HTTPS 进行 git clone（不建议），而 git.code.oa.com 的 HTTPS 需要用户名密码。

解决办法：

- 方法 1：彻底的解决办法是：不使用 HTTPS 而是使用 SSH。
- 方法 2：配置 git config。在命令行中输入 `git config --global --edit`，然后编辑 git 配置文件：

  ```raw
  [url "git@git.code.oa.com:"]
          insteadOf = https://git.code.oa.com/
          insteadOf = http://git.code.oa.com/
  [url "git@git.woa.com:"]
          insteadOf = https://git.woa.com/
          insteadOf = http://git.woa.com/
  ```

- 方法 3：[设置命令行环境变量，如 Mac 中 export GIT_TERMINAL_PROMPT=1](https://stackoverflow.com/questions/32232655/go-get-results-in-terminal-prompts-disabled-error-for-github-private-repo/45830854#45830854), 再执行 go get 并按提示输入用户名、密码。

### Q3 - 排查小技巧

整体配置完成后，出现 unknown version，首先确定该仓库是否存在（复制仓库链接到浏览器中打开，eg: `git.woa.com/ing/gomyoa`），可新建文件夹（eg: `test`），进入文件夹中使用命令 `go mod init {自定义名称 eg: test}`，创建完成后尝试拉取：`go get -v {出现 unknown version 的仓库地址 eg: git.woa.com/ing/gomyoa}`，执行后报错会出现对应错误日志。

## 5.3 证书相关问题

### Q1 - x509: certificate signed by unknown authority

```
Fetching https://git.code.oa.com/trpc-go/trpc-go?go-get=1
https fetch failed: Get https://git.code.oa.com/trpc-go/trpc-go?go-get=1: x509: certificate signed by unknown authority
```

修正 SubCA 证书：by tensorchen

在 **Mac** 上，SubCA 是随着 iOA 安装时安装的证书。

找到过期时间最久的 SubCA1 证书，先改为永不信任，退出保存。再改为始终信任，退出保存。

最后再重新 `go build -v` 即可。

在 **Linux** 上，参考：

<http://km.oa.com/group/35101/articles/show/375176> #3.配置支持 go get git.woa.com 工蜂平台代码

### Q2 - x509: certificate has expired or is not yet valid

编译 trpc-go 报错显示：

```text
unrecognized import path "honnef.co/go/tools" (https fetch: Get https://honnef.co/go/tools?go-get=1: x509: certificate has expired or is not yet valid)
```

原因是系统的证书过期了，可联系 O2000 解决，或者执行下述命令更新证书：

```shell
yum update ca-certificates -y
update-ca-trust
```

## 5.4 代码生成工具相关问题

**注意：遇到 trpc 工具问题可以先尝试卸载再重新安装，大概率可以解决问题。**

请查看 [常见的代码生成问题](https://iwiki.woa.com/p/278972980#5-常见代码生成问题)。

## 更多问题

请参考 [tRPC 技术咨询](https://iwiki.woa.com/p/491739953) 以寻求帮助
