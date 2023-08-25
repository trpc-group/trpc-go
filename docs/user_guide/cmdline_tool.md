[TOC]




# Introduction

The **trpc** command is a command-line tool supporting the tRPC framework. The command name is trpc. It assists
 developers in developing and testing, and improves R&D efficiency. It is mainly to help you quickly generate service
 projects. It integrates the mock test, the rpc test function of trpc-cli, and the swagger api document generation
 function.

# Feature Introduction

The main features include:
1. `trpc create` , specify the pb file, quickly generate the corresponding service template, rpc-related client stub,
 *.pb.go, etc.;
2. `trpc version` ,  display the command version number;
3. `trpc issue` , quickly create an issue or issue page;
4. Other capabilities, welcome to submit issues, and encourage everyone to contribute;
5. Support generating rpc service stub in multiple languages such as `go, python, java`;
- Template files can be added arbitrarily in the template directory, and trpc will traverse and process each template
 file;
- The generated service directory structure is consistent with the directory structure in the template;

# Installation Instructions

## Install trpc

### Install with go get

If you have already set up the go development environment, you can execute the command
 `go get -u trpc.tech/trpc-go/trpc-go-cmdline/v2/trpc` to complete the installation, and subsequent upgrades can be
 completed through `trpc upgrade`.

**Note:** Starting from version 1.17 of go, the way of installing packages through `go get` is deprecated (see
 [https://go.dev/doc/go-get-install-deprecation](https://go.dev/doc/go-get-install-deprecation)), so it is recommended
 to use `go install trpc.tech/trpc-go/trpc-go-cmdline/v2/trpc@latest`

If you see something like a certificate error or `code.oa` related error, check the following steps in order:

1. Refer to [https://goproxy.woa.com/](https://goproxy.woa.com/) to configure goproxy
2. Execute `go env` to check whether `GOPROXY` is the value set in the previous step. If not, it means that the system
 environment variable `GOPROXY` is overriding the value, and the `GOPROXY` system variable needs to be removed
3. Execute `go env`, check whether `GONOPROXY, GOPRIVATE` contains `git.code.oa.com`, if so, reset it through
 `go env -w` so that it does not contain
4. Make sure `~/.gitconfig` has the following content:
    ```
    [url "git@git.code.oa.com:"]
            insteadOf = https://git.code.oa.com/
            insteadOf = http://git.code.oa.com/
    [url "git@git.woa.com:"]
            insteadOf = https://git.woa.com/
            insteadOf = http://git.woa.com/
    ```
5. Make sure the **package path** you execute `go get` `go install` is correct and exists (no typo etc.)

### Install from Tencent Cloud software source (recommended)

(**Recommended**) Tencent Cloud software source: https://mirrors.tencent.com/repository/generic/trpc-go-cmdline/
 Download and install, click to enter the link, select the latest tag folder, and select the corresponding operating
  system after entering Click directly to download (or right-click to copy the link address to download), the
  operation of mac/linux on the command line is as follows:

```shell
# Replace v2.0.8 with the latest tag version
$ export TAG=v2.0.8
$ wget -O trpc https://mirrors.tencent.com/repository/generic/trpc-go-cmdline/$TAG/trpc_linux  # change trpc_linux to trpc_darwin for mac
# Place it in the ~/go/bin directory and add this directory to $PATH
$ mkdir -p ~/go/bin && chmod +x trpc && mv trpc ~/go/bin
$ export PATH=~/go/bin:$PATH # Or put this line in ~/.bashrc, execute source ~/.bashrc
```

### Download directly from tag attachment

If you have not set up the go development environment, you can directly download the executable program of the
 component from the tags attachment. Currently, the executable program under the three platforms of windows, macOS,
  and Linux is provided. There are two ways to download and install.
#### Download binary installation
- Select the required platform binary file download on
 [The release record of the trpc tool](https://mirrors.tencent.com/repository/generic/trpc-go-cmdline/)
- Unzip and check the version
    ```shell
    chmod +x trpc
    cp trpc /usr/local/bin/
    trpc version
    ```

#### Installation based on RPM package
- To configure the RPM source, please refer to [The instructions for using the internal RPM warehouse function of
 Tencent Software Source](https://mirrors.tencent.com/#/document/rpm)
    ```shell
    Add yum repo configuration on the machine, edit or create a new /etc/yum.repos.d/artifactory.repo, add the following
    configuration:
    [Artifactory]
    name=Artifactory
    baseurl=http://username:password@mirrors.tencent.com/repository/rpm/tencent_rpm/x86_64
    enabled=1
    ```
Among them, `username` and `password`, please click [Get Access Token](https://mirrors.tencent.com/#/private/rpm)
 under the page
- Install and use the `trpc` tool through yum
    ```shell
    # Clean up the yum cache
    yum clean metadata
    # Check available versions
    yum list trpc --showduplicates
    # Install the version built on TAG
    yum install trpc-0.x.x-1 --nogpgcheck
    # Install the latest version built on master
    yum install trpc --nogpgcheck
    ```
**Note: 1. Check [The released TAG](https://git.woa.com/trpc-go/trpc-go-cmdline/-/tags); 2. Since the RPM package is
 not signed, you need to add the --nogpgcheck tag when installing**

### How to install feflow

It has been built as a feflow plug-in. Installation through feflow does not require protoc, protoc-gen-go and other
 environment preparations. It has been integrated for distribution and has the ability to automatically update.

If you have already installed feflow, you can directly execute `fef install feflow-plugin-trpc` , if you have not
 installed it, you can use the following one-click script to install it.

#### linux/macos

Execute the following command on the command line

```bash
PRE_INSTALL_PLUGINS="trpc" /bin/bash -c "$(curl -fsSL https://mirrors.tencent.com/repository/generic/cli-market/env/unix-like/env-latest.sh)"
```

#### windows

Open **powershell** with administrator privileges and execute the following command to install trpc

```bash
$env:PRE_INSTALL_PLUGINS="trpc"; iwr https://mirrors.tencent.com/repository/generic/cli-market/env/windows/env-latest.ps1 -useb | iex
```

After running the script, open a new window and execute `trpc -h`. If you have any questions during the installation
 process, please go to  [https://docs.qq.com/doc/DWEpxdnhoVk9XdEdO](https://docs.qq.com/doc/DWEpxdnhoVk9XdEdO)

## Install third-party dependent tools

### trpc setup

trpc supports one-click installation of dependent tools: `trpc setup`, automatically completes whether dependent tools
 are installed, version checks, and installs the required version.

```bash
Run the command: trpc setup

# If the following tools are missing in the system, they will be installed automatically
zhangjie@knight ~ $ rm go/bin/protoc-gen-go
zhangjie@knight ~ $ rm go/bin/protoc-gen-secv
zhangjie@knight ~ $ rm go/bin/mockgen

zhangjie@knight ~ $ trpc setup
[setup] Initialize settings && install dependent tools
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
[setup] setup complete
```

The trpc command itself does not force developers to use proto3 or proto2, and developers need to ensure that protoc
 can support the syntax specified in the pb file.

### Upgrade protoc
It is recommended to upgrade protoc to v3.6+ or above

```bash
git clone https://github.com/protocolbuffers/protobuf
git checkout v3.6.1.3
./autogen.sh
./configure
make -j8
make install
```

After the installation is complete, please confirm whether the `protoc --version` is normal. Some students encountered
 the problem of "***the shared library libprotobuf.so.17 cannot be found***", because the shared library has not been
 refreshed.
In this case, execute `sudo ldconfig` or `export LD_LIBRARY=/usr/local/lib` to solve it.

>Before using proto3, please understand the difference with proto2

### Upgrade protoc-gen-go
It is recommended to upgrade protoc-gen-go to the latest version

```bash
git clone https://github.com/golang/protobuf golang-protobuf
cd golang-protobuf
make install
```

By default, protoc-gen-go will be installed under $GOPATH/bin, remember to add this path to $PATH.

>The old version of protoc-gen-go (release time <2018.3) does not support the source_relative option, and there may
 be problems with multi-level nesting of paths under the generated stub

### Install mockgen

>go install github.com/golang/mock/mockgen

### 3.2.4. Security verification protoc-gen-secv
If you have related features that need to be validated using the message field, you need to `import "validate.proto"`
 and add fieldoption descriptions to the message field.
This feature depends on the plugin [protoc-gen-secv](https://git.woa.com/devsec/protoc-gen-secv), which you need to
 install yourself.

## Upgrade trpc

The following methods can be used to upgrade trpc:

- trpc upgrade, all versions after v0.3.19 are supported;
- go get -u trpc.tech/trpc-go/trpc-go-cmdline/v2/trpc (or go install)
- Download from Tencent Cloud software source: https://mirrors.tencent.com/repository/generic/trpc-go-cmdline/
- Find the latest tag from the latest tag list, download the attachment trpc_${os} corresponding to the operating
 system version, and overwrite the executable program locally
- If it is installed by feflow, please refer to the instructions of feflow upgrade tool

## Uninstall trpc

- You can directly delete the corresponding executable file and template file directory (/etc/trpc, ~/.trpc) if it is
 installed through `go get` and tag attachment download
- If it is installed through feflow, please refer to the method of feflow uninstall tool to uninstall it.

# Create a project

You can use `trpc help <create>` to view specific help options.

1. Write a pb file, such as helloworld.proto
2. Execute the command `trpc create -p helloworld.proto` to support multiple languages

`trpc <create>`, here are some operations when creating a project, which may help to improve the experience of
 different developers:
- Specify language:
 - `trpc create --lang=xxx ...` The language can be go, python and other languages. The following usage uses the go
  language as an example. For other languages, you can consult the owner of this article
- Specify the output directory:
    - If `go mod init` has been executed, the project will be generated in the path where go.mod is located by default;
    - Conversely, if the `-o` output path is specified, the option value shall prevail;
    - In other cases, it is consistent with the basename of the pb file;
- Initialize go.mod:
    - You can manually initialize go.mod by `go mod init`, and then execute `trpc create` to complete code generation;
    - You can also combine the above two steps into one step, `trpc create ... --mod $module`, just specify the `-mod`
     option;
- Specify to use the http protocol:
    - The service project generated by the framework uses the trpc protocol by default in the configuration file
     trpc_go.yaml;
    - If you want to use the http protocol, or other protocols, you can:
        - Modify the configuration file manually;
        - You can also modify the team's code template;
- pb has an import scenario:
    - Please add options `trpc create --protodir=$dir1 --protodir=$dir2 ...` specify the search path for dependent pb
     files.
    - The tool needs to locate the path corresponding to each dependent pb file so that it can be passed to `protoc`
     for processing;
    - If you encounter problems with pbimport, you can also view `trpc create -v` to view protoc-related operations,
     which will help you locate the problem.
- Designated system:
    - When different departments and organizations use development services, the list of dependent third-party
     components is different, and the third-party components relied on in a certain system may not be required by
      other systems.
    - Therefore, provide the parameter `--opsys=xxx` to specify the system you depend on. For supported systems, see
     the configuration of **opsys.json** in the trpc installation directory.
    - The default value is **pcg123**, if the value does not exist in **opsys.json**, the generated code is a bare
     frame.

A demo:

```bash
curl https://git.woa.com/trpc-go/trpc-go/tree/master/examples/helloworld/helloworld.proto
trpc create -p helloworld.proto
cd Greeter
go build -v
```

# Interface aliases

After writing the pb file, if you want to define an alias for rpc requests, this situation applies to two scenarios:

- http request, hope to customize the request URI;
- The stock protocol does not support string command words, but only supports numeric command words, and a conversion
 is required here;

Let's take the following service as an example for illustration:
```
package hello;
service HelloSvr {
    rpc Hello(Req) returns(Rsp);
}
```

Hope to change rpcname from `/hello.HelloSvr/Hello` to `/cgi-bin/hello`, there are two methods:

- Method 1: Specify methodoption `option (trpc.alias) = "/cgi-bin/hello"` for rpc
    ```
    package hello;
    import "trpc.proto";
    service HelloSvr {
        rpc Hello(Req) returns(Rsp) {option (trpc.alias) = "/cgi-bin/hello"; };
    }
    ```
Then execute the command `trpc create --protofile hello.proto`.

>The above trpc.proto, and the validate.proto mentioned later are maintained under the project
 [trpc-go/trpc](https://git.woa.com/trpc-go/trpc).

- Method 2: Add comments to rpc `//@alias=/cgi-bin/hello`, both leadingComments and trailingComments are acceptable
    ```
    package hello;
    service HelloSvr {
        //@alias=/cgi-bin/hello
        rpc Hello(Req) returns(Rsp);
    }
    ```
Then execute the command `trpc create --protofile hello.proto-alias`.

# Update project

Suppose there is a proto file with the following dependencies, and we use a.proto to generate the service project.

```bash
a.proto
   |- b.proto
         |- c.proto
```

If a.proto changes:
- If it involves rpc, trpc create ... -rpconly; just modify and execute it in the stub directory, and it will replace
 the stub,*.pb.go in the local directory
- If the message has been modified, it can be directly processed by protoc instead of executing trpc

If b.proto/c.proto is modified:
- If rpc is involved, the processing method is the same as above
- If it is only message modification, the processing method is the same as above

# Custom service templates

The trpc code generation uses go template. If you want to customize the template, please understand how to use go
 template. You can refer to the [go template document](https://golang.org/pkg/text/template/).

Taking the go language as an example, the service template file is located in the project install/asset_go directory,
 and the development machine with trpc installed is located in the ~/.trpc directory, and you can customize the
  template according to your needs.

# Swagger API Documentation

The following is a pb example, described by option, the trpc tool extracts information through the option here to build
 the swagger api document:

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
            title : "Hello World"
            method: "get"
            description:
                "Input parameters: msg\n"
                "Function: used to demonstrate helloword\n"
        };
    };
}
```

If the above file is named helloworld.proto, execute `trpc create --protofile helloworld.proto --swagger` to generate a
 swagger api document while creating the project.

A json file will be generated: apidocs.swagger.json

Imported into swagger editor, the display is as follows:
![swagger1](/.resources/user_guide/cmdline_tool/apidocs_swagger_json.1.png)
![swagger2](/.resources/user_guide/cmdline_tool/apidocs_swagger_json.2.png)
# Encounter problems

There may be bugs in the trpc tool itself. On the one hand, it is the quality of the code itself. On the other hand,
 there are different habits of different developers. The test cases do not cover all possible situations.

When you encounter a problem, please don't worry, please try to feedback the problem to us. In order to help us locate
 the problem and restore the problem scenario better and faster, please:
- Try to describe the trpc tool version clearly by raising an issue, which can be obtained through `trpc version`
- Execute `trpc issue` to quickly open the problem feedback page of the project;
- And paste `trpc create -v` (add option -v to output detailed log information) to issue;
  According to the log information here, it will help developers locate the problem, thank you for your cooperation;

Please raise an issue to provide feedback and follow up on issues.
