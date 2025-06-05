# tRPC-Go 数据校验

## 1 前言

输入数据校验是应用的重要组成部分，其不仅与功能逻辑高度相关，历史经验表明，约 80% 的安全漏洞也可通过数据校验规避。
tRPC-Go 框架提供了一套数据校验组件，仅需在 pb 字段后定义校验规则，框架将自动完成数据校验代码的生成与调用，整个流程与传统的编写 pb 开发 RPC 程序无异。
这样一来，在做 tRPC-Go 应用开发时，不仅可显著减少代码编写量，还能预防约 80% 的安全风险。

## 2 快速开始

### 2.1 validation v3 接入
>
> 推荐使用新版 validation v3，iWiki&QuickStart：[tRPC Validation V3 - 腾讯 iWiki (woa.com)](https://iwiki.woa.com/p/4012527158)

#### 2.1.1 在 proto 内定义校验规则

##### 2.1.1.1 编写 proto 文件，引入 validate.proto 描述文件**

如果在步骤 2.1.2 中要在**本地使用 trpc 命令行工具**，或是 protoc 插件生成桩代码则使用以下方式引入：
从[validation-proto](https://git.woa.com/sec-api/protovalidate/validation-proto)下载 proto 规则文件，并将`buf`目录下所有文件放到项目根目录

```
import "buf/validate/validate.proto";
```

如果在步骤 2.1.2 中**使用 Rick 平台**，则使用以下方式引入：

```
import "trpcsec/common/validate.proto"
```

##### 2.1.1.2 在扩展字段添加 Validation 规则（规则详细配置请参考[规则编写和差异](https://iwiki.woa.com/p/4012531738)），例如：**

```
string email = 2 [(buf.validate.field).string.email = true];
```

#### 2.1.2 使用脚手架工具生成桩代码

##### 2.1.2.1 方案一、使用[Rick 统一 proto 托管平台](http://trpc.rick.woa.com/)（推荐，免本地环境安装）

![rick](../../.resources/user_guide/data_validation/rick.png)

##### 2.1.2.2 方案二、使用 trpc 命令行工具

1. 首先，分别下载安装[trpc 命令工具](https://git.woa.com/trpc-go/trpc-go-cmdline)和 proto 规则：[validation-proto](https://git.woa.com/sec-api/protovalidate/validation-proto)
2. 执行如下命令，`--protodir`要指定[validation-proto](https://git.woa.com/sec-api/protovalidate/validation-proto)的路径，不然关联不到会报错。

```shell
trpc create -protofile=test.proto --protodir <buf_dir> -protocol=trpc
```

##### 2.1.2.3 方案三、使用 Protoc 插件

`-I`要指定[validation-proto](https://git.woa.com/sec-api/protovalidate/validation-proto)的文件路径。

```shell
protoc -I . -I <buf_dir>  --go_out=.  test.proto
```

#### 2.1.3 引入拦截器并打开框架的 yaml 配置

在 main.go 中添加如下代码引入[拦截器](https://git.woa.com/trpc-go/trpc-filter/tree/master/validation)：

```go
import (
    _ "git.code.oa.com/trpc-go/trpc-filter/validation/v3"
)
```

在`trpc_go.yaml`中打开拦截器开关，使校验生效：

```yaml
server:
  ...
  filter:
    ...
    - validation
```

### 2.2 旧版接入

#### 2.2.1 在 proto 内定义校验规则

##### 2.2.1.1 编写 proto 文件，引入 validate.proto 描述文件**

如果在步骤 2.2 中要在**本地使用 trpc 命令行工具**，使用以下方式引入：

```
import "validate.proto"
```

如果在步骤 2.2 中**使用 Rick 平台**，则使用以下方式引入：

```
import "trpc/common/validate.proto"
```

##### 2.2.1.2 在扩展字段添加 Validation 规则（校验规则详参考本文 3.1 部分），例如：**

```
string email = 2 [(validate.rules).string.email = true];
```

#### 2.2.2 使用脚手架工具生成桩代码

##### 2.2.2.1 方案一、使用[Rick 统一 proto 托管平台](http://trpc.rick.woa.com/)（推荐，免本地环境安装）

![rick](../../.resources/user_guide/data_validation/rick.png)

##### 2.2.2.2 方案二、使用 trpc 命令行工具

1. 首先，分别下载安装[trpc 命令工具](https://git.woa.com/trpc-go/trpc-go-cmdline)和[secv validation 插件](https://git.woa.com/devsec/protoc-gen-secv)
2. 执行如下命令

```shell
trpc create -protofile=test.proto -protocol=trpc
```

##### 2.2.2.3 方案三、使用 Protoc 插件

```shell
protoc -I . -I ${GOPATH}/src/git.code.oa.com/devsec/protoc-gen-secv/validate --secv_out="lang=go:./" helloworld.proto
```

#### 2.2.3 引入拦截器并打开框架的 yaml 配置

在 main.go 中添加如下代码引入[拦截器](https://git.woa.com/trpc-go/trpc-filter/tree/master/validation)：

```go
import (
    _ "git.code.oa.com/trpc-go/trpc-filter/validation"
)
```

在`trpc_go.yaml`中打开拦截器开关，使校验生效：

```yaml
server:
  ...
  filter:
    ...
    - validation
```

## 3 规则示例
>
> validation v3 规则示例查看 iWiki：[tRPC Validation V3 - 腾讯 iWiki (woa.com)](https://iwiki.woa.com/p/4012527158)

### 3.1 基础规则

#### 3.1.1 规则写法

Validation 规则紧跟在字段申明后，用`[]`包裹，结构如下：

![rule](../../.resources/user_guide/data_validation/rule.png)

说明

- ① 规则头：固定值，本插件规则，均填写`(validate.rules)`
- ② 数据类型：支持 16 种基础数据类型（[Proto3 - Scalar Value Types](https://developers.google.com/protocol-buffers/docs/proto3#scalar) ），以及 5 种高级数据类型。
- ③ 规则内容：参考`规则索引`部分

#### 3.1.2 常用规则索引

- [字符串（Strings](https://iwiki.woa.com/pages/viewpage.action?pageId=241919746)
  - [IP 或域名](https://iwiki.woa.com/pages/viewpage.action?isEmbeded=true&pageId=241919746#mdch0#mdch#IP%E6%88%96%E5%9F%9F%E5%90%8D)
  - [纯大小写字母](https://iwiki.woa.com/pages/viewpage.action?isEmbeded=true&pageId=241919746#mdch0#mdch#%E7%BA%AF%E5%A4%A7%E5%B0%8F%E5%86%99%E5%AD%97%E6%AF%8D)（例：aBc）
  - [大小写字母与数字组合](https://iwiki.woa.com/pages/viewpage.action?isEmbeded=true&pageId=241919746#mdch0#mdch#%E5%A4%A7%E5%B0%8F%E5%86%99%E5%AD%97%E6%AF%8D%E4%B8%8E%E6%95%B0%E5%AD%97%E7%BB%84%E5%90%88)（例：a1）
  - [纯小写字母](https://iwiki.woa.com/pages/viewpage.action?isEmbeded=true&pageId=241919746#mdch0#mdch#%E7%BA%AF%E5%B0%8F%E5%86%99%E5%AD%97%E6%AF%8D)（例：abc）
  - [默认安全的字符串范围](https://iwiki.woa.com/pages/viewpage.action?isEmbeded=true&pageId=241919746#mdch0#mdch#%E9%BB%98%E8%AE%A4%E5%AE%89%E5%85%A8%E7%9A%84%E5%AD%97%E7%AC%A6%E4%B8%B2%E8%8C%83%E5%9B%B4)（可预防 SQL 注入、命令注入、路径穿越等常见高风险问题）
  - [自定义正则表达式](https://iwiki.woa.com/pages/viewpage.action?isEmbeded=true&pageId=241919746#mdch0#mdch#pattern:%20%E8%87%AA%E5%AE%9A%E4%B9%89%E6%AD%A3%E5%88%99%E8%A1%A8%E8%BE%BE%E5%BC%8F)
  - [限制字符串长度范围](https://iwiki.woa.com/pages/viewpage.action?isEmbeded=true&pageId=241919746#mdch0#mdch#len/min_len/max_len:%20%E9%99%90%E5%AE%9A%E5%AD%97%E6%AE%B5%E5%80%BC%E5%8F%AF%E5%8C%85%E5%90%AB%E7%9A%84Unicode%E5%AD%97%E7%AC%A6%E4%B8%B2%E9%95%BF%E5%BA%A6)
- [数字（Numerics](https://iwiki.woa.com/pages/viewpage.action?pageId=241919768)
- [布尔值（Bools](https://iwiki.woa.com/pages/viewpage.action?pageId=241919765)
- [字节（Bytes](https://iwiki.woa.com/pages/viewpage.action?pageId=241919751)
- [枚举（Enums](https://git.woa.com/devsec/protoc-gen-secv/wikis/%E6%A0%A1%E9%AA%8C%E8%A7%84%E5%88%99/%E6%9E%9A%E4%B8%BE-Enums/)
- [嵌套（Repeated](https://iwiki.woa.com/pages/viewpage.action?pageId=241919763)
- [消息（Message](https://iwiki.woa.com/pages/viewpage.action?pageId=241919790)
- [映射（Maps](https://iwiki.woa.com/pages/viewpage.action?pageId=241919774)
- [泛型（Any](https://iwiki.woa.com/pages/viewpage.action?pageId=241919787)
- [时间戳（Timestamps](https://iwiki.woa.com/pages/viewpage.action?pageId=241919772)

### 3.2 高级用法

#### 3.2.1 同一字段添加两条规则

**Q：** 同一个 proto 字段，要添加两条及以上的校验规则
**A：** 以 string 字段为例，示例如下：

```
// 方案 1（大括号数组内的 key 不能包含.，如果要包含。请参考方案 2）
string x = 1 [(validate.rules).string={tsecstr:true,min_len:2}];

// 方案 2
repeated uint64 msg = 1 [(validate.rules).repeated.items.uint64.gt = 2, (validate.rules).repeated.unique = true];
```

#### 3.2.2 限定字段必填 (Required)

**Q：** 要限定字段必须传入一个值
**A：**
Strings&Bytes 类型，使用限定最小长度实现，如：

```
// 限定最小长度为 1，即代表字段必填
string x = 1 [(validate.rules).string.min_len = 1];
bytes x = 1 [(validate.rules).bytes.min_len = 1];
```

Repeated 类型，使用限定最小子项目个数实现，如：

```
repeated int32 x = 1 [(validate.rules).repeated.min_items = 1];
```

Numerics 类型，使用范围限定方法实现，如：

```
// 假如该字段值是恒定不为 0 的数字则使用 not_in，代表所有非 0 的 uint32 整数
uint32 x = 1 [(validate.rules).uint32  = {not_in: [0]}];

// 字段值必须大于等于 0，即代表 0 ~ 18446744073709551615，相当于要求 required
uint64 x = 1 [(validate.rules).uint64.gte = 0];
```

Timestamps 类型，直接使用提供的 required 方法校验：

```
google.protobuf.Timestamp x = 1 [(validate.rules).timestamp.required = true];
```

Any 类型，直接使用提供的 required 方法校验：

```
// 字段值必须传入
google.protobuf.Any x = 1 [(validate.rules).any.required = true];
```

Message 类型，直接使用提供的 required 方法校验：

```
Person x = 1 [(validate.rules).message.required = true];
```

Maps 类型，对 Key 和 Value 使用最小长度实现，参见上述 Strings、Bytes、Numerics 等类型的建议方案

#### 3.2.3 repeated 字段添加 unique，以及对 items 的字段值限制

**Q：**repeated 字段，需要限定传入的字段值的唯一性，且需要对子项目递归进行基础校验。
**A：**组合使用 unique、items 规则命令。示例如下：

```
repeated uint64 msg = 1 [(validate.rules).repeated.items.uint64.gt = 2, (validate.rules).repeated.unique = true];
```

自动生成的校验桩代码：

```go
_HelloRequest_Msg_Unique := make(map[uint64]struct{}, len(m.GetMsg()))

for idx, item := range m.GetMsg() {
    _, _ = idx, item

    if _, exists := _HelloRequest_Msg_Unique[item]; exists {
        return HelloRequestValidationError{
            field:  fmt.Sprintf("Msg[%v]", idx),
            reason: "repeated value must contain unique items",
        }
    } else {
        _HelloRequest_Msg_Unique[item] = struct{}{}
    }

    if item <= 2 {
        return HelloRequestValidationError{
            field:  fmt.Sprintf("Msg[%v]", idx),
            reason: "value must be greater than 2",
        }
    }
}
```

## 4 业务案例

目前，tRPC-Go 数据校验已在腾讯会议、PCG 看点直播、CDG AMS、安平铁将军等业务上有稳定的实践落地。相关经验分享可参见：

### 4.1 JOOX

- [JOOX 的 trpc-go validation 实践（踩坑经历）](https://km.woa.com/posts/show/504276?kmref=knowledge)

### 4.2 看点

- [trpc 初探系列 (三)--服务参数自动验证](https://km.woa.com/posts/show/442191?kmref=knowledge)

### 4.3 腾讯会议

- [含 Validation 的 proto 示例一](https://git.woa.com/trpcprotocol/wemeet/blob/master/asr_speech_convert/asr_speech_convert.proto)

## 5 FAQ

#### Q1：如何引用 validate.proto？protoc 提示 validate.proto 找不到？

**A1：**

[validate.proto](https://git.woa.com/devsec/protoc-gen-secv/blob/master/validate/validate.proto)文件路径、引用方式，按平台有区分，参考如下：

| 平台           | Proto 文件路径  | 引用方式                             |
| -------------- | -------------- | ------------------------------------ |
| [tRPC 命令行工具（本地使用）](https://git.woa.com/trpc-go/trpc-go-cmdline "tRPC命令行工具") | /etc/trpc      | import "validate.proto"               |
| [Rick 平台](http://trpc.rick.woa.com/ "Rick平台")       | 平台后端公共库 | import "trpc/common/validate.proto"; |

#### Q2：正确引用 validate.proto，生成桩代码失败？

validate.proto 文件已按 Q1 正确引用，运行 tRPC 命令行工具时。提示失败，显示缺失`google/protobuf/`类 proto 文件。
![q2](../../.resources/user_guide/data_validation/q2.png)

**A2：**

需要正确、完整安装`protobuf`。参考指引如下：

- 按系统平台类型，下载最新版本[Protobuf](https://github.com/protocolbuffers/protobuf/releases)

- 解压并将`protoc`移动至`/usr/local/bin/`

  ```shell
  sudo mv protoc3/bin/* /usr/local/bin/
  ```

- 将`protoc3/include`移动至`/usr/local/include/`

  ```shell
  sudo mv protoc3/include/* /usr/local/include/
  ```

- 变更用户组

  ```shell
  sudo chown [user] /usr/local/bin/protoc
  sudo chown -R [user] /usr/local/include/google
  ```

#### Q3：SECV 插件，make build 时报错

make build 时，报错提示：

```
dial tcp 34.64.4.81:443: i/o timeout
```

![q3](../../.resources/user_guide/data_validation/q3.png)
**A3：**
修改 go env 环境变量

```
export GOPROXY=https://goproxy.io
export GO111MODULE=on
```

#### Q4：自定义 Validation 错误消息输出位置

tRPC on HTTP 默认的 Validation 错误消息会通过响应头 trpc-ret/trpc-func-ret 透传。现在需要输出到响应中，或自定义格式。
![q4](../../.resources/user_guide/data_validation/q4.png)
**A4：**
可参考 tRPC http 组件的“[自定义错误码处理函数](https://git.woa.com/trpc-go/trpc-go/tree/master/http)”部分

```go
import (
    "net/http"
   
    "git.code.oa.com/trpc-go/trpc-go/errs"
   
    thttp "git.code.oa.com/trpc-go/trpc-go/http"
    trpc "git.code.oa.com/trpc-go/trpc-go"
)

func init() {
    thttp.DefaultServerCodec.ErrHandler = func(w http.ResponseWriter, r *http.Request, e *errs.Error) {
        // 一般自己定义 retcode retmsg 字段，并组成 json 写到 response body 里面
        w.Write([]byte(fmt.Sprintf(`{"retcode": %d, "retmsg":"%s"}`, e.Code, e.Msg)))
        // 每个业务团队可以定义到自己的 git 上，业务代码 import 进来即可
    }
}
```

#### Q5：已经按配置流程操作，但 Validation 不生效

已按如下流程操作，但测试时 Validation 仍不生效

1. proto 编写的时候按照指定语法
2. main 里面 import
3. 修改 trpc_go.yaml 开启 filter
4. 编译然后发布
5. 测试

**A5：**
调整 go.mod 文件，`go build`重新生成二进制文件并重启服务。

#### Q6：Proto3 中如何实现一个字段如果不传的话，就不进行校验；如果传了，就进行校验

**A6：**
可以使用[Wrapper](https://github.com/protocolbuffers/protobuf/blob/d0f91c863ae0fbb75b41460c8bbb786ade197a0f/src/google/protobuf/util/internal/testdata/wrappers.proto)，假设原始 pb 如下：

```proto
message QueryRuleRequest {
    uint32 project_id = 1;
    uint32 case_bid = 2;
}
```

使用 Wrapper 实现“一个字段如果不传的话，就不进行校验；如果传了，就进行校验”，如下：

```proto
import "google/protobuf/wrappers.proto";

message QueryRuleRequest {
    uint32 project_id = 1;
    google.protobuf.UInt32Value case_bid = 2 [(validate.rules).int32.gt = 3];
}
```

#### Q7：SECV protoc 插件安装失败

使用 go get 下载 SECV 插件后，显示安装失败：
![q7](../../.resources/user_guide/data_validation/q7.png)

**A7：**
一般是进入的目录不对，根据以下完整流程操作：

```shell
# 下载插件源代码至$GOPATH
go get -d git.code.oa.com/devsec/protoc-gen-secv

# 进入目录
cd  $GOPATH/src/git.code.oa.com/devsec/protoc-gen-secv

# 执行命令，将 SECV 安装至$GOPATH/bin
make build
```

#### Q8：proto 保存时提示“字段未正确进行参数校验，请限定字符集范围”，应该如何处理？

![q8](../../.resources/user_guide/data_validation/q8.png)

**A8：**

按照指引为 proto 的 string 字段，添加校验规则，限定格式/字符范围 + 长度。如需对整个 app_svr 模块加白，请企业微信联系 braveyzhang、yuyangzhou

推荐方案如下：

|  规则 | 示例  |
| ------------ | ------------ |
| 为 string 字段设置了 well-known 类型限制 <br/>* 包括：tsecstr、email、address、hostname、ip、ipv4、uri、uri_ref、uuid、alphabets、alphanums、lowercase  |  string x = 1 [(validate.rules).string.tsecstr = true] // 限制传入参数只能为 tsecstr 默认安全类型|
| 为 string 字段设置了正则表达式限制  | string x = 1 [(validate.rules).string.pattern = "(?i)^[0-9a-f]+$"]  // 用正则限制了字符范围内|
| 为 string 字段组合设置了格式/字符范围 + 长度的限制  |  string x = 1 [(validate.rules).string = { tsecstr: true, min_len: 2 }]; // 同时设置了格式/字符范围 + 最小长度的限制|
| 为 string 字段设置加白注释（*不推荐，但业务需要时可使用）  |  string x = 1; // unsafe_str|

#### Q9：引入公共 proto 文件，无法生成校验桩代码？

**A9：**
在 Rick 平台上，可以为公共 proto 文件单独生成桩代码。选择“扩展功能” -> “TRPC-GO Stub Mod” / “TRPC-GO-服务生成”

#### Q10：proto 文件中未定义 service，平台报错，无法生成桩代码？

**A10：**
可以为 proto 文件添加一个空 service，再点击生成“Stub Mod”或“Go-服务”。例如，可参考[样例 proto](http://trpc.rick.woa.com/rick/pb/view_protobuf?id=20712)：

```
service Void {}
```

#### Q11：使用 validation 后，如何对数据校验逻辑进行单元测试？单元测试时，数据校验逻辑无效？

**A11：**

**方案一、集成/接口测试**。使用 Rick 平台的接口测试，相当于直接做集成测试。功能入口：<http://trpc.rick.woa.com/rick/test/list?id=19855>

**方案二、单元测试**。在代码或单元测试用利中调用`Validate()方法`

```go
var (
    secvValidateClientProxy pb.SecvValidateClientProxy
)

func init() {

    log.SetFlags(log.LstdFlags | log.Lshortfile)

    // 默认使用配置文件中配置
    err := trpc.LoadGlobalConfig("../trpc_go.yaml")
    if err == nil {
        for _, cfg := range trpc.GlobalConfig().Client.Service {
            client.RegisterClientConfig(cfg.Callee, cfg)
        }
    }

    // 如果配置文件未提供，默认使用如下选项
    opts := []client.Option{
        client.WithProtocol("trpc"),
        client.WithNetwork("tcp4"),
        client.WithTarget("ip://127.0.0.1:8002"),
        client.WithTimeout(time.Second * 2000),
    }

    secvValidateClientProxy = pb.NewSecvValidateClientProxy(opts...)
}

func Test_SecvValidate_Validate(t *testing.T) {
    ctx := context.Background()
    convey.Convey("测试合法", t, func() {
        req := &pb.ValidateReq{}
        req.V1 = "Abc123"
        req.V2 = "Abc"
        req.V4 = "12345"
        req.V5 = "1234"
        req.V6 = "123456"
        req.V7 = "fizz101buzz"
        req.V8 = "foo.proto"
        req.V101 = 9
        req.V102 = 21
        req.V103 = 31
        req.V104 = 1
        req.V105 = 0.1
        req.V106 = 1.23
        rsp, _ := secvValidateClientProxy.Validate(ctx, req)
        convey.So(rsp.Code, convey.ShouldEqual, 0)
    })

    convey.Convey("测试 V1 非法", t, func() {
        req := &pb.ValidateReq{}
        req.V1 = "_abc123"
        rsp, err := secvValidateClientProxy.Validate(ctx, req)
        tLog.ErrorContextf(ctx, "rsp: %v err: %v\n", rsp, err)
        convey.So(err, convey.ShouldNotBeNil)
    })

    convey.Convey("测试 V101 非法", t, func() {
        req := &pb.ValidateReq{}
        req.V101 = 10
        rsp, err := secvValidateClientProxy.Validate(ctx, req)
        tLog.ErrorContextf(ctx, "rsp: %v err: %v\n", rsp, err)
        convey.So(err, convey.ShouldNotBeNil)
    })

    convey.Convey("测试 V106 非法", t, func() {
        req := &pb.ValidateReq{}
        req.V106 = 1.22
          // 调用 Validate 方法，进行校验的单元测试
        rsp, err := secvValidateClientProxy.Validate(ctx, req)
        tLog.ErrorContextf(ctx, "rsp: %v err: %v\n", rsp, err)
        convey.So(err, convey.ShouldNotBeNil)
    })
}
```

#### Q12：uint32 字段类型，使用 Validation 后，传入字符串仍有效？

**A12：**
PB 本身的特性，允许 uint32 字段类型传入整数或字符串：`JSON value will be a decimal string. Either numbers or strings are accepted.`

可以通过引入`trpc.proto`追加字段类型描述解决。

```proto
import "trpc/common/trpc.proto";

// ...省略 proto 定义内容

uint32 port = 4 [(validate.rules).uint32.gt = 1, (trpc.go_tag)='json:",int"'];
```

更多细节参见码客讨论[《pb 定义接口字段类型为 uint32 时：http 调用接口时 json 传入{"msg":"1"}和{"msg":1}，{"msg":"1"}的请求为什么未被拦截？》](https://mk.woa.com/q/275511)

#### Q13：Rick 生成的桩代码报证书错误？

![q13](../../.resources/user_guide/data_validation/q13.png)

**A13：**
内网 Go 切换使用 [https://goproxy.woa.com/ 详参考 [《修复指引》](https://goproxy.woa.com/faq.html)

#### Q14：secv 下载报证书错误

```raw
package git.code.oa.com/devsec/protoc-gen-secv: unrecognized import path "git.code.oa.com/devsec/protoc-gen-secv": https fetch: Get "https://git.code.oa.com/devsec/protoc-gen-secv?go-get=1": x509: certificate has expired or is not yet valid: current time 2021-12-21T15:04:08+08:00 is after 2021-09-06T05:19:55Z
```

**A14：**
证书过期，详参考[《修复指引》](https://iwiki.woa.com/pages/viewpage.action?pageId=1004304553)

#### Q15：重复 proto 名，secv 插件 panic

**A15：**
设置环境变量 `GOLANG_PROTOBUF_REGISTRATION_CONFLICT=warn ./main`

## 更多问题

请参考 [tRPC 技术咨询](https://iwiki.woa.com/p/491739953) 以寻求帮助
