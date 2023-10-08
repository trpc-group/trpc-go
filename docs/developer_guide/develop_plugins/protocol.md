English | [中文](protocol_zh_CN.md)

# How to develop a protocol type plugin

This guide will introduce how to develop a protocol type plugin that does not depend on a configuration file.

To develop a protocol type plugin, you need to implement at least the following two sub-functions:

- Implement `codec.Framer` and `codec.FramerBuilder` interfaces to read the complete business package from the connection
- Implement `codec.Codec` interface to parse the binary request body from the complete binary network data package and package the binary response body into a complete binary network data package

In addition, depending on the specific protocol, you may also need to implement `codec.Serializer` and `codec.Compressor` interfaces.

The following uses the implementation of the "rawstring" protocol in trpc-codec as an example to introduce the related development steps. More specific code can be found [here](https://github.com/trpc-ecosystem/go-codec/tree/main/rawstring).

The "rawstring" protocol is a simple communication protocol based on tcp, characterized by using the "\n" character as the delimiter for sending and receiving packets.

## Implement `codec.Framer` and `codec.FramerBuilder` interfaces

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

Register the implemented FramerBuilder to the `transport` package

```go
var DefaultFramerBuilder = &FramerBuilder{}
func init() {
    transport.RegisterFramerBuilder("rawstring", DefaultFramerBuilder)
}
```

## Implement `codec.Codec` interface

### Implement server-side Codec

```go
type serverCodec struct{}

func (sc *serverCodec) Decode(_ codec.Msg, req []byte) ([]byte, error) {
    return req, nil
}

func (sc *serverCodec) Encode(_ codec.Msg, rsp []byte) ([]byte, error) {
    return []byte(string(rsp) + "\n"), nil
}
```

### Implement client-side Codec

```go
type clientCodec struct{}

func (cc *clientCodec) Encode(_ codec.Msg, reqBody []byte) ([]byte, error) {
    return []byte(string(reqBody) + "\n"), nil
}

func (cc *clientCodec) Decode(_ codec.Msg, rspBody []byte) ([]byte, error) {
    return rspBody, nil
}
```

### Register the implemented Codec to the `codec` package

```go
func init() {
	codec.Register("rawstring", &serverCodec{}, &clientCodec{})
}
```

## More examples

For more examples, you can refer to the [trpc-codec code repository](https://github.com/trpc-ecosystem/go-codec)