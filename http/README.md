English | [中文](README.zh_CN.md)

- [tRPC-Go HTTP protocol](#trpc-go-http-protocol)
  - [Pan-HTTP standard services](#pan-http-standard-services)
    - [Server-side](#server-side)
      - [configuration writing](#configuration-writing)
      - [code writing](#code-writing)
        - [single URL registration](#single-url-registration)
        - [MUX Registration](#mux-registration)
    - [Client](#client)
      - [configuration writing](#configuration-writing-1)
      - [code writing](#code-writing-1)
  - [Pan HTTP RPC Service](#pan-http-rpc-service)
    - [Server-side](#server-side-1)
      - [configuration writing](#configuration-writing-2)
      - [code writing](#code-writing-2)
      - [Custom URL path](#custom-url-path)
      - [Custom error code handling functions](#custom-error-code-handling-functions)
    - [Client](#client-1)
      - [configuration writing](#configuration-writing-3)
      - [code writing](#code-writing-3)
  - [HTTP Connection Pool Configuration](#http-connection-pool-configuration)
    - [configuration writing](#configuration-writing-4)
    - [code writing](#code-writing-4)
  - [FAQ](#faq)
    - [Enable HTTPS for Client and Server](#enable-https-for-client-and-server)
      - [Mutual Authentication](#mutual-authentication)
        - [Configuration Only](#configuration-only)
        - [Code Only](#code-only)
      - [Client Certificate Not Authenticated](#client-certificate-not-authenticated)
        - [Configuration Only](#configuration-only-1)
        - [Code Only](#code-only-1)
    - [Client uses `io.Reader` for streaming file upload](#client-uses-ioreader-for-streaming-file-upload)
    - [Reading Response Body Stream Using io.Reader in the Client](#reading-response-body-stream-using-ioreader-in-the-client)
    - [Sending and Receiving SSE Content-Type](#sending-and-receiving-sse-content-type)
    - [Sending and Receiving SSE (Based on github.com/r3labs/sse)](#sending-and-receiving-sse-based-on-githubcomr3labssse)
    - [Sending and Receiving SSE (Based on github.com/r3labs/sse)](#sending-and-receiving-sse-based-on-githubcomr3labssse-1)
    - [Client-side Forwarding](#client-side-forwarding)
    - [Client and Server Sending and Receiving HTTP Chunked](#client-and-server-sending-and-receiving-http-chunked)
    - [Sending Data with Arbitrary Content-Type from the Client](#sending-data-with-arbitrary-content-type-from-the-client)
    - [Submitting Form Data from the Client](#submitting-form-data-from-the-client)
      - [Submitting Form data with Content-Type application/x-www-form-urlencoded](#submitting-form-data-with-content-type-applicationx-www-form-urlencoded)
      - [Submitting Form data with Content-Type multipart/form-data](#submitting-form-data-with-content-type-multipartform-data)
    - [Server-side File Upload (using `multipart/form-data`)](#server-side-file-upload-using-multipartform-data)
    - [Empty req and rsp reported when using HTTP standard services and clients](#empty-req-and-rsp-reported-when-using-http-standard-services-and-clients)
    - [Reasons for Receiving Empty Response Content](#reasons-for-receiving-empty-response-content)
    - [Restrict to Only Accept POST Method Requests](#restrict-to-only-accept-post-method-requests)
    - [Provide individual timeouts for each handler in the http\_no\_protocol service](#provide-individual-timeouts-for-each-handler-in-the-http_no_protocol-service)
    - [Customize the constructed http.Request of the framework (e.g., modify Content-Length)](#customize-the-constructed-httprequest-of-the-framework-eg-modify-content-length)
    - [Supporting both generic HTTP standard services and RESTful services simultaneously](#supporting-both-generic-http-standard-services-and-restful-services-simultaneously)
    - [Setting the behavior of GetSerialization for deserializing query parameters](#setting-the-behavior-of-getserialization-for-deserializing-query-parameters)
    - [About the Resource Leak Issue Caused by Value Detached Transport](#about-the-resource-leak-issue-caused-by-value-detached-transport)

# tRPC-Go HTTP protocol

The tRPC-Go framework supports building three types of HTTP-related services:

1. pan-HTTP standard service (no stub code and IDL file required)
2. pan-HTTP RPC service (shares the stub code and IDL files used by the RPC protocol)
3. pan-HTTP RESTful service (provides RESTful API based on IDL and stub code)

The RESTful related documentation is available in [../restful](../restful/)

## Pan-HTTP standard services

The tRPC-Go framework provides pervasive HTTP standard service capabilities, mainly by adding service registration, service discovery, filters and other capabilities to the annotation library HTTP, so that the HTTP protocol can be seamlessly integrated into the tRPC ecosystem

Compared with the tRPC protocol, the pan-HTTP standard service does not rely on stub code, so the protocol on the service side is named `http_no_protocol`.

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

    "git.code.oa.com/trpc-go/trpc-go/codec"
    "git.code.oa.com/trpc-go/trpc-go/log"
    thttp "git.code.oa.com/trpc-go/trpc-go/http"
    trpc "git.code.oa.com/trpc-go/trpc-go"
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

    "git.code.oa.com/trpc-go/trpc-go/codec"
    "git.code.oa.com/trpc-go/trpc-go/log"
    thttp "git.code.oa.com/trpc-go/trpc-go/http"
    trpc "git.code.oa.com/trpc-go/trpc-go"
    "github.com/gorilla/mux"
)

func main() {
    s := trpc.NewServer()
    // Routing registration
    router := mux.NewRouter()
    router.HandleFunc("/{dir0}/{dir1}/{day}/{hour}/{vid:[a-z0-9A-Z]+}_{index:[0-9]+}.jpg", handle).
        Methods(http.MethodGet)
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

The cleanest way is actually to use the HTTP Client provided by the standard library directly, but you can't use the service discovery and various plug-in filters that provide capabilities (such as monitoring reporting)

#### configuration writing

```yaml
client: # backend configuration for client calls
  timeout: 1000 # Maximum processing time for all backend requests
  namespace: Development # environment for all backends
  filter: # List of filters before and after all backend function calls
    - simpledebuglog # This is the debug log filter, you can add other filters, such as monitoring, etc.
  service: # Configuration for a single backend
    - name: trpc.app.server.stdhttp # service name of the downstream http service 
    # # You can use target to select other selector, only service name will be used for service discovery by default (in case of using polaris plugin)
    #   target: polaris://trpc.app.server.stdhttp # or ip://127.0.0.1:8080 to specify ip:port for invocation
    #   ca_cert: "none" # CA certificate, this field must be filled with "none" if client certificate authentication is not required
```

In the configuration section, please note that if you are accessing HTTPS, you need to add ca_cert: "none" (or specify a complete certificate file). For more details, please refer to [Enable HTTPS for Client and Server](#enable-https-for-client-and-server).

#### code writing

```go
package main

import (
    "context"

    trpc "git.code.oa.com/trpc-go/trpc-go"
    "git.code.oa.com/trpc-go/trpc-go/client"
    "git.code.oa.com/trpc-go/trpc-go/codec"
    "git.code.oa.com/trpc-go/trpc-go/http"
    "git.code.oa.com/trpc-go/trpc-go/log"
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
    reqHeader := &http.ClientReqHeader{
        // Note: When using a custom ClientReqHeader,
        // you need to explicitly specify the required HTTP method.
        Method: http.MethodPost,
    }
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

Compared to the **Pan HTTP Standard Service**, the main difference of the Pan HTTP RPC Service is the reuse of the IDL protocol file and its generated stub code, while seamlessly integrating into the tRPC ecosystem (service registration, service routing, service discovery, various plug-in filters, etc.)

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
      port: 8000 # The service listens to the port.
      protocol: trpc # Application layer protocol trpc http
    ## Here is the main example, note that the application layer protocol is http
    - name: trpc.test.helloworld.GreeterHTTP # service's route name
      ip: 127.0.0.0 # service listener ip address can use placeholder ${ip},ip or nic, ip is preferred
      port: 8001 # The service listens to the port. 
      protocol: http # Application layer protocol trpc http
```

#### code writing

```go
// Reference:
// https://git.woa.com/cooperyan/trpc-go-in-a-nutshell
import (
    "context"
    "fmt"

    "git.code.oa.com/trpc-go/trpc-go"
    "git.code.oa.com/trpc-go/trpc-go/client"
    pb "git.woa.com/xxxx/helloworld/pb"
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
option go_package="git.code.oa.com/trpcprotocol/app/server";

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

```go
import (
    "net/http"

    "git.code.oa.com/trpc-go/trpc-go/errs"
    thttp "git.code.oa.com/trpc-go/trpc-go/http"
)

func init() {
    thttp.DefaultServerCodec.ErrHandler = func(w http.ResponseWriter, r *http.Request, e *errs.Error) {
        // Generally define your own retcode retmsg field, compose the json and write it to the response body
        w.Write([]byte(fmt.Sprintf(`{"retcode": %d, "retmsg": "%s"}`, e.Code, e.Msg)))
        // Each business team can define it in their own git, and the business code can be imported into it
    }
}
```

### Client

There is considerable flexibility in actually calling a pan-HTTP RPC service, as the service provides the HTTP protocol externally, so any HTTP Client can be called, in general, in one of three ways:

- using the standard library HTTP Client, which constructs the request and parses the response based on the interface documentation provided downstream, with the disadvantage that it does not fit into the tRPC ecosystem (service discovery, plug-in filters, etc.)
- `NewStdHTTPClient`, which constructs requests and parses responses based on downstream documentation, can be integrated into the tRPC ecosystem, but request responses require documentation to construct and parse.
- `NewClientProxy`, using `Get/Post/Put` interfaces on top of the returned `Client`, can be integrated into the tRPC ecosystem, and `req,rsp` strictly conforms to the definition in the IDL protocol file, can reuse the stub code, the disadvantage is the lack of flexibility of the standard library HTTP Client, For example, it is not possible to read back packets in a stream

`NewStdHTTPClient` is used in the **client** section of the **Pan HTTP Standard Service**, and the following describes the stub-based HTTP Client `thttp.NewClientProxy`.

#### configuration writing

It is written in the same way as a normal RPC Client, just change the configuration `protocol` to `http`:

```yaml
client:
  namespace: Development # for all backend environments
  filter: # List of filters for all backends before and after function calls
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

    "git.code.oa.com/trpc-go/trpc-go"
    "git.code.oa.com/trpc-go/trpc-go/client"
    thttp "git.code.oa.com/trpc-go/trpc-go/http"
    "git.code.oa.com/trpc-go/trpc-go/log"
    pb "git.code.oa.com/trpcprotocol/test/rpchttp"
)
func main() {
    // omit the configuration loading part of the tRPC-Go framework, if the following logic is in some RPC handle, the configuration is usually already loaded properly
    // Create a ClientProxy, set the protocol to HTTP, serialize it to JSON
    proxy := pb.NewHelloClientProxy()
    reqHeader := &thttp.ClientReqHeader{}
    // must be left blank or set to "POST"
    reqHeader.Method = http.MethodPost
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

## HTTP Connection Pool Configuration

`HTTP Transport` allows connection pooling parameters to be set via configuration files or code.

### configuration writing

Set http connection pooling parameters through the configuration file.

```yaml
client:
  service:
    - name: trpc.test.helloworld.GreeterHTTP
      protocol: http
      conn_type: httppool  # connection type is httppool, the following options are all for httppool.
      httppool:
        max_idle_conns: 100  # httppool: max number of idle connections, default 0 (means no limit).
        max_idle_conns_per_host: 10  # httppool: max number of idle connections per-host, default 2.
        max_conns_per_host: 20  # httppool: max number of connections, default 0 (means no limit).
        idle_conn_timeout: 1s  # httppool: idle timeout, default 0s (means no limit).
```

### code writing

Set `transport.HTTPRoundTripOptions` via `client.WithHTTPRoundTripOptions` to configure parameters related to HTTP connection pooling.

```go
httpOpts := transport.HTTPRoundTripOptions{
    Pool: httppool.Options{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        MaxConnsPerHost:     20,
        IdleConnTimeout:     time.Second,
    },
}
proxy := pb.NewGreeterClientProxy(
    client.WithTarget("ip://127.0.0.1:8000"),
    client.WithProtocol("http"),
    client.WithHTTPRoundTripOptions(httpOpts),
)
```

## FAQ

### Enable HTTPS for Client and Server

There are two types of authentication: mutual authentication and one-way authentication. When using the framework, most often one-way authentication is used. To access an existing HTTPS service using trpc-go, you can construct an HTTPS client and perform one-way authentication.

#### Mutual Authentication

##### Configuration Only

Simply add the corresponding configuration items (certificate and private key) in `trpc_go.yaml`:

```yaml
server:  # Server configuration
  service:  # Business services provided, can have multiple
    - name: trpc.app.server.stdhttp
      network: tcp
      protocol: http_no_protocol  # Fill in http for generic HTTP RPC services (Starting from v0.16.0, this field can be filled with "https_no_protocol" or "https")
      tls_cert: "../testdata/server.crt"  # Add certificate path
      tls_key: "../testdata/server.key"  # Add private key path
      ca_cert: "../testdata/ca.pem"  # CA certificate, fill in when mutual authentication is required
client:  # Client configuration
  service:  # Business services provided, can have multiple
    - name: trpc.app.server.stdhttp
      network: tcp
      protocol: http # Starting from v0.16.0, this field can be filled with "https"
      # 1. Certificates/Private Keys/CA
      tls_cert: "../testdata/server.crt"  # Add certificate path
      tls_key: "../testdata/server.key"  # Add private key path
      ca_cert: "../testdata/ca.pem"  # CA certificate, fill in when mutual authentication is required
      # 2. Add the domain name "https://some-example.com" to "dns://some-example.com" as the target
      #    When accessing the ip:port directly, you can simply write the target as ip://x.x.x.x:xx
      target: dns://some-example.com  # Corresponds to curl "https://some-example.com"
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
// 1. Certificates/Private Keys/CA
client.WithTLS(
    "../testdata/client.crt",
    "../testdata/client.key",
    "../testdata/ca.pem",
    "localhost",  // Fill in the server name
)
// 2. Add the domain name "https://some-example.com" to "dns://some-example.com" as the target
//    When accessing the ip:port directly, you can simply write the target as ip://x.x.x.x:xx
client.WithTarget("ip://x.x.x.x:xx")
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
        server.WithProtocol("http_no_protocol"), // Starting from v0.16.0, this field can be filled with "https_no_protocol"
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
      protocol: http_no_protocol  # Fill in http for generic HTTP RPC services (Starting from v0.16.0, this field can be filled with "https_no_protocol" or "https")
      tls_cert: "../testdata/server.crt"  # Add certificate path
      tls_key: "../testdata/server.key"  # Add private key path
      # ca_cert: ""  # CA certificate, leave empty when the client certificate is not authenticated
client:  # Client configuration
  service:  # Business services provided, can have multiple
    - name: trpc.app.server.stdhttp
      network: tcp
      protocol: http  # Starting from v0.16.0, this field can be filled with "https" and no need to set ca_cert to "none" to enable HTTPS
      # 1. Certificates/Private Keys/CA
      # tls_cert: ""  # Certificate path, leave empty when the client certificate is not authenticated
      # tls_key: ""  # Private key path, leave empty when the client certificate is not authenticated
      ca_cert: "none"  # CA certificate, fill in "none" when the client certificate is not authenticated
      # 2. Add the domain name "https://some-example.com" to "dns://some-example.com" as the target
      #    When accessing the ip:port directly, you can simply write the target as ip://x.x.x.x:xx
      target: dns://some-example.com  # Corresponds to curl "https://some-example.com"
```

For the mutual authentication part, the main difference is that the server's `ca_cert` needs to be left empty and the client's `ca_cert` needs to be filled with "none".

No additional TLS/HTTPS-related operations are needed in the code (no need to specify the scheme as `https`, no need to manually add the `WithTLS` option, and no need to find a way to include an HTTPS-related identifier in `WithTarget` or other places).

**Note**: Starting from v0.16.0, users can directly fill in the `protocol` field with `https` to enable HTTPS, without the need to specify `ca_cert` or any other options. (Refer to <https://mk.woa.com/note/7509> )

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
// 1. Certificates/Private Keys/CA
client.WithTLS(
    "",  // Leave the certificate path empty
    "",  // Leave the private key path empty
    "none",  // Fill in "none" for the CA certificate when the client certificate is not authenticated
    "",  // Leave the server name empty
)
// 2. Add the domain name "https://some-example.com" to "dns://some-example.com" as the target
//    When accessing the ip:port directly, you can simply write the target as ip://x.x.x.x:xx
client.WithTarget("ip://x.x.x.x:xx")
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
    Method:  http.MethodPost,
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
        Method:  http.MethodPost,
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

### Sending and Receiving SSE Content-Type

Server-Sent Events (SSE) is a technology that establishes one-way communication between the server and the client, allowing the server to push real-time updates to the client. There are two key points to implementing SSE:

- **Setting Content-Type and related headers on both the server and client**
  - Set `Content-Type` to `text/event-stream` and ensure the response is streamed.

- **Adhering to the [SSE format](https://html.spec.whatwg.org/multipage/server-sent-events.html#server-sent-events) for communication on both the server and client**
  - Server
    - It is necessary to send events in the SSE format and flush them to the client in a timely manner.
    - For versions >= v0.19.0, `thttp` provides a `WriteSSE` function that allows you to quickly write `sse.Event` structures to an `io.Writer` in the SSE format. This eliminates the need for users to worry about the SSE data format.
    - For versions < v0.19.0, you need to **manually construct the response body** and then write it to the `http.ResponseWriter`.
  - Client
    - For versions >= v0.17.0, **`thttp.ClientRspHeader` provides a field named `SSEHandler` for registering a callback to receive SSE data**.
    - For versions < v0.17.0, **manual parsing is required, using `io.Reader` to stream and read the response** (see the previous section).

Below is a complete SSE test example, including both server and client implementations. For more detailed examples, you can refer to the [SSE normal example](https://git.woa.com/trpc-go/trpc-go/tree/master/examples/features/sse/normal).

```go
func TestHTTPSendAndReceiveSSE(t *testing.T) {
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
        server.WithNetwork(network),
        server.WithProtocol("http_no_protocol"),
        server.WithListener(ln),
    )
    pattern := "/" + t.Name()
    thttp.RegisterNoProtocolServiceMux(service, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        flusher, ok := w.(http.Flusher)
        if !ok {
            http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
            return
        }
        w.Header().Set("Content-Type", "text/event-stream")
        w.Header().Set("Cache-Control", "no-cache")
        w.Header().Set(thttp.Connection, "keep-alive")
        w.Header().Set("Access-Control-Allow-Origin", "*")
        bs, err := io.ReadAll(r.Body)
        if err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
        msg := string(bs)
        for i := 0; i < 3; i++ {
            e := sse.Event{Event: []byte("message"), Data: []byte(msg + strconv.Itoa(i))}
            if err := thttp.WriteSSE(w, e); err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
            }
            flusher.Flush()
            time.Sleep(500 * time.Millisecond)
        }
        return
    }))
    s := &server.Server{}
    s.AddService(serviceName, service)
    go s.Serve()
    defer s.Close(nil)
    time.Sleep(100 * time.Millisecond)

    c := thttp.NewClientProxy(
        serviceName,
        client.WithTarget("ip://"+ln.Addr().String()),
    )
    t.Run("automatically", func(t *testing.T) {
        reqHeader := &thttp.ClientReqHeader{
            Method: http.MethodPost,
        }
        var data []byte
        rspHead := &thttp.ClientRspHeader{
            ManualReadBody: false,
            SSEHandler: sseHandler(func(e *sse.Event) error {
                t.Logf("Receive sse event: %s, data: %s", e.Event, e.Data)
                if string(e.Event) == "message" {
                    data = append(data, e.Data...)
                }
                return nil
            }),
        }
        req := &codec.Body{Data: []byte("hello")}
        rsp := &codec.Body{}
        require.Nil(t,
            c.Post(context.Background(), pattern, req, rsp,
                client.WithCurrentSerializationType(codec.SerializationTypeNoop),
                client.WithSerializationType(codec.SerializationTypeNoop),
                client.WithCurrentCompressType(codec.CompressTypeNoop),
                client.WithReqHead(reqHeader),
                client.WithRspHead(rspHead),
                client.WithTimeout(time.Minute),
            ))
        require.Equal(t, "hello0hello1hello2", string(data))
    })

    t.Run("manually", func(t *testing.T) {
        reqHeader := &thttp.ClientReqHeader{
            Method: http.MethodPost,
        }
        rspHead := &thttp.ClientRspHeader{
            ManualReadBody: true,
        }
        req := &codec.Body{Data: []byte("hello")}
        rsp := &codec.Body{}
        require.Nil(t,
            c.Post(context.Background(), pattern, req, rsp,
                client.WithCurrentSerializationType(codec.SerializationTypeNoop),
                client.WithSerializationType(codec.SerializationTypeNoop),
                client.WithCurrentCompressType(codec.CompressTypeNoop),
                client.WithReqHead(reqHeader),
                client.WithRspHead(rspHead),
                client.WithTimeout(time.Minute),
            ))

        body := rspHead.Response.Body // Do stream reads directly from rspHead.Response.Body.
        defer body.Close()            // Do remember to close the body.
        // Note that the following code disobeys the SSE protocol, which is simply splitting the lines with '\n'
        // and discarding the "data:" prefix. Since the manual process is too troublesome, we do not recommend this.
        buf := make([]byte, 1024)
        var data strings.Builder
        for {
            n, err := body.Read(buf)
            if err == io.EOF {
                break
            }
            require.Nil(t, err)
            lines := bytes.Split(buf[:n], []byte("\n"))
            for _, line := range lines {
                if !bytes.HasPrefix(line, []byte("data:")) {
                    continue
                }
                fromIndex := len("data:")
                if line[fromIndex] == ' ' {
                    fromIndex++ // Ignore the optional space after the data: prefix.
                }
                data.Write(line[fromIndex:])
            }
        }

        require.Equal(t, "hello0hello1hello2", data.String())
    })
}
```

For APIs that may return SSE or non-SSE responses, the client provides the following fields:

- In versions >= v0.19.0, **`thttp.ClientRspHeader` provides `SSECondition` and `ResponseHandler` fields to adopt different callback strategies based on the server's response**.
  - `SSECondition`: If **`SSECondition` returns `true` and the user has implemented `SSEHandler`**, the `SSEHandler` callback is invoked. Users can implement this interface themselves and can check if the response header contains `Content-Type: text/event-stream`,
  but please note that **not all services strictly adhere to this rule**. If this field is left empty, the framework will use the default implementation (returns `true`).
  - `ResponseHandler`: If **`SSECondition` returns `false` or the user has not implemented `SSEHandler`**, the `ResponseHandler` callback is invoked. If the user has not implemented this interface, the framework's fallback strategy is to automatically read the response body.

- In versions < v0.19.0, **manual parsing operations are required to distinguish whether the response is an SSE message, and then use `io.Reader` to adopt different strategies for streaming the response body** (see the previous section).

Please note that **both `SSEHandler` and `ResponseHandler` will only take effect when `ManualReadBody` is set to `false`**.

Below is a complete SSE test example, including both server and client implementations. For more detailed examples, you can refer to the [SSE multiple example](https://git.woa.com/trpc-go/trpc-go/tree/master/examples/features/sse/multiple).

```go
func TestHTTPSendAndReceiveSSEAndNormalResponse(t *testing.T) {
    ln, err := net.Listen(network, address)
    require.Nil(t, err)
    defer ln.Close()
    serviceName := "trpc.app.server.Service" + t.Name()
    service := server.New(
        server.WithServiceName(serviceName),
        server.WithNetwork(network),
        server.WithProtocol("http_no_protocol"),
        server.WithListener(ln),
    )
    pattern := "/" + t.Name()
    isSSE := true // Whether to send an SSE event, the first time is true.
    thttp.RegisterNoProtocolServiceMux(service, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Switch between SSE and normal response.
        defer func() { isSSE = !isSSE }()
        if isSSE {
            sseHandlerFunc(w, r)
            return
        }
        normalHandlerFunc(w, r)
    }))

    s := &server.Server{}
    s.AddService(serviceName, service)
    go s.Serve()
    defer s.Close(nil)
    time.Sleep(100 * time.Millisecond)

    c := thttp.NewClientProxy(
        serviceName,
        client.WithTarget("ip://"+ln.Addr().String()),
    )

    reqHeader := &thttp.ClientReqHeader{
        Method: http.MethodPost,
    }

    var data []byte
    rspHead := &thttp.ClientRspHeader{
        ManualReadBody: false,
        SSECondition: func(r *http.Response) bool {
            return r.Header.Get("Content-Type") == "text/event-stream"
        },
        ResponseHandler: rspHandler(func(r *http.Response) error {
            bs, err := io.ReadAll(r.Body)
            if err != nil {
                return err
            }
            t.Logf("Receive http response: %s", string(bs))
            data = append(data, bs...)
            return nil
        }),
        SSEHandler: sseHandler(func(e *sse.Event) error {
            t.Logf("Receive sse event: %s, data: %s", e.Event, e.Data)
            if string(e.Event) == "message" {
                data = append(data, e.Data...)
            }
            return nil
        }),
    }

    req := &codec.Body{Data: []byte("hello")}
    rsp := &codec.Body{}
    // The first time we send a request, the response is an SSE event, and the second is a normal response.
    // It is to say, the handler will switch between SSE and normal response, but the response data are the same.
    for i := 0; i < 4; i++ {
        t.Run(fmt.Sprintf("request "+strconv.Itoa(i)), func(t *testing.T) {
            data = []byte{} // Clear the data.
            require.Nil(t,
                c.Post(context.Background(), pattern, req, rsp,
                    client.WithCurrentSerializationType(codec.SerializationTypeNoop),
                    client.WithSerializationType(codec.SerializationTypeNoop),
                    client.WithCurrentCompressType(codec.CompressTypeNoop),
                    client.WithReqHead(reqHeader),
                    client.WithRspHead(rspHead),
                    client.WithTimeout(time.Minute),
                ))
            require.Equal(t, "hello0hello1hello2", string(data))
        })
    }
}

// sseHandler is a handler that handles sse events.
// It sends responses with the header of "Content-Type: text/event-stream".
func sseHandlerFunc(w http.ResponseWriter, r *http.Request) {
    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set(thttp.Connection, "keep-alive")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    bs, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    msg := string(bs)
    // Send sse message.
    for i := 0; i < 3; i++ {
        e := sse.Event{Event: []byte("message"), Data: []byte(msg + strconv.Itoa(i))}
        if err := thttp.WriteSSE(w, e); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        flusher.Flush()
        time.Sleep(500 * time.Millisecond)
    }
}

// normalHandler is a handler that handles normal responses.
// It sends responses with the header of "Content-Type: text/plain".
func normalHandlerFunc(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/plain")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set(thttp.Connection, "keep-alive")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    bs, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    msg := string(bs)
    var data []byte
    for i := 0; i < 3; i++ {
        data = append(data, []byte(msg+strconv.Itoa(i))...)
    }
    _, _ = w.Write(data)
}

type sseHandler func(*sse.Event) error

// Handle handles sse event, if the returned error is non-nil,
// the framework will abort the reading of the HTTP connection.
func (h sseHandler) Handle(e *sse.Event) error {
    return h(e)
}

type rspHandler func(*http.Response) error

// Handle handles common HTTP response.
func (h rspHandler) Handle(r *http.Response) error {
    return h(r)
}
```

### Sending and Receiving SSE (Based on github.com/r3labs/sse)

For more complex SSE handling, you might consider using the third-party library [r3labs/sse](https://github.com/r3labs/sse).

> Note: [r3labs/sse](https://github.com/r3labs/sse) uses `sse.Client` instead of the standard library's `http.Client`, and it only supports `http.MethodGet` requests with limited customization options.
> If you need more customization, you can extract the client implementation logic from [r3labs/sse](https://github.com/r3labs/sse) and combine it with the client-side SSE handling approach mentioned in the previous section.
> However, this method might have some impact on client-side forwarding, so it is **currently not recommended** to handle SSE this way.

Below is a complete SSE test example based on r3labs/sse, including both server and client implementations. For more detailed examples, you can refer to the
[SSE r3labs example](https://git.woa.com/trpc-go/trpc-go/tree/master/examples/features/sse/r3labs) and [r3labs/sse/http_test.go](https://github.com/r3labs/sse/blob/v2.10.0/http_test.go).

```go
func TestHTTPSendAndReceiveSSEWithR3Lab(t *testing.T) {
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
        server.WithNetwork(network),
        server.WithProtocol("http_no_protocol"),
        server.WithListener(ln),
    )

    pattern := "/" + t.Name()

    svr := sse.New()
    mux := http.NewServeMux()
    mux.Handle(pattern, svr)
    thttp.RegisterNoProtocolServiceMux(service, mux)
    svr.CreateStream("test")

    for i := 0; i < 3; i++ {
        event := &sse.Event{
            ID:    []byte(fmt.Sprintf("%d", i)),
            Event: []byte("message"),
            Data:  []byte(fmt.Sprintf("This is message %d", i)),
        }
        svr.Publish("test", event)
    }

    s := &server.Server{}
    s.AddService(serviceName, service)
    go s.Serve()
    defer s.Close(nil)
    time.Sleep(100 * time.Millisecond)

    c := sse.NewClient(fmt.Sprintf("http://%s%s", ln.Addr().String(), pattern))

    events := make(chan *sse.Event)
    go func() {
        err = c.Subscribe("test", func(msg *sse.Event) {
            if len(msg.Data) > 0 {
                events <- msg
            }
        })
    }()

    // Wait for the subscription to succeed.
    time.Sleep(200 * time.Millisecond)
    require.Nil(t, err)

    for i := 0; i < 3; i++ {
        msg, err := wait(events, 500*time.Millisecond)
        require.Nil(t, err)
        require.Equal(t, []byte(fmt.Sprintf("This is message %d", i)), msg)
    }
}

// wait waits for the sse event and read data into msg. If timeout, return error.
func wait(ch chan *sse.Event, duration time.Duration) ([]byte, error) {
    var err error
    var msg []byte

    select {
    case event := <-ch:
        msg = event.Data
    case <-time.After(duration):
        err = errors.New("timeout")
    }
    return msg, err
}
```

### Sending and Receiving SSE (Based on github.com/r3labs/sse)

For more complex SSE handling, you might consider using the third-party library [r3labs/sse](https://github.com/r3labs/sse).

> Note: [r3labs/sse](https://github.com/r3labs/sse) uses `sse.Client` instead of the standard library's `http.Client`, and it only supports `http.MethodGet` requests with limited customization options.
> If you need more customization, you can extract the client implementation logic from [r3labs/sse](https://github.com/r3labs/sse) and combine it with the client-side SSE handling approach mentioned in the previous section.
> However, this method might have some impact on client-side forwarding, so it is **currently not recommended** to handle SSE this way.

Below is a complete SSE test example based on r3labs/sse, including both server and client implementations. For more detailed examples, you can refer to the
[SSE r3labs example](https://git.woa.com/trpc-go/trpc-go/tree/master/examples/features/sse/r3labs) and [r3labs/sse/http_test.go](https://github.com/r3labs/sse/blob/v2.10.0/http_test.go).

```go
func TestHTTPSendAndReceiveSSEWithR3Lab(t *testing.T) {
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
        server.WithNetwork(network),
        server.WithProtocol("http_no_protocol"),
        server.WithListener(ln),
    )

    pattern := "/" + t.Name()

    svr := sse.New()
    mux := http.NewServeMux()
    mux.Handle(pattern, svr)
    thttp.RegisterNoProtocolServiceMux(service, mux)
    svr.CreateStream("test")

    for i := 0; i < 3; i++ {
        event := &sse.Event{
            ID:    []byte(fmt.Sprintf("%d", i)),
            Event: []byte("message"),
            Data:  []byte(fmt.Sprintf("This is message %d", i)),
        }
        svr.Publish("test", event)
    }

    s := &server.Server{}
    s.AddService(serviceName, service)
    go s.Serve()
    defer s.Close(nil)
    time.Sleep(100 * time.Millisecond)

    c := sse.NewClient(fmt.Sprintf("http://%s%s", ln.Addr().String(), pattern))

    events := make(chan *sse.Event)
    go func() {
        err = c.Subscribe("test", func(msg *sse.Event) {
            if len(msg.Data) > 0 {
                events <- msg
            }
        })
    }()

    // Wait for the subscription to succeed.
    time.Sleep(200 * time.Millisecond)
    require.Nil(t, err)

    for i := 0; i < 3; i++ {
        msg, err := wait(events, 500*time.Millisecond)
        require.Nil(t, err)
        require.Equal(t, []byte(fmt.Sprintf("This is message %d", i)), msg)
    }
}

// wait waits for the sse event and read data into msg. If timeout, return error.
func wait(ch chan *sse.Event, duration time.Duration) ([]byte, error) {
    var err error
    var msg []byte

    select {
    case event := <-ch:
        msg = event.Data
    case <-time.After(duration):
        err = errors.New("timeout")
    }
    return msg, err
}
```

### Client-side Forwarding

Scenario: The client requests the server and forwards the server's response to another service.

In some cases, the specific form of the server's response is unknown, so the client cannot construct a response structure in advance for deserialization.

In such cases, you can use `client.WithCurrentSerializationType(codec.SerializationTypeNoop)` to specify a serialization/deserialization method as a no-op, allowing direct manipulation of raw data.

Here is an example:

```go
func TestHTTPProxy(t *testing.T) {
    // Start server.
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
        server.WithNetwork(network),
        server.WithProtocol("http_no_protocol"),
        server.WithListener(ln),
    )
    thttp.RegisterNoProtocolServiceMux(service, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        bs, err := io.ReadAll(r.Body)
        if err != nil {
            w.WriteHeader(http.StatusBadRequest)
            return
        }
        w.Header().Add("Content-Type", "application/json")
        w.Write(bs)
        return
    }))
    s := &server.Server{}
    s.AddService(serviceName, service)
    go s.Serve()
    defer s.Close(nil)
    time.Sleep(100 * time.Millisecond)

    // Start client.
    c := thttp.NewClientProxy(
        "trpc.app.server.Service_http",
        client.WithTarget("ip://"+ln.Addr().String()),
    )
    type request struct {
        Message string `json:"message"`
    }
    data := "hello"
    bs, err := json.Marshal(&request{Message: data})
    require.Nil(t, err)
    req := &codec.Body{Data: bs}
    rsp := &codec.Body{}
    require.Nil(t,
        c.Post(context.Background(), "/", req, rsp,
            client.WithCurrentSerializationType(codec.SerializationTypeNoop),
            client.WithSerializationType(codec.SerializationTypeJSON),
        ))
    require.Equal(t, bs, rsp.Data)
}
```

Additionally, this example can be combined with streaming to read the response packet, as follows:

```go
    // Enable manual body reading in order to
    // disable the framework's automatic body reading capability,
    // so that users can manually do their own client-side streaming reads.
    rspHead := &thttp.ClientRspHeader{
        ManualReadBody: true,
    }
    req = &codec.Body{Data: bs}
    rsp = &codec.Body{}
    require.Nil(t,
        c.Post(context.Background(), "/", req, rsp,
            client.WithCurrentSerializationType(codec.SerializationTypeNoop),
            client.WithSerializationType(codec.SerializationTypeNoop),
            client.WithRspHead(rspHead),
        ))
    require.Nil(t, rsp.Data)
    body := rspHead.Response.Body // Do stream reads directly from rspHead.Response.Body.
    defer body.Close()            // Do remember to close the body.
    result, err := io.ReadAll(body)
    require.Nil(t, err)
    require.Equal(t, bs, result)
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
        Method:  http.MethodPost,
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

### Sending Data with Arbitrary Content-Type from the Client

Two steps:

- For requests and responses, use the `*codec.Body` type. Put the expected request body (after processing it in the desired serialization format) into `(*code.Body).Data`.
- Specify the required `Content-Type` using `ClientReqHeader` and pass in two options (1. Provide reqHead, 2. Specify noop serialization):

```go
reqHead := &thttp.ClientReqHeader{} 
reqHead.AddHeader("Content-Type", "application/soap+xml; charset=utf-8")
c.Post(.., 
    client.WithReqHead(reqHead),
    client.WithCurrentSerializationType(codec.SerializationTypeNoop))
```

```go
func TestHTTPArbitraryContentType(t *testing.T) {
    c := thttp.NewClientProxy(
        "trpc.app.server.Service_http",
        client.WithTarget("ip://127.0.0.1:80"),
    )
    req := &codec.Body{
        Data: []byte(`<?xml version="1.0" encoding="utf-8"?>` +
        `<soap12:Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" ` +
        `xmlns:xsd="http://www.w3.org/2001/XMLSchema" ` +
        `xmlns:soap12="http://www.w3.org/2003/05/soap-envelope">` +
        `<soap12:Body>` +
        `<GetActivityInfo xmlns="http://tempuri.org/">` +
        `<ActivityID>id</ActivityID>` +
        `</GetActivityInfo>` +
        `</soap12:Body>` +
        `</soap12:Envelope>`),
    }
    reqHead := &thttp.ClientReqHeader{}
    reqHead.AddHeader("Content-Type", "application/soap+xml; charset=utf-8")
    rsp := &codec.Body{}
    require.Nil(t,
        c.Post(context.Background(), "/", req, rsp,
            client.WithReqHead(reqHead),
            client.WithCurrentSerializationType(codec.SerializationTypeNoop),
        ))
    require.NotNil(t, rsp.Data)
    t.Logf("receive: %q\n", rsp.Data)
}
```

### Submitting Form Data from the Client

#### Submitting Form data with Content-Type application/x-www-form-urlencoded

Specify `client.WithSerializationType(codec.SerializationTypeForm)` and pass a request of type `url.Values`.

When reading the response, you can add `thttp.ClientRspHeader` and set the `thttp.ClientRspHeader.ManualReadBody` field to `true` to read the response using `io.Reader` for streaming (requires trpc-go version >= v0.13.0).

Alternatively, you can define a response struct in advance to avoid using the `ManualReadBody` feature in higher versions.

```go
func TestHTTPSendFormData(t *testing.T) {
    // Start server.
    const (
        network = "tcp"
        address = "127.0.0.1:0"
    )
    ln, err := net.Listen(network, address)
    require.Nil(t, err)
    defer ln.Close()
    type response struct {
        Message string `json:"message"`
    }
    s := http.Server{
        Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            bs, err := io.ReadAll(r.Body)
            if err != nil {
                w.WriteHeader(http.StatusBadRequest)
                return
            }
            t.Logf("server read: %q\n", bs)
            rsp := &response{Message: string(bs)}
            bs, err = json.Marshal(rsp)
            if err != nil {
                w.WriteHeader(http.StatusInternalServerError)
                return
            }
            w.Header().Add("Content-Type", "application/json")
            w.WriteHeader(http.StatusOK)
            w.Write(bs)
        }),
    }
    go s.Serve(ln)

    // Start client.
    c := thttp.NewClientProxy(
        "trpc.app.server.Service_http",
        client.WithTarget("ip://"+ln.Addr().String()),
    )
    req := make(url.Values)
    req.Add("key", "value")

    // Option 1: Use manual read to read response (requires trpc-go >= v0.13.0) 
    // (If you are using an older version of trpc-go, please refer to Option 2 below.)
    rspHead := &thttp.ClientRspHeader{
        ManualReadBody: true, // Requires trpc-go >= v0.13.0.
    }
    rsp := &codec.Body{}
    require.Nil(t,
        c.Post(context.Background(), "/", req, rsp,
            client.WithSerializationType(codec.SerializationTypeForm),
            client.WithRspHead(rspHead),
        ))
    require.Nil(t, rsp.Data)
    body := rspHead.Response.Body // Do stream reads directly from rspHead.Response.Body.
    defer body.Close()            // Do remember to close the body.
    bs, err := io.ReadAll(body)
    require.Nil(t, err)
    require.NotNil(t, bs)

    // Option 2: Predefine the response struct to avoid manual read.
    rsp1 := &response{}
    require.Nil(t,
        c.Post(context.Background(), "/", req, rsp1,
            client.WithSerializationType(codec.SerializationTypeForm),
        ))
    require.NotNil(t, rsp1.Message)
    t.Logf("receive: %s\n", rsp1.Message)
}
```

Note: Data sent in the above format will be URL encoded (such as [Percent-encoding](https://en.wikipedia.org/wiki/Percent-encoding)). If you do not want this to happen, you can use `codec.SerializationTypeNoop`. In this case, make sure both the request and response are of type `*codec.Body`.

```go
func TestHTTPSendFormData2(t *testing.T) {
    c := thttp.NewClientProxy(
        "trpc.app.server.Service_http",
        client.WithTarget("ip://127.0.0.1:43221"),
    )
    req := &codec.Body{
        Data: []byte(`data='{"cycle":10}'`),
    }
    rsp := &codec.Body{}
    require.Nil(t,
        c.Post(context.Background(), "/", req, rsp,
            client.WithSerializationType(codec.SerializationTypeForm),
            client.WithCurrentSerializationType(codec.SerializationTypeNoop),
        ))
    require.NotNil(t, rsp.Data)
    t.Logf("receive: %q\n", rsp.Data)
}
```

#### Submitting Form data with Content-Type multipart/form-data

Please follow the following steps:

1. use [mime/multipart](https://pkg.go.dev/mime/multipart) to encode the request parameters
2. wrap above encoded result into an io.Reader,
3. refer to the example in the FAQ "Client uses `io.Reader` for streaming file upload".

### Server-side File Upload (using `multipart/form-data`)

When dealing with `multipart/form-data` type data, it is always recommended to use a separate generic HTTP standard service (rather than generic HTTP RPC or RESTful services) for processing, as shown in the example below:

```go
package main

import (
    "net/http"

    thttp "git.code.oa.com/trpc-go/trpc-go/http"
)

func main() {
    s := trpc.NewServer()
    // Register HTTP standard service.
    thttp.RegisterNoProtocolServiceMux(
        s.Service("trpc.test.hello.stdhttp"),
        http.HandlerFunc(handle),
    )

    // Start server.
    s.Serve()
}

func handle(w http.ResponseWriter, r *http.Request) {
    // Custom parsing and judgment processing for RequestURI.
    uri := r.RequestURI
    if match(uri) { /*..*/ }

    r.ParseMultipartForm(0) // Parse multipart/formdata.
    // Access r.MultipartForm to get the received files, etc.
}
```

For custom routing issues of RESTful services, you can additionally refer to [Adding Extra Custom Routes to RESTful Services](../restful/README.md#adding-extra-custom-routes-to-restful-services)

### Empty req and rsp reported when using HTTP standard services and clients

First, confirm whether the business service can directly use HTTP RPC services or RESTful APIs. In both cases, req and rsp can be properly intercepted by the monitoring plugin filter.

For HTTP standard services, it is by design that req and rsp are nil. This is because the HTTP protocol cannot perfectly correspond to RPC frameworks on a one-to-one basis. Responses in the form of chunks or multipart form data, for example, cannot be compared to RPC and provide a specific rsp structure.

If the user's requirement leans more towards using HTTP as an RPC, meaning req and rsp are specific and defined with fields, in such cases, consider using HTTP RPC services with proto files or RESTful services.

If it is necessary, you can customize a pair of server-side or client-side filters to sandwich the monitoring plugin filter in between:

"http_req_collector": Before the monitoring plugin filter, provide the req that needs to be reported and restore the rsp that was modified by "http_rsp_collector".
"http_rsp_collector": After the monitoring plugin filter, provide the rsp that needs to be reported and restore the req that was modified by "http_req_collector".

```go
import (
    "bytes"
    "context"
    "net/http"

    "git.code.oa.com/trpc-go/trpc-go/codec"
    "git.code.oa.com/trpc-go/trpc-go/filter"
    thttp "git.code.oa.com/trpc-go/trpc-go/http"
)

func ExampleRegister() {
    name1 := "http_req_collector"
    name2 := "http_rsp_collector"
    // Example trpc_go.yaml:
    //
    // server:
    //   service:
    //     - name: trpc.server.service.StdHTTPMethod
    //       filter:
    //         - http_req_collector
    //         - metric_filter_name
    //         - http_rsp_collector
    // client:
    //   service:
    //     - name: trpc.server.service.StdHTTPMethod
    //       filter:
    //         - http_req_collector
    //         - metric_filter_name
    //         - http_rsp_collector
    filter.Register(name1, func(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (interface{}, error) {
        h := thttp.Head(ctx)
        if h != nil {
            w := &customResponseWriter{ResponseWriter: h.Response}
            h.Response = w
            _, err := next(ctx, &customRequest{req, h.Request}) // Pass the request you want to report.
            return w.originalRsp, err                           // Preserve the original rsp.
        }
        return next(ctx, req)
    }, func(ctx context.Context, req, rsp interface{}, next filter.ClientHandleFunc) error {
        msg := codec.Message(ctx)
        reqHeader, ok := msg.ClientReqHead().(*thttp.ClientReqHeader)
        if ok {
            // For thttp.Get, you can pass msg.ClientRPCName() to report the url parameters.
            return next(ctx, &customRequest{req, reqHeader}, rsp) // Pass the request you want to report.
        }
        return nil
    })
    filter.Register(name2, func(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (interface{}, error) {
        if cr, ok := req.(*customRequest); ok {
            h := thttp.Head(ctx)
            if h != nil {
                if w, ok := h.Response.(*customResponseWriter); ok {
                    rsp, err := next(ctx, cr.originalReq) // Preserve the original req.
                    w.originalRsp = rsp
                    return w.response.Bytes(), err // Return the response you want to report.
                }
            }
        }
        return next(ctx, req)
    }, func(ctx context.Context, req, rsp interface{}, next filter.ClientHandleFunc) error {
        if cr, ok := req.(*customRequest); ok {
            return next(ctx, cr.originalReq, rsp) // Preserve the original req.
        }
        return next(ctx, req, rsp)
    })
}

type customRequest struct {
    originalReq interface{}
    request     interface{}
}

type customResponseWriter struct {
    originalRsp interface{}
    http.ResponseWriter
    code     int
    response bytes.Buffer
}

func (w *customResponseWriter) WriteHeader(statusCode int) {
    w.code = statusCode
    w.ResponseWriter.WriteHeader(statusCode)
}

func (w *customResponseWriter) Write(bs []byte) (int, error) {
    w.response.Write(bs)
    return w.ResponseWriter.Write(bs)
}
```

### Reasons for Receiving Empty Response Content

1. Incorrect use of `client.WithCurrentSerializationType`. This option is typically used for transparent forwarding. Its essential function is to force both the request and response to use the serialization method specified by this option. Under normal circumstances, the framework determines the deserialization operation of the return packet by reading the `Content-Type` header in the return packet. If the serialization type specified by `WithCurrentSerializationType` does not match the type of the return packet itself, it is possible to get an empty return packet.
2. The server's return packet uses an inappropriate `Content-Type`. For example, the actual serialization method of the return packet content is `application/json`, but the `Content-Type` is written as `application/protobuf`. The best practice for this situation is to have the server correct its incorrect practice. For some inaccurate `Content-Type`, such as using `text/html` as the header and the actual content is `application/json`, users can manually register this `Content-Type` by calling `thttp.SetContentType("text/html", codec.SerializationTypeJSON)` during service initialization.
3. The content of the server's return packet does not correspond to the specified response structure. For example, the response body specified in the code is `type rsp struct { Message string }`, but the actual return packet is `{'data':{'message':'hello'}}`. In this case, the user needs to construct a correct response structure to ensure normal serialization, or manually read the packet and then deserialize it as mentioned in the [manual read body section](#reading-response-body-stream-using-ioreader-in-the-client).

### Restrict to Only Accept POST Method Requests

In HTTP RPC services, both GET and POST requests are acceptable. If you only want users to make requests via the POST method, you can set the `POSTOnly` field of `thttp.ServerCodec` (requires version >= v0.16.0)

```go
// Change all protocol: http services to only accept POST requests
thttp.DefaultServerCodec.POSTOnly = true
```

At this point, when a GET method request is sent, the sender will receive a "400 Bad Request" error code, and see the following error message in the "trpc-error-msg" header: "service codec Decode: server codec only allows POST method request, the current method is GET"

### Provide individual timeouts for each handler in the http_no_protocol service

The key point is to use `http.TimeoutHandler` to encapsulate your custom `http.Handler`.

An example is as follows:

```go
func TestHTTPTimeoutHandler(t *testing.T) {
    // Start server.
    const (
        network = "tcp"
        address = "127.0.0.1:0"
    )
    ln, err := net.Listen(network, address)
    require.Nil(t, err)
    defer ln.Close()
    s := server.New(
        server.WithServiceName("trpc.app.server.Service_http"),
        server.WithListener(ln),
        server.WithProtocol("http_no_protocol"))
    defer s.Close(nil)
    const timeout = 50 * time.Millisecond
    thttp.Handle("/", http.TimeoutHandler(&fileServer{sleep: 2 * timeout}, timeout, "timeout"))
    thttp.RegisterNoProtocolService(s)
    go s.Serve()

    // Start client.
    c := thttp.NewClientProxy(
        "trpc.app.server.Service_http",
        client.WithTarget("ip://"+ln.Addr().String()),
    )

    req := &codec.Body{}
    rsp := &codec.Body{}
    err = c.Post(context.Background(), "/", req, rsp,
        client.WithCurrentSerializationType(codec.SerializationTypeNoop),
        client.WithSerializationType(codec.SerializationTypeNoop),
        client.WithCurrentCompressType(codec.CompressTypeNoop),
    )
    require.NotNil(t, err)
    require.Contains(t, fmt.Sprint(err), "timeout", "expect err is timeout err, got: %s", err)
}

type fileServer struct {
    sleep time.Duration
}

func (s *fileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    time.Sleep(s.sleep)
    http.ServeFile(w, r, "./README.md")
    return
}
```

### Customize the constructed http.Request of the framework (e.g., modify Content-Length)

You can use `client.WithReqHead(&thttp.ClientReqHeader{Request: xx})` to directly specify the `http.Request` that the framework should send. However, this method cannot make the `Address` constructed by the framework's service discovery take effect (for example, it will not work when using Polaris for addressing).

The framework provides the `DecorateRequest` field in `thttp.ClientReqHeader` to make custom modifications to the `http.Request` constructed by the framework.

> trpc-go version requirement: >= v0.16.0

For example, one scenario is to use a custom `io.Reader` to send requests and manually set the Content-Length in the `http.Request`:

```go
data := []byte("hello")
reader := bytes.NewBuffer(data)
reqHeader := &thttp.ClientReqHeader{
    ReqBody: io.LimitReader(reader, int64(len(data))),
    DecorateRequest: func(r *http.Request) *http.Request {
        r.ContentLength = int64(len(data))
        return r
    },
}
req := &codec.Body{}
rsp := &codec.Body{}
c.Post(context.Background(), "/", req, rsp,
    client.WithCurrentSerializationType(codec.SerializationTypeNoop),
    client.WithReqHead(reqHeader),
)
```

When the framework constructs the `http.Request`, the length of `thttp.ClientReqHeader.ReqBody` cannot be recognized, and the standard library will eventually use chunked encoding to send the request. By specifying `thttp.ClientReqHeader.DecorateRequest` to explicitly set the Content-Length, this situation can be avoided (i.e. no chunked encoding).

For a complete test case, please refer to `transport_test.go` and the `TestDecorateRequest` test.

For the original question, please refer to: [Coder Question: How does trpc-go's http client set content-length while using the Polaris plugin?](http://mk.woa.com/q/292458)

### Supporting both generic HTTP standard services and RESTful services simultaneously

Users expect to be able to use stub-based RESTful services while handling files with generic HTTP standard services. It is recommended to read the section on adding additional custom routes to RESTful services [adding-extra-custom-routes-to-restful-services](../restful/README#adding-extra-custom-routes-to-restful-services) and support them as two separate services.

### Setting the behavior of GetSerialization for deserializing query parameters

In trpc-go before v0.16.0, the default behavior of `GetSerialization` for deserializing query parameters is case-insensitive.
In trpc-go version between v0.16.0 and v0.18.1, the default behavior of `GetSerialization` for deserializing query parameters is case-sensitive.
In trpc-go version after v0.18.1, the default behavior of `GetSerialization` for deserializing query parameters is case-insensitive.
If users want GetSerialization to deserialize query parameters in a case-sensitive manner, they can do the following:

```go
// Remember to invoke codec.RegisterSerializer to register the new Serializer.
codec.RegisterSerializer(codec.SerializationTypeGet,
    // Set the GetSerialization's caseSensitive = false.
    http.NewGetSerializationWithCaseSensitive("json", true))
```

Notice: If `GetSerialization` is set to be case-insensitive, there is a drawback that it cannot unmarshal into nested structures. For more details, see <https://git.woa.com/trpc-go/trpc-go/issues/865>.

### About the Resource Leak Issue Caused by Value Detached Transport

Due to the standard library `net/http` holding onto the passed `ctx` before go1.22, it indirectly holds onto the `ReqBody` in `ClientReqHeader`, causing memory leaks. The framework designed a value detached transport, which detaches the value from `ctx` before passing it to the lower transport layer. To preserve the timeout and cancellation capabilities of `ctx`, a new goroutine is created to listen to `ctx.Done()`. However, if the passed `ctx` only has cancel, no timeout, and `ctx` is never called to cancel, then the newly created goroutine and the resources on the original `ctx` will leak together. Although !2403 attempts to reduce the leakage of goroutines, the resource leakage is unavoidable. If users are in this scenario, it is recommended to compile with go1.22 or higher and add the following code to remove value detached transport:

```go
import (
    "net/http"

    thttp "git.code.oa.com/trpc-go/trpc-go/http"
)

func main() {
    thttp.NewRoundTripper = func(r http.RoundTripper) http.RoundTripper {
        return r
    }
}
```
