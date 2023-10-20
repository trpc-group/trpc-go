[English](README.md) | 中文

# tRPC-Go HTTP 协议

tRPC-Go 框架支持搭建与 HTTP 相关的三种服务:

1. 泛 HTTP 标准服务 (无需桩代码及 IDL 文件)
2. 泛 HTTP RPC 服务 (共享 RPC 协议使用的桩代码以及 IDL 文件)
3. 泛 HTTP RESTful 服务 (基于 IDL 及桩代码提供 RESTful API)

其中 RESTful 相关文档见 [/restful](/restful/)

## 泛 HTTP 标准服务

tRPC-Go 框架提供了泛 HTTP 标准服务能力, 主要是在标准库 HTTP 的能力上添加了服务注册、服务发现、拦截器等能力, 使 HTTP 协议能够无缝接入 tRPC 生态

相较于 tRPC 协议而言, 泛 HTTP 标准服务服务不依赖桩代码, 因此服务侧对应的 protocol 名为 `http_no_protocol`

### 服务端

#### 配置编写

在 `trpc_go.yaml` 配置文件中配置 service，协议为 `http_no_protocol`，http2 则为 `http2_no_protocol`:

```yaml
server:
  service:  # 业务服务提供的 service，可以有多个
    - name: trpc.app.server.stdhttp  # service 的路由名称
      network: tcp  # 网络监听类型，tcp 或 udp
      protocol: http_no_protocol  # 应用层协议 http_no_protocol
      timeout: 1000  # 请求最长处理时间，单位毫秒
      ip: 127.0.0.1
      port: 8080  # 服务监听端口
```

注意确保配置文件的正常加载

#### 代码编写

##### 单一 URL 注册

```go
import (
    "net/http"

    "trpc.group/trpc-go/trpc-go/codec"
    "trpc.group/trpc-go/trpc-go/log"
    thttp "trpc.group/trpc-go/trpc-go/http"
    trpc "trpc.group/trpc-go/trpc-go"
)

func main() {
    s := trpc.NewServer()
    thttp.HandleFunc("/xxx", handle) 
    // 注册 NoProtocolService 时传的参数必须和配置中的 service name 一致: s.Service("trpc.app.server.stdhttp")
    thttp.RegisterNoProtocolService(s.Service("trpc.app.server.stdhttp")) 
    s.Serve()
}

func handle(w http.ResponseWriter, r *http.Request) error {
    // handle 的编写方法完全同标准库 HTTP 的使用方式一致
    // 比如可以在 r 中读取 Header 等
    // 可以在 r.Body 对 client 进行流式读包
    clientReq, err := io.ReadAll(r.Body)
    if err != nil { /*..*/ }
    // 最后使用 w 来进行回包
    w.Header().Set("Content-type", "application/text")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("response body"))
    return nil
}
```

##### MUX 注册

```go
import (
    "net/http"

    "trpc.group/trpc-go/trpc-go/codec"
    "trpc.group/trpc-go/trpc-go/log"
    thttp "trpc.group/trpc-go/trpc-go/http"
    trpc "trpc.group/trpc-go/trpc-go"
    "github.com/gorilla/mux"
)

func main() {
    s := trpc.NewServer()
    // 路由注册
    router := mux.NewRouter()
    router.HandleFunc("/{dir0}/{dir1}/{day}/{hour}/{vid:[a-z0-9A-Z]+}_{index:[0-9]+}.jpg", handle).
        Methods("GET")
    // 注册 RegisterNoProtocolServiceMux 时传的参数必须和配置中的 service name 一致: s.Service("trpc.app.server.stdhttp")
    thttp.RegisterNoProtocolServiceMux(s.Service("trpc.app.server.stdhttp"), router)
    s.Serve()
}

func handle(w http.ResponseWriter, r *http.Request) error {
    // 取 url 中的参数
    vars := mux.Vars(r)
    vid := vars["vid"]
    index := vars["index"]
    log.Infof("vid: %s, index: %s", vid, index)
    return nil
}
```

### 客户端

这里指的是调用一个标准 HTTP 服务, 下游这个标准 HTTP 服务并不一定是基于 tRPC-Go 框架构建的

最简洁的方式实际上是直接使用标准库提供的 HTTP Client, 但是就无法使用服务发现以及各种插件拦截器提供的能力(比如监控上报)

#### 配置编写

```yaml
client:  # 客户端调用的后端配置
  timeout: 1000  # 针对所有后端的请求最长处理时间
  namespace: Development  # 针对所有后端的环境
  filter:  # 针对所有后端调用函数前后的拦截器列表
    - simpledebuglog  # 这是 debug log 拦截器, 可以再添加其他拦截器, 比如监控等
  service:  # 针对单个后端的配置
    - name: trpc.app.server.stdhttp  # 下游 http 服务的 service name 
    ## 可以使用 target 来选用其他的 selector, 只有 service name 的情况下默认会使用北极星做服务发现(在使用了北极星插件的情况下)
    #   target: polaris://trpc.app.server.stdhttp  # 或者 ip://127.0.0.1:8080 来指定 ip:port 进行调用
```

#### 代码编写

```go
package main

import (
    "context"

    trpc "trpc.group/trpc-go/trpc-go"
    "trpc.group/trpc-go/trpc-go/client"
    "trpc.group/trpc-go/trpc-go/codec"
  trpc "trpc.group/trpc-go/trpc-go"
    "trpc.group/trpc-go/trpc-go/log"
)

// Data 请求报文数据
type Data struct {
    Msg string
}

func main() {
    // 省略掉 tRPC-Go 框架配置加载部分, 假如以下逻辑在某个 RPC handle 中, 配置一般已经正常加载
    // 创建 ClientProxy, 设置协议为 HTTP 协议，序列化为 JSON
    httpCli := http.NewClientProxy("trpc.app.server.stdhttp",
        client.WithSerializationType(codec.SerializationTypeJSON))
    reqHeader := &http.ClientReqHeader{}
    // 为 HTTP Head 添加 request 字段
    reqHeader.AddHeader("request", "test")
    rspHead := &http.ClientRspHeader{}
    req := &Data{Msg: "Hello, I am stdhttp client!"}
    rsp := &Data{}
    // 发送 HTTP POST 请求
    if err := httpCli.Post(context.Background(), "/v1/hello", req, rsp,
        client.WithReqHead(reqHeader),
        client.WithRspHead(rspHead),
    ); err != nil {
        log.Warn("get http response err")
        return
    }
    // 获取 HTTP 响应报文头中的 reply 字段
    replyHead := rspHead.Response.Header.Get("reply")
    log.Infof("data is %s, request head is %s\n", rsp, replyHead)
}
```


## 泛 HTTP RPC 服务

相较于**泛 HTTP 标准服务**, 泛 HTTP RPC 服务的最大区别是复用了 IDL 协议文件及其生成的桩代码, 同时无缝融入了 tRPC 生态(服务注册、服务路由、服务发现、各种插件拦截器等)

注意: 

在这种服务形式下, HTTP 协议与 tRPC 协议保持一致：当服务端返回失败时，body 为空，错误码错误信息放在 HTTP header 里

### 服务端

#### 配置编写

首先需要生成桩代码:

```shell
trpc create -p helloworld.proto --protocol http -o out
```

假如本身已经是一个 tRPC 服务已经存在桩代码, 只是想在同样的接口上支持 HTTP 协议, 那么无需再次生成桩代码, 而是在配置中添加 `http` 协议项即可

```yaml
server: # 服务端配置
  service:
    ## 同一套接口可以通过两份配置同时提供 trpc 协议以及 http 协议服务
    - name: trpc.test.helloworld.Greeter  # service 的路由名称
      ip: 127.0.0.0  # 服务监听 ip 地址 可使用占位符 ${ip},ip 和 nic 二选一，优先 ip
      port: 80  # 服务监听端口 可使用占位符 ${port}
      protocol: trpc  # 应用层协议 trpc http
    ## 以下为主要示例, 注意应用层协议为 http
    - name: trpc.test.helloworld.GreeterHTTP  # service 的路由名称
      ip: 127.0.0.0  # 服务监听 ip 地址 可使用占位符 ${ip},ip 和 nic 二选一，优先 ip
      port: 80  # 服务监听端口 可使用占位符 ${port}
      protocol: http  # 应用层协议 trpc http
```

#### 代码编写

```go
import (
    "context"
    "fmt"

  trpc "trpc.group/trpc-go/trpc-go"
    "trpc.group/trpc-go/trpc-go/client"
    pb "github.com/xxxx/helloworld/pb"
)

func main() {
    s := trpc.NewServer()
    hello := Hello{}
    pb.RegisterHelloTrpcGoService(s.Service("trpc.test.helloworld.Greeter"), &hello)
    // 和一般的 tRPC 服务注册是一致的
    pb.RegisterHelloTrpcGoService(s.Service("trpc.test.helloworld.GreeterHTTP"), &hello)
    log.Println(s.Serve())
}

type Hello struct {}

// RPC 服务接口的实现无需感知 HTTP 协议, 只需按照通常的逻辑处理请求并返回响应即可
func (h *Hello) Hello(ctx context.Context, req *pb.HelloReq) (*pb.HelloRsp, error) {
    fmt.Println("--- got HelloReq", req)
    time.Sleep(time.Second)
    return &pb.HelloRsp{Msg: "Welcome " + req.Name}, nil
}
```
#### 自定义 URL path

默认为 `/package.service/method`，可通过 alias 参数自定义任意 URL

- 协议定义：

```protobuf
syntax = "proto3";
package trpc.app.server;
option go_package="github.com/your_repo/app/server";

import "trpc.proto";

message Request {
    bytes req = 1;
}

message Reply {
    bytes rsp = 1;
}

service Greeter {
    rpc SayHello(Request) returns (Reply) {
        option (trpc.alias) = "/cgi-bin/module/say_hello";
    };
}
```

#### 自定义错误码处理函数

默认错误码处理函数，会将错误码填充到 HTTP header 的 `trpc-ret/trpc-func-ret` 字段中，也可以通过自己定义 ErrorHandler 进行替换。

```golang
import (
    "net/http"

    "trpc.group/trpc-go/trpc-go/errs"
    thttp "trpc.group/trpc-go/trpc-go/http"
)

func init() {
    thttp.DefaultServerCodec.ErrHandler = func(w http.ResponseWriter, r *http.Request, e *errs.Error) {
        // 一般自行定义 retcode retmsg 字段，组成 json 并写到 response body 里
        w.Write([]byte(fmt.Sprintf(`{"retcode":%d, "retmsg":"%s"}`, e.Code, e.Msg)))
        // 每个业务团队可以定义到自己的 git 里，业务代码 import 进来即可
    }
}
```

### 客户端

#### 配置编写

和一般的 RPC Client 书写方式相同, 只需把配置 `protocol` 改为 `http`:

```yaml
client:
  namespace: Development  # 针对所有后端的环境
  filter:  # 针对所有后端调用函数前后的拦截器列表
  service:  # 针对单个后端的配置
    - name: trpc.test.helloworld.GreeterHTTP  # 后端服务的 service name
      network: tcp  # 后端服务的网络类型 tcp udp
      protocol: http  # 应用层协议 trpc http
      ## 可以使用 target 来选用其他的 selector, 只有 service name 的情况下默认会使用北极星做服务发现(在使用了北极星插件的情况下)
      # target: ip://127.0.0.1:8000  # 请求服务地址
      timeout: 1000  # 请求最长处理时间
```

#### 代码编写

```go
import (
    "context"
    "net/http"

    "trpc.group/trpc-go/trpc-go/client"
    thttp "trpc.group/trpc-go/trpc-go/http"
    "trpc.group/trpc-go/trpc-go/log"
    pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)

func main() {
    // 省略掉 tRPC-Go 框架配置加载部分, 假如以下逻辑在某个 RPC handle 中, 配置一般已经正常加载
    // 创建 ClientProxy, 设置协议为 HTTP 协议, 序列化为 JSON
    proxy := pb.NewGreeterClientProxy()
    reqHeader := &thttp.ClientReqHeader{}
    // 必须留空或设置为 "POST"
    reqHeader.Method = "POST"
    // 为 HTTP Head 添加 request 字段
    reqHeader.AddHeader("request", "test")
    // 设置 Cookie
    cookie := &http.Cookie{Name: "sample", Value: "sample", HttpOnly: false}
    reqHeader.AddHeader("Cookie", cookie.String())
    req := &pb.HelloRequest{Msg: "Hello, I am tRPC-Go client."}
    rspHead := &thttp.ClientRspHeader{}
    // 发送 HTTP RPC 请求
    rsp, err := proxy.SayHello(context.Background(), req,
        client.WithReqHead(reqHeader),
        client.WithRspHead(rspHead),
        // 此处可以使用代码强制覆盖 trpc_go.yaml 配置中的 target 字段来设置其他 selector, 一般没必要, 这里只是展示有这个功能
        // client.WithTarget("ip://127.0.0.1:8000"),
    )
    if err != nil {
        log.Warn("get http response err")
        return
    }
    // 获取 HTTP 响应报文头中的 reply 字段
    replyHead := rspHead.Response.Header.Get("reply")
    log.Infof("data is %s, request head is %s\n", rsp, replyHead)
}
```

## FAQ

### 客户端及服务端开启 HTTPS

#### 双向认证

##### 仅配置填写

只需在 `trpc_go.yaml` 中添加相应的配置项(证书以及私钥):

```yaml
server:  # 服务端配置
  service:  # 业务服务提供的 service，可以有多个
    - name: trpc.app.server.stdhttp
      network: tcp
      protocol: http_no_protocol # 泛 HTTP RPC 服务则填 http
      tls_cert: "../testdata/server.crt" # 添加证书路径
      tls_key: "../testdata/server.key" # 添加私钥路径
      ca_cert: "../testdata/ca.pem" # CA 证书, 需要双向认证时可填写
client:  # 客户端配置
  service:  # 业务服务提供的 service，可以有多个
    - name: trpc.app.server.stdhttp
      network: tcp
      protocol: http
      tls_cert: "../testdata/server.crt" # 添加证书路径
      tls_key: "../testdata/server.key" # 添加私钥路径
      ca_cert: "../testdata/ca.pem" # CA 证书, 需要双向认证时可填写
```

代码中不在需要额外考虑任何和 TLS/HTTPS 相关的操作(不需要指定 scheme 为 `https`, 不需要手动添加 `WithTLS` option, 也不需要在 `WithTarget` 等其他地方想办法塞一个有关 HTTPS 的标识进去)

##### 仅代码填写

服务端使用 `server.WithTLS` 依次指定服务端证书、私钥、CA 证书即可:

```go
server.WithTLS(
	"../testdata/server.crt",
	"../testdata/server.key",
	"../testdata/ca.pem",
),
```

客户端使用 `client.WithTLS` 依次指定客户端端证书、私钥、CA 证书即可:

```go
client.WithTLS(
	"../testdata/client.crt",
	"../testdata/client.key",
	"../testdata/ca.pem",
	"localhost", // 填写 server name
),
```

除了这两个 option 以外, 代码中不在需要额外考虑任何和 TLS/HTTPS 相关的操作

示例如下:

```go
func TestHTTPSUseClientVerify(t *testing.T) {
	const (
		network = "tcp"
		address = "127.0.0.1:0"
	)
	ln, err := net.Listen(network, address)
	require.Nil(t, err)
	defer ln.Close()
	serviceName := "trpc.app.server.Service" + t.Name()
	service := server.New(
		server.WithServiceName(serviceName),
		server.WithNetwork("tcp"),
		server.WithProtocol("http_no_protocol"),
		server.WithListener(ln),
		server.WithTLS(
			"../testdata/server.crt",
			"../testdata/server.key",
			"../testdata/ca.pem",
		),
	)
	thttp.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) error {
		w.Write([]byte(t.Name()))
		return nil
	})
	thttp.RegisterNoProtocolService(service)
	s := &server.Server{}
	s.AddService(serviceName, service)
	go s.Serve()
	defer s.Close(nil)
	time.Sleep(100 * time.Millisecond)

	c := thttp.NewClientProxy(
		serviceName,
		client.WithTarget("ip://"+ln.Addr().String()),
	)
	req := &codec.Body{}
	rsp := &codec.Body{}
	require.Nil(t,
		c.Post(context.Background(), "/", req, rsp,
			client.WithCurrentSerializationType(codec.SerializationTypeNoop),
			client.WithSerializationType(codec.SerializationTypeNoop),
			client.WithCurrentCompressType(codec.CompressTypeNoop),
			client.WithTLS(
				"../testdata/client.crt",
				"../testdata/client.key",
				"../testdata/ca.pem",
				"localhost",
			),
		))
	require.Equal(t, []byte(t.Name()), rsp.Data)
}
```

#### 不认证客户端证书

##### 仅配置填写

只需在 `trpc_go.yaml` 中添加相应的配置项(证书以及私钥):

```yaml
server:  # 服务端配置
  service:  # 业务服务提供的 service，可以有多个
    - name: trpc.app.server.stdhttp
      network: tcp
      protocol: http_no_protocol # 泛 HTTP RPC 服务则填 http
      tls_cert: "../testdata/server.crt" # 添加证书路径
      tls_key: "../testdata/server.key" # 添加私钥路径
      # ca_cert: "" # CA 证书, 不认证客户端证书时此处不填或留空
client:  # 客户端配置
  service:  # 业务服务提供的 service，可以有多个
    - name: trpc.app.server.stdhttp
      network: tcp
      protocol: http
      # tls_cert: "" # 证书路径, 不认证客户端证书时此处不填或留空
      # tls_key: "" # 私钥路径, 不认证客户端证书时此处不填或留空
      ca_cert: "none" # CA 证书, 不认证客户端证书时此处必须填写, 并且要填 "none"
```

可以双向认证部分, 主要的区别在于服务端的 `ca_cert` 需要留空, 客户端的 `ca_cert` 需要填 `none`

代码中不在需要额外考虑任何和 TLS/HTTPS 相关的操作(不需要指定 scheme 为 `https`, 不需要手动添加 `WithTLS` option, 也不需要在 `WithTarget` 等其他地方想办法塞一个有关 HTTPS 的标识进去)

##### 仅代码填写

服务端使用 `server.WithTLS` 依次指定服务端证书、私钥、CA 证书即可:

```go
server.WithTLS(
	"../testdata/server.crt",
	"../testdata/server.key",
	"", // CA 证书, 不认证客户端证书时此处留空
),
```

客户端使用 `client.WithTLS` 依次指定客户端端证书、私钥、CA 证书即可:

```go
client.WithTLS(
	"", // 证书路径, 留空
	"", // 私钥路径, 留空
	"none", // CA 证书, 不认证客户端证书时此处必须填 "none"
	"", // server name, 留空
),
```

除了这两个 option 以外, 代码中不在需要额外考虑任何和 TLS/HTTPS 相关的操作

示例如下:


```go
func TestHTTPSSkipClientVerify(t *testing.T) {
	const (
		network = "tcp"
		address = "127.0.0.1:0"
	)
	ln, err := net.Listen(network, address)
	require.Nil(t, err)
	defer ln.Close()
	serviceName := "trpc.app.server.Service" + t.Name()
	service := server.New(
		server.WithServiceName(serviceName),
		server.WithNetwork("tcp"),
		server.WithProtocol("http_no_protocol"),
		server.WithListener(ln),
		server.WithTLS(
			"../testdata/server.crt",
			"../testdata/server.key",
			"",
		),
	)
	thttp.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) error {
		w.Write([]byte(t.Name()))
		return nil
	})
	thttp.RegisterNoProtocolService(service)
	s := &server.Server{}
	s.AddService(serviceName, service)
	go s.Serve()
	defer s.Close(nil)
	time.Sleep(100 * time.Millisecond)

	c := thttp.NewClientProxy(
		serviceName,
		client.WithTarget("ip://"+ln.Addr().String()),
	)
	req := &codec.Body{}
	rsp := &codec.Body{}
	require.Nil(t,
		c.Post(context.Background(), "/", req, rsp,
			client.WithCurrentSerializationType(codec.SerializationTypeNoop),
			client.WithSerializationType(codec.SerializationTypeNoop),
			client.WithCurrentCompressType(codec.CompressTypeNoop),
			client.WithTLS(
				"", "", "none", "",
			),
		))
	require.Equal(t, []byte(t.Name()), rsp.Data)
}
```


### 客户端使用 io.Reader 进行流式发送文件

需要 trpc-go 版本 >= v0.13.0

关键点在于将一个 `io.Reader` 填到 `thttp.ClientReqHeader.ReqBody` 字段上 (`body` 是一个 `io.Reader`):

```go
reqHeader := &thttp.ClientReqHeader{
	Header:  header,
	ReqBody: body, // Stream send.
}
```

然后在调用时指定 `client.WithReqHead(reqHeader)`:

```go
c.Post(context.Background(), "/", req, rsp,
	client.WithCurrentSerializationType(codec.SerializationTypeNoop),
	client.WithSerializationType(codec.SerializationTypeNoop),
	client.WithCurrentCompressType(codec.CompressTypeNoop),
	client.WithReqHead(reqHeader),
)
```

示例如下:

```go
func TestHTTPStreamFileUpload(t *testing.T) {
	// Start server.
	const (
		network = "tcp"
		address = "127.0.0.1:0"
	)
	ln, err := net.Listen(network, address)
	require.Nil(t, err)
	defer ln.Close()
	go http.Serve(ln, &fileHandler{})
	// Start client.
	c := thttp.NewClientProxy(
		"trpc.app.server.Service_http",
		client.WithTarget("ip://"+ln.Addr().String()),
	)
	// Open and read file.
	fileDir, err := os.Getwd()
	require.Nil(t, err)
	fileName := "README.md"
	filePath := path.Join(fileDir, fileName)
	file, err := os.Open(filePath)
	require.Nil(t, err)
	defer file.Close()
	// Construct multipart form file.
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("field_name", filepath.Base(file.Name()))
	require.Nil(t, err)
	io.Copy(part, file)
	require.Nil(t, writer.Close())
	// Add multipart form data header.
	header := http.Header{}
	header.Add("Content-Type", writer.FormDataContentType())
	reqHeader := &thttp.ClientReqHeader{
		Header:  header,
		ReqBody: body, // Stream send.
	}
	req := &codec.Body{}
	rsp := &codec.Body{}
	// Upload file.
	require.Nil(t,
		c.Post(context.Background(), "/", req, rsp,
			client.WithCurrentSerializationType(codec.SerializationTypeNoop),
			client.WithSerializationType(codec.SerializationTypeNoop),
			client.WithCurrentCompressType(codec.CompressTypeNoop),
			client.WithReqHead(reqHeader),
		))
	require.Equal(t, []byte(fileName), rsp.Data)
}

type fileHandler struct{}

func (*fileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, h, err := r.FormFile("field_name")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	// Write back file name.
	w.Write([]byte(h.Filename))
	return
}
```

### 客户端使用 io.Reader 进行流式读取回包

需要 trpc-go 版本 >= v0.13.0

关键在于添加 `thttp.ClientRspHeader` 并指定 `thttp.ClientRspHeader.ManualReadBody` 字段为 `true`:

```go
rspHead := &thttp.ClientRspHeader{
	ManualReadBody: true,
}
```

然后调用时加上 `client.WithRspHead(rspHead)`:

```go
c.Post(context.Background(), "/", req, rsp,
	client.WithCurrentSerializationType(codec.SerializationTypeNoop),
	client.WithSerializationType(codec.SerializationTypeNoop),
	client.WithCurrentCompressType(codec.CompressTypeNoop),
	client.WithRspHead(rspHead),
)
```

最后可以在 `rspHead.Response.Body` 上进行流式读包:

```go
body := rspHead.Response.Body // Do stream reads directly from rspHead.Response.Body.
defer body.Close()            // Do remember to close the body.
bs, err := io.ReadAll(body)
```

示例如下:

```go
func TestHTTPStreamRead(t *testing.T) {
	// Start server.
	const (
		network = "tcp"
		address = "127.0.0.1:0"
	)
	ln, err := net.Listen(network, address)
	require.Nil(t, err)
	defer ln.Close()
	go http.Serve(ln, &fileServer{})

	// Start client.
	c := thttp.NewClientProxy(
		"trpc.app.server.Service_http",
		client.WithTarget("ip://"+ln.Addr().String()),
	)

	// Enable manual body reading in order to
	// disable the framework's automatic body reading capability,
	// so that users can manually do their own client-side streaming reads.
	rspHead := &thttp.ClientRspHeader{
		ManualReadBody: true,
	}
	req := &codec.Body{}
	rsp := &codec.Body{}
	require.Nil(t,
		c.Post(context.Background(), "/", req, rsp,
			client.WithCurrentSerializationType(codec.SerializationTypeNoop),
			client.WithSerializationType(codec.SerializationTypeNoop),
			client.WithCurrentCompressType(codec.CompressTypeNoop),
			client.WithRspHead(rspHead),
		))
	require.Nil(t, rsp.Data)
	body := rspHead.Response.Body // Do stream reads directly from rspHead.Response.Body.
	defer body.Close()            // Do remember to close the body.
	bs, err := io.ReadAll(body)
	require.Nil(t, err)
	require.NotNil(t, bs)
}

type fileServer struct{}

func (*fileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./README.md")
	return
}
```


### 客户端服务端收发 HTTP chunked

1. 客户端发送 HTTP chunked: 
   1. 添加 `chunked` Transfer-Encoding header
   2. 然后使用 io.Reader 进行发包
2. 客户端接收 HTTP chunked: Go 标准库 HTTP 自动支持了对 chunked 的处理, 上层用户对其是无感知的, 只需在 resp.Body 上面循环读直至 `io.EOF` (或者用 `io.ReadAll`)
3. 服务端读取 HTTP chunked: 和客户端读取类似
4. 服务端发送 HTTP chunked: 将 `http.ResponseWriter` 断言为 `http.Flusher`, 然后在每发送一部分数据后调用 `flusher.Flush()`, 这样就会自动触发 `chunked` encoding 从而发送出一个 chunk

示例如下:

```go
func TestHTTPSendReceiveChunk(t *testing.T) {
	// HTTP chunked example:
	//   1. Client sends chunks: Add "chunked" transfer encoding header, and use io.Reader as body.
	//   2. Client reads chunks: The Go/net/http automatically handles the chunked reading.
	//                           Users can simply read resp.Body in a loop until io.EOF.
	//   3. Server reads chunks: Similar to client reads chunks.
	//   4. Server sends chunks: Assert http.ResponseWriter as http.Flusher, call flusher.Flush() after
	//         writing a part of data, it will automatically trigger "chunked" encoding to send a chunk.

	// Start server.
	const (
		network = "tcp"
		address = "127.0.0.1:0"
	)
	ln, err := net.Listen(network, address)
	require.Nil(t, err)
	defer ln.Close()
	go http.Serve(ln, &chunkedServer{})

	// Start client.
	c := thttp.NewClientProxy(
		"trpc.app.server.Service_http",
		client.WithTarget("ip://"+ln.Addr().String()),
	)

	// Open and read file.
	fileDir, err := os.Getwd()
	require.Nil(t, err)
	fileName := "README.md"
	filePath := path.Join(fileDir, fileName)
	file, err := os.Open(filePath)
	require.Nil(t, err)
	defer file.Close()

	// 1. Client sends chunks.

	// Add request headers.
	header := http.Header{}
	header.Add("Content-Type", "text/plain")
	// Add chunked transfer encoding header.
	header.Add("Transfer-Encoding", "chunked")
	reqHead := &thttp.ClientReqHeader{
		Header:  header,
		ReqBody: file, // Stream send (for chunks).
	}

	// Enable manual body reading in order to
	// disable the framework's automatic body reading capability,
	// so that users can manually do their own client-side streaming reads.
	rspHead := &thttp.ClientRspHeader{
		ManualReadBody: true,
	}
	req := &codec.Body{}
	rsp := &codec.Body{}
	require.Nil(t,
		c.Post(context.Background(), "/", req, rsp,
			client.WithCurrentSerializationType(codec.SerializationTypeNoop),
			client.WithSerializationType(codec.SerializationTypeNoop),
			client.WithCurrentCompressType(codec.CompressTypeNoop),
			client.WithReqHead(reqHead),
			client.WithRspHead(rspHead),
		))
	require.Nil(t, rsp.Data)

	// 2. Client reads chunks.

	// Do stream reads directly from rspHead.Response.Body.
	body := rspHead.Response.Body
	defer body.Close() // Do remember to close the body.
	buf := make([]byte, 4096)
	var idx int
	for {
		n, err := body.Read(buf)
		if err == io.EOF {
			t.Logf("reached io.EOF\n")
			break
		}
		t.Logf("read chunk %d of length %d: %q\n", idx, n, buf[:n])
		idx++
	}
}

type chunkedServer struct{}

func (*chunkedServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 3. Server reads chunks.

	// io.ReadAll will read until io.EOF.
	// Go/net/http will automatically handle chunked body reads.
	bs, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("io.ReadAll err: %+v", err)))
		return
	}

	// 4. Server sends chunks.

	// Send HTTP chunks using http.Flusher.
	// Reference: https://stackoverflow.com/questions/26769626/send-a-chunked-http-response-from-a-go-server.
	// The "Transfer-Encoding" header will be handled by the writer implicitly, so no need to set it.
	flusher, ok := w.(http.Flusher)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("expected http.ResponseWriter to be an http.Flusher"))
		return
	}
	chunks := 10
	chunkSize := (len(bs) + chunks - 1) / chunks
	for i := 0; i < chunks; i++ {
		start := i * chunkSize
		end := (i + 1) * chunkSize
		if end > len(bs) {
			end = len(bs)
		}
		w.Write(bs[start:end])
		flusher.Flush() // Trigger "chunked" encoding and send a chunk.
		time.Sleep(500 * time.Millisecond)
	}
	return
}
```

