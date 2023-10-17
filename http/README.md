English | [中文](README.zh_CN.md)

# tRPC-Go HTTP protocol 

The tRPC-Go framework supports building three types of HTTP-related services:

1. pan-HTTP standard service (no stub code and IDL file required)
2. pan-HTTP RPC service (shares the stub code and IDL files used by the RPC protocol)
3. pan-HTTP RESTful service (provides RESTful API based on IDL and stub code)

The RESTful related documentation is available in [/restful](/restful/)

## Pan-HTTP standard services

The tRPC-Go framework provides pervasive HTTP standard service capabilities, mainly by adding service registration, service discovery, interceptors and other capabilities to the annotation library HTTP, so that the HTTP protocol can be seamlessly integrated into the tRPC ecosystem

Compared with the tRPC protocol, the pan-HTTP standard service service does not rely on stub code, so the protocol on the service side is named `http_no_protocol`.

### Server-side

#### configuration writing

Configure the service in the `trpc_go.yaml` configuration file with protocol `http_no_protocol` and http2 with `http2_no_protocol`:

```yaml
server:
    service: # The service provided by the business service, there can be more than one
      - name: trpc.app.server.stdhttp # The service's route name
        network: tcp # the type of network listening, tcp or udp
        protocol: http_no_protocol # Application layer protocol http_no_protocol
        timeout: 1000 # Maximum request processing time, in milliseconds
        ip: 127.0.0.1
        port: 8080 # Service listening port
```

Take care to ensure that the configuration file is loaded properly

#### code writing

##### single URL registration

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
    // The parameters passed when registering the NoProtocolService must match the service name in the configuration: s.Service("trpc.app.server.stdhttp")
    thttp.RegisterNoProtocolService(s.Service("trpc.app.server.stdhttp")) 
    s.Serve()
}

func handle(w http.ResponseWriter, r *http.Request) error {
    // handle is written in exactly the same way as the standard library HTTP
    // For example, you can read the header in r, etc.
    // You can stream the packet to the client in r.
    clientReq, err := io.ReadAll(r.Body)
    if err ! = nil { /*... */ }
    // Finally use w for packet return
    w.Header().Set("Content-type", "application/text")
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("response body"))
    return nil
}
```

##### MUX Registration

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
    // Routing registration
    router := mux.NewRouter()
    router.HandleFunc("/{dir0}/{dir1}/{day}/{hour}/{vid:[a-z0-9A-Z]+}_{index:[0-9]+}.jpg", handle).
        Methods("GET")
    // The parameters passed when registering RegisterNoProtocolServiceMux must be consistent with the service name in the configuration: s.Service("trpc.app.server.stdhttp")
    thttp.RegisterNoProtocolServiceMux(s.Service("trpc.app.server.stdhttp"), router)
    s.Serve()
}

func handle(w http.ResponseWriter, r *http.Request) error {
    // take the arguments in the url
    vars := mux.Vars(r)
    vid := vars["vid"]
    index := vars["index"]
    log.Infof("vid: %s, index: %s", vid, index)
    return nil
}
```

### Client

This refers to calling a standard HTTP service, which is not necessarily built on the tRPC-Go framework downstream

The cleanest way is actually to use the HTTP Client provided by the standard library directly, but you can't use the service discovery and various plug-in interceptors that provide capabilities (such as monitoring reporting)

#### configuration writing

```yaml
client: # backend configuration for client calls
  timeout: 1000 # Maximum processing time for all backend requests
  namespace: Development # environment for all backends
  filter: # List of interceptors before and after all backend function calls
    - simpledebuglog # This is the debug log interceptor, you can add other interceptors, such as monitoring, etc.
  service: # Configuration for a single backend
    - name: trpc.app.server.stdhttp # service name of the downstream http service 
    # # You can use target to select other selector, only service name will be used for service discovery by default (in case of using polaris plugin)
    # target: polaris://trpc.app.server.stdhttp # or ip://127.0.0.1:8080 to specify ip:port for invocation
```

#### code writing

```go
package main

import (
    "context"

    trpc "trpc.group/trpc-go/trpc-go"
    "trpc.group/trpc-go/trpc-go/client"
    "trpc.group/trpc-go/trpc-go/codec"
    "trpc.group/trpc-go/trpc-go/http"
    "trpc.group/trpc-go/trpc-go/log"
)

// Data is request message data.
type Data struct {
    Msg string
}

func main() {
    // Omit the tRPC-Go framework configuration loading part, if the following logic is in an RPC handle, 
    // the configuration has generally been loaded normally.
    // Create ClientProxy, set the protocol to HTTP protocol, and serialize it to JSON.
    httpCli := http.NewClientProxy("trpc.app.server.stdhttp",
        client.WithSerializationType(codec.SerializationTypeJSON))
    reqHeader := &http.ClientReqHeader{}
    // Add request field for HTTP Head.
    reqHeader.AddHeader("request", "test")
    rspHead := &http.ClientRspHeader{}
    req := &Data{Msg: "Hello, I am stdhttp client!"}
    rsp := &Data{}
    // Send HTTP POST request.
    if err := httpCli.Post(context.Background(), "/v1/hello", req, rsp,
        client.WithReqHead(reqHeader),
        client.WithRspHead(rspHead),
    ); err != nil {
        log.Warn("get http response err")
        return
    }
    // Get the reply field in the HTTP response header.
    replyHead := rspHead.Response.Header.Get("reply")
    log.Infof("data is %s, request head is %s\n", rsp, replyHead)
}
```

## Pan HTTP RPC Service

Compared to the **Pan HTTP Standard Service**, the main difference of the Pan HTTP RPC Service is the reuse of the IDL protocol file and its generated stub code, while seamlessly integrating into the tRPC ecosystem (service registration, service routing, service discovery, various plug-in interceptors, etc.)

Note: 

In this service form, the HTTP protocol is consistent with the tRPC protocol: when the server returns a failure, the body is empty and the error code error message is placed in the HTTP header

### Server-side

#### configuration writing

First you need to generate the stub code:

```shell
trpc create -p helloworld.proto --protocol http -o out
```

If you are already a tRPC service and want to support the HTTP protocol on the same interface, you don't need to generate the stakes again, just add the `http` protocol to the configuration

```yaml
server: # server-side configuration
  service:
    ## The same interface can provide both trpc protocol and http protocol services through two configurations
    - name: trpc.test.helloworld.Greeter # service's route name
      ip: 127.0.0.0 # service listener ip address can use placeholder ${ip},ip or nic, ip is preferred
      port: 80 # The service listens to the port.
      protocol: trpc # Application layer protocol trpc http
    ## Here is the main example, note that the application layer protocol is http
    - name: trpc.test.helloworld.GreeterHTTP # service's route name
      ip: 127.0.0.0 # service listener ip address can use placeholder ${ip},ip or nic, ip is preferred
      port: 80 # The service listens to the port.
      protocol: http # Application layer protocol trpc http
```

#### code writing

```go
import (
    "context"
    "fmt"

    "trpc.group/trpc-go/trpc-go"
    "trpc.group/trpc-go/trpc-go/client"
    pb "github.com/xxxx/helloworld/pb"
)

func main() {
    s := trpc.NewServer()
    hello := Hello{}
    pb.RegisterHelloTrpcGoService(s.Service("trpc.test.helloworld.Greeter"), &hello)
    // Same as the normal tRPC service registration
    pb.RegisterHelloTrpcGoService(s.Service("trpc.test.helloworld.GreeterHTTP"), &hello)
    log.Println(s.Serve())
}

type Hello struct {}

// The implementation of the RPC service interface does not need to be aware of the HTTP protocol, it just needs to follow the usual logic to process the request and return a response
func (h *Hello) Hello(ctx context.Context, req *pb.HelloReq) (*pb.HelloRsp, error) {
    fmt.Println("--- got HelloReq", req)
    time.Sleep(time.Second)
    return &pb.HelloRsp{Msg: "Welcome " + req.Name}, nil
}
```
#### Custom URL path

Default is `/package.service/method`, you can customize any URL by alias parameter

- Protocol definition.

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

#### Custom error code handling functions

The default error handling function, which populates the `trpc-ret/trpc-func-ret` field in the HTTP header, can also be replaced by defining your own ErrorHandler.

```golang
import (
    "net/http"

    "trpc.group/trpc-go/trpc-go/errs"
    thttp "trpc.group/trpc-go/trpc-go/http"
)

func init() {
    thttp.DefaultServerCodec.ErrHandler = func(w http.ResponseWriter, r *http.Request, e *errs.Error) {
        // Generally define your own retcode retmsg field, compose the json and write it to the response body
        w.Write([]byte(fmt.Sprintf(`{"retcode":%d, "retmsg":"%s"}`, e.Code, e.Msg)))
        // Each business team can define it in their own git, and the business code can be imported into it
    }
}
```


### Client

There is considerable flexibility in actually calling a pan-HTTP RPC service, as the service provides the HTTP protocol externally, so any HTTP Client can be called, in general, in one of three ways:

* using the standard library HTTP Client, which constructs the request and parses the response based on the interface documentation provided downstream, with the disadvantage that it does not fit into the tRPC ecosystem (service discovery, plug-in interceptors, etc.)
* `NewStdHTTPClient`, which constructs requests and parses responses based on downstream documentation, can be integrated into the tRPC ecosystem, but request responses require documentation to construct and parse.
* `NewClientProxy`, using `Get/Post/Put` interfaces on top of the returned `Client`, can be integrated into the tRPC ecosystem, and `req,rsp` strictly conforms to the definition in the IDL protocol file, can reuse the stub code, the disadvantage is the lack of flexibility of the standard library HTTP Client, For example, it is not possible to read back packets in a stream

`NewStdHTTPClient` is used in the **client** section of the **Pan HTTP Standard Service**, and the following describes the stub-based HTTP Client `thttp.NewClientProxy`.

#### configuration writing

It is written in the same way as a normal RPC Client, just change the configuration `protocol` to `http`:

```yaml
client:
  namespace: Development # for all backend environments
  filter: # List of interceptors for all backends before and after function calls
  service: # Configuration for a single backend
    - name: trpc.test.helloworld.GreeterHTTP # service name of the backend service
      network: tcp # The network type of the backend service tcp udp
      protocol: http # Application layer protocol trpc http
      # # You can use target to select other selector, only service name will be used by default for service discovery (if Polaris plugin is used)
      # target: ip://127.0.0.1:8000 # request service address
      timeout: 1000 # maximum request processing time
```

#### code writing

```go
// Package main is the main package.
package main
import (
    "context"
    "net/http"

    "trpc.group/trpc-go/trpc-go/client"
    thttp "trpc.group/trpc-go/trpc-go/http"
    "trpc.group/trpc-go/trpc-go/log"
    pb "trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"
)
func main() {
    // omit the configuration loading part of the tRPC-Go framework, if the following logic is in some RPC handle, the configuration is usually already loaded properly
    // Create a ClientProxy, set the protocol to HTTP, serialize it to JSON
    proxy := pb.NewGreeterClientProxy()
    reqHeader := &thttp.ClientReqHeader{}
    // must be left blank or set to "POST"
    reqHeader.Method = "POST"
    // Add request field to HTTP Head
    reqHeader.AddHeader("request", "test")
    // Set a cookie
    cookie := &http.Cookie{Name: "sample", Value: "sample", HttpOnly: false}
    reqHeader.AddHeader("Cookie", cookie.String())
    req := &pb.HelloRequest{Msg: "Hello, I am tRPC-Go client."}
    rspHead := &thttp.ClientRspHeader{}
    // Send HTTP RPC request
    rsp, err := proxy.SayHello(context.Background(), req,
        client.WithReqHead(reqHeader),
        client.WithRspHead(rspHead),
        // Here you can use the code to force the target field in trpc_go.yaml to be overridden to set other selectors, which is generally not necessary, this is just a demonstration of the functionality
        // client.WithTarget("ip://127.0.0.1:8000"),
    )
    if err != nil {
        log.Warn("get http response err")
        return
    }
    // Get the reply field in the HTTP response header
    replyHead := rspHead.Response.Header.Get("reply")
    log.Infof("data is %s, request head is %s\n", rsp, replyHead)
}
```

## FAQ

### Enable HTTPS for Client and Server

#### Mutual Authentication

##### Configuration Only

Simply add the corresponding configuration items (certificate and private key) in `trpc_go.yaml`:

```yaml
server:  # Server configuration
  service:  # Business services provided, can have multiple
    - name: trpc.app.server.stdhttp
      network: tcp
      protocol: http_no_protocol  # Fill in http for generic HTTP RPC services
      tls_cert: "../testdata/server.crt"  # Add certificate path
      tls_key: "../testdata/server.key"  # Add private key path
      ca_cert: "../testdata/ca.pem"  # CA certificate, fill in when mutual authentication is required
client:  # Client configuration
  service:  # Business services provided, can have multiple
    - name: trpc.app.server.stdhttp
      network: tcp
      protocol: http
      tls_cert: "../testdata/server.crt"  # Add certificate path
      tls_key: "../testdata/server.key"  # Add private key path
      ca_cert: "../testdata/ca.pem"  # CA certificate, fill in when mutual authentication is required
```

No additional TLS/HTTPS-related operations are needed in the code (no need to specify the scheme as `https`, no need to manually add the `WithTLS` option, and no need to find a way to include an HTTPS-related identifier in `WithTarget` or other places).

##### Code Only

For the server, use `server.WithTLS` to specify the server certificate, private key, and CA certificate in order:

```go
server.WithTLS(
	"../testdata/server.crt",
	"../testdata/server.key",
	"../testdata/ca.pem",
)
```

For the client, use `client.WithTLS` to specify the client certificate, private key, CA certificate, and server name in order:

```go
client.WithTLS(
	"../testdata/client.crt",
	"../testdata/client.key",
	"../testdata/ca.pem",
	"localhost",  // Fill in the server name
)
```

No additional TLS/HTTPS-related operations are needed in the code.

Example:

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

#### Client Certificate Not Authenticated

##### Configuration Only

Simply add the corresponding configuration items (certificate and private key) in `trpc_go.yaml`:

```yaml
server:  # Server configuration
  service:  # Business services provided, can have multiple
    - name: trpc.app.server.stdhttp
      network: tcp
      protocol: http_no_protocol  # Fill in http for generic HTTP RPC services
      tls_cert: "../testdata/server.crt"  # Add certificate path
      tls_key: "../testdata/server.key"  # Add private key path
      # ca_cert: ""  # CA certificate, leave empty when the client certificate is not authenticated
client:  # Client configuration
  service:  # Business services provided, can have multiple
    - name: trpc.app.server.stdhttp
      network: tcp
      protocol: http
      # tls_cert: ""  # Certificate path, leave empty when the client certificate is not authenticated
      # tls_key: ""  # Private key path, leave empty when the client certificate is not authenticated
      ca_cert: "none"  # CA certificate, fill in "none" when the client certificate is not authenticated
```

For the mutual authentication part, the main difference is that the server's `ca_cert` needs to be left empty and the client's `ca_cert` needs to be filled with "none".

No additional TLS/HTTPS-related operations are needed in the code (no need to specify the scheme as `https`, no need to manually add the `WithTLS` option, and no need to find a way to include an HTTPS-related identifier in `WithTarget` or other places).

##### Code Only

For the server, use `server.WithTLS` to specify the server certificate, private key, and leave the CA certificate empty:

```go
server.WithTLS(
	"../testdata/server.crt",
	"../testdata/server.key",
	"",  // Leave the CA certificate empty when the client certificate is not authenticated
)
```

For the client, use `client.WithTLS` to specify the client certificate, private key, and fill in "none" for the CA certificate:

```go
client.WithTLS(
	"",  // Leave the certificate path empty
	"",  // Leave the private key path empty
	"none",  // Fill in "none" for the CA certificate when the client certificate is not authenticated
	"",  // Leave the server name empty
)
```

No additional TLS/HTTPS-related operations are needed in the code.

Example: 

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


### Client uses `io.Reader` for streaming file upload

Requires trpc-go version >= v0.13.0

The key point is to assign an `io.Reader` to the `thttp.ClientReqHeader.ReqBody` field (`body` is an `io.Reader`):

```go
reqHeader := &thttp.ClientReqHeader{
	Header:  header,
	ReqBody: body, // Stream send.
}
```

Then specify `client.WithReqHead(reqHeader)` when making the call:

```go
c.Post(context.Background(), "/", req, rsp,
	client.WithCurrentSerializationType(codec.SerializationTypeNoop),
	client.WithSerializationType(codec.SerializationTypeNoop),
	client.WithCurrentCompressType(codec.CompressTypeNoop),
	client.WithReqHead(reqHeader),
)
```

Here's an example:

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
		Header: header,
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

### Reading Response Body Stream Using io.Reader in the Client

Requires trpc-go version >= v0.13.0

The key is to add `thttp.ClientRspHeader` and specify the `thttp.ClientRspHeader.ManualReadBody` field as `true`:

```go
rspHead := &thttp.ClientRspHeader{
	ManualReadBody: true,
}
```

Then, when making the call, add `client.WithRspHead(rspHead)`:

```go
c.Post(context.Background(), "/", req, rsp,
	client.WithCurrentSerializationType(codec.SerializationTypeNoop),
	client.WithSerializationType(codec.SerializationTypeNoop),
	client.WithCurrentCompressType(codec.CompressTypeNoop),
	client.WithRspHead(rspHead),
)
```

Finally, you can perform streaming reads on `rspHead.Response.Body`:

```go
body := rspHead.Response.Body // Do stream reads directly from rspHead.Response.Body.
defer body.Close()            // Do remember to close the body.
bs, err := io.ReadAll(body)
```

Here's an example:

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

### Client and Server Sending and Receiving HTTP Chunked

1. Client sends HTTP chunked:
   1. Add `chunked` Transfer-Encoding header.
   2. Use io.Reader to send the data.
2. Client receives HTTP chunked: The Go standard library's HTTP automatically supports handling chunked responses. The upper-level user is unaware of it and only needs to loop over reading from `resp.Body` until `io.EOF` (or use `io.ReadAll`).
3. Server reads HTTP chunked: Similar to client reading.
4. Server sends HTTP chunked: Assert `http.ResponseWriter` as `http.Flusher`, then call `flusher.Flush()` after sending a portion of the data. This will automatically trigger the `chunked` encoding and send a chunk.

Here is an example:

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
