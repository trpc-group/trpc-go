[English](protocol.md) | 中文

# 怎么开发一个 protocol 类型的插件

本指南将介绍如何开发一个不依赖配置文件的 protocol 类型的插件。

开发一个 protocol 类型的插件至少需要实现以下两个子功能：

- 实现 `codec.Framer` 和  `codec.FramerBuilder` 接口, 从连接中读取出完整的业务包
- 实现 `codec.Codec` 接口， 从完整的二进制网络数据包解析出二进制请求包体，和把二进制响应包体打包成一个完整的二进制网络数据

除此之外，根据具体的 protocol，还有可能需要实现 `codec.Serializer` 和 `codec.Compressor` 接口。

下面以实现 trpc-codec 中的 “rawstring” 协议为例，来介绍相关开发步骤，更具体的代码可以参考[这里](https://github.com/trpc-ecosystem/go-codec/tree/main/rawstring)。

"rawstring"协议是一种简单的基于 tcp 的通信协议，其特点是以 “\n” 字符为分隔符进行收发包。

## 实现 `codec.Framer` 和  `codec.FramerBuilder` 接口

```go
type FramerBuilder struct{}

func (fd *FramerBuilder) New(reader io.Reader) transport.Framer {
    return &framer{
        reader: reader,
    }
}

type framer struct {
    reader io.Reader
}

func (f *framer) ReadFrame() (msg []byte, err error) {
    reader := bufio.NewReader(f.reader)
    return reader.ReadBytes('\n')
}
```

将实现好的 FramerBuilder 注册到 `transport` 包

```go
var DefaultFramerBuilder = &FramerBuilder{}
func init() {
    transport.RegisterFramerBuilder("rawstring", DefaultFramerBuilder)
}
```

## 实现 `codec.Codec` 接口

### 实现服务端 Codec

```go
type serverCodec struct{}

func (sc *serverCodec) Decode(_ codec.Msg, req []byte) ([]byte, error) {
    return req, nil
}

func (sc *serverCodec) Encode(_ codec.Msg, rsp []byte) ([]byte, error) {
    return []byte(string(rsp) + "\n"), nil
}
```

### 实现客户端 Codec

```go
type clientCodec struct{}

func (cc *clientCodec) Encode(_ codec.Msg, reqBody []byte) ([]byte, error) {
    return []byte(string(reqBody) + "\n"), nil
}

func (cc *clientCodec) Decode(_ codec.Msg, rspBody []byte) ([]byte, error) {
    return rspBody, nil
}
```

### 将实现好的 Codec 注册到 `codec` 包

```go
func init() {
	codec.Register("rawstring", &serverCodec{}, &clientCodec{})
}
```

## 更多例子

更多例子可以参考 [trpc-codec 代码仓库](https://github.com/trpc-ecosystem/go-codec)