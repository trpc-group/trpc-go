English | [中文](README.zh_CN.md)

# tRPC-Go Development of Filter


## Introduction

This article introduces how to develop filter also known as interceptor, for the tRPC-Go framework. The tRPC framework uses the filter mechanism to modularize and make specific logic components of interface requests pluggable. This decouples specific business logic and promotes reusability. Examples of filters include monitoring filters, distributed tracing filters, logging filters, authentication filters, and more.

## Principles

Understanding the principles of filters is crucial, focusing on the `trigger timing` and `sequencing` of filters.

**Trigger Timing**: Filters can intercept interface requests and responses, and handle requests, responses, and contexts (in simpler terms, they can perform actions `before receiving a request` and `after processing a request`). Therefore, filters can be functionally divided into two parts: pre-processing (before business logic) and post-processing (after business logic).

**Sequencing**: As shown in the diagram below, filters follow a clear sequence. They execute the pre-processing logic in the order of filter registration and then execute the post-processing logic in reverse order.

![The Order of Filters](/.resources_without_git_lfs/filter/filter.png)

## Examples

Below is an example of how to develop a filter for reporting RPC call duration.

**Step 1**: Define the filter functions:

```golang
// ServerFilter: Server-side duration statistics from receiving the request to returning the response.
func ServerFilter(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (rsp interface{}, err error)

// ClientFilter: Client-side duration statistics from initiating the request to receiving the response.
func ClientFilter(ctx context.Context, req, rsp interface{}, next filter.ClientHandleFunc) (err error)
```

**Step 2**: Implementation:

```golang
func ServerFilter(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (interface{}, error) {
    begin := time.Now()        // Timestamp before processing business logic

    rsp, err := next(ctx, req) // Note that here you must call the next filter unless there is a specific purpose for returning directly.

    // Calculate elapsed time after processing business logic
    cost := time.Since(begin)

    // Report the elapsed time to a specific monitoring platform

    return rsp, err // You must return the rsp and err from the next function, be careful not to override them with your own logic.
}

func ClientFilter(ctx context.Context, req, rsp interface{}, next filter.ClientHandleFunc) error {
    begin := time.Now() // Timestamp before sending the request

    err := next(ctx, req, rsp)

    // Calculate elapsed time after receiving the response
    cost := time.Since(begin)

    // Report the elapsed time to a specific monitoring platform

    return err
}
```

**Step 3**: Register the filters to the framework:

```golang
filter.Register("name", ServerFilter, ClientFilter) // You can define the filter name as you like. It should be registered before trpc.NewServer().
```

**Step 4**: Configuration:

```yaml
server:
  filter: # Applies to all services
    - name1 # The name of the server filter registered in the previous step
  service:
    - name: trpc.app.server.service
      filter: # Applies only to the current service
        - name2

client:
  filter:
    - name
```

## Stream Filters

Due to the significant differences between streaming services and regular RPC calls, such as how a client initiates a streaming request and how a server handles streaming, tRPC-Go provides a different interface for stream filters.

While the exposed interface is different, the underlying implementation is similar to regular RPC filters. The principles are the same as those explained for regular RPC filters.

### Client-side

To configure a client-side stream filter, you need to implement `client.StreamFilter`:

```golang
type StreamFilter func(context.Context, *client.ClientStreamDesc, client.Streamer) (client.ClientStream, error)
```

Here's an example of a stream filter for monitoring the duration of streaming interactions:

**Step 1**: Implement `client.StreamFilter`:

```golang
func StreamClientFilter(ctx context.Context, desc *client.ClientStreamDesc, streamer client.Streamer) (client.ClientStream, error) {
    begin := time.Now() // Timestamp before creating the stream

    s, err := streamer(ctx, desc) // Note that here you must call streamer to execute the next filter unless there is a specific purpose for returning directly.

    cost := time.Since(begin) // Calculate elapsed time after creating the stream

    // Report the elapsed time to a specific monitoring platform

    return &wrappedStream{s}, err // The wrappedStream encapsulates client.ClientStream for intercepting methods like SendMsg, RecvMsg, etc. You must return the err from streamer.
}
```

**Step 2**: Wrap `client.ClientStream` and override the corresponding methods:

Since streaming services involve methods like `SendMsg`, `RecvMsg`, and `CloseSend`, you need to introduce a new struct for intercepting these interactions. You should implement the `client.ClientStream` interface in this struct. When the tRPC framework calls the `client.ClientStream` interface methods, it will execute the corresponding methods in this struct, allowing interception.

Since you may not want to intercept all `client.ClientStream` methods, you can embed `client.ClientStream` as an anonymous field in the struct. This way, methods that you don't want to intercept will pass through directly. You only need to override the methods you want to intercept.

Here's an example:

```golang
// wrappedStream encapsulates the original stream. Override the methods you want to intercept.
type wrappedStream struct {
    client.ClientStream // You must embed client.ClientStream
}

// Override RecvMsg to intercept all RecvMsg calls on the stream.
func (w *wrappedStream) RecvMsg(m interface{}) error {
    begin := time.Now() // Timestamp before receiving data

    err := w.ClientStream.RecvMsg(m) // Note that here you must call RecvMsg to let the underlying stream receive data unless there is a specific purpose for returning directly.

    cost := time.Since(begin) // Calculate elapsed time after receiving data

    // Report the elapsed time to a specific monitoring platform

    return err // You must return the err generated earlier.
}

// Override SendMsg to intercept all SendMsg calls on the stream.
func (w *wrappedStream) SendMsg(m interface{}) error {
    begin := time.Now() // Timestamp before sending data

    err := w.ClientStream.SendMsg(m) // Note that here you must call SendMsg to let the underlying stream send data unless there is a specific purpose for returning directly.

    cost := time.Since(begin) // Calculate elapsed time after sending data

    // Report the elapsed time to a specific monitoring platform

    return err // You must return the err generated earlier.
}

// Override CloseSend to intercept all CloseSend calls on the stream.
func (w *wrappedStream) CloseSend() error {
    begin := time.Now() // Timestamp before closing the local end

    err := w.ClientStream.CloseSend() // Note that here you must call CloseSend to let the underlying stream close the local end unless there is a specific purpose for returning directly.

    cost := time.Since(begin) // Calculate elapsed time after closing the local end

    // Report the elapsed time to a specific monitoring platform

    return err // You must return the err generated earlier.
}
```

**Step 3**: Configure the stream filter in the client, either through a configuration file or in code.

Option 1: Configuration File

Register the stream filter with the framework first:

```golang
client.RegisterStreamFilter("name1", StreamClientFilter) // You can define the stream filter name as you like. It should be registered before trpc.NewServer().
```

Then, configure it in the configuration file:

```yaml
client:
  stream_filter: # Applies to all services
    - name1 # The name of the client stream filter registered in the previous step
  service:
    - name: trpc.app.server.service
      stream_filter: # Applies only to the current service
        - name2
```

Option 2: Code Configuration

```golang
// Add the stream filter using client.WithStreamFilters
proxy := pb.NewGreeterClientProxy(client.WithStreamFilters(StreamClientFilter))

// Create a stream
cstream, err := proxy.ClientStreamSayHello(ctx)

// Interact with the stream
cstream.Send()
cstream.Recv()
```

### Server-side

To configure a server-side stream filter, you need to implement `server.StreamFilter`:

```golang
type StreamFilter func(Stream, *StreamServerInfo, StreamHandler) error
```

Here's an example of a server-side stream filter for monitoring the duration of streaming interactions:

**Step 1**: Implement `server.StreamFilter`:

```golang
func StreamServerFilter(ss server.Stream, si *server.StreamServerInfo, handler server.StreamHandler) error {
    begin := time.Now() // Timestamp before entering streaming processing

    // wrappedStream encapsulates server.Stream. Override SendMsg, RecvMsg, and other methods for interception.
    ws := &wrappedStream{ss}

    // Note that here you must call handler to execute the next filter unless there is a specific purpose for returning directly.
    err := handler(ws)

    cost := time.Since(begin) // Calculate elapsed time after the business process.

    // Report the elapsed time to a specific monitoring platform

    return err // You must return the err generated earlier from the handler.
}

// Override the methods you want to intercept in the wrappedStream struct.
type wrappedStream struct {
    server.Stream // You must embed server.Stream
}

// Override RecvMsg to intercept all RecvMsg calls on the stream.
func (w *wrappedStream) RecvMsg(m interface{}) error {
    begin := time.Now() // Timestamp before receiving data

    err := w.Stream.RecvMsg(m) // Note that here you must call RecvMsg to let the underlying stream receive data unless there is a specific purpose for returning directly.

    cost := time.Since(begin) // Calculate elapsed time after receiving data

    // Report the elapsed time to a specific monitoring platform

    return err // You must return the err generated earlier.
}

// Override SendMsg to intercept all SendMsg calls on the stream.
func (w *wrappedStream) SendMsg(m interface{}) error {
    begin := time.Now() // Timestamp before sending data

    err := w.Stream.SendMsg(m) // Note that here you must call SendMsg to let the underlying stream send data unless there is a specific purpose for returning directly.

    cost := time.Since(begin) // Calculate elapsed time after sending data

    // Report the elapsed time to a specific monitoring platform

    return err // You must return the err generated earlier.
}
```

**Step 3**: Configure the stream filter on the server, either through a configuration file or in code.

Option 1: Configuration File

Register the stream filter with the framework first:

```golang
server.RegisterStreamFilter("name1", StreamServerFilter) // You can define the stream filter name as you like. It should be registered before trpc.NewServer().
```

Then, configure it in the configuration file:

```yaml
server:
  stream_filter: # Applies to all services
    - name1 # The name of the server stream filter registered in the previous step
  service:
    - name: trpc.app.server.service
      stream_filter: # Applies only to the current service
        - name2
```

Option 2: Code Configuration

```golang
// Add the stream filter using server.WithStreamFilters
s := trpc.NewServer(server.WithStreamFilters(StreamServerFilter))

pb.RegisterGreeterService(s, &greeterServiceImpl{})
if err := s.Serve(); err != nil {
    log.Fatal(err)
}
```

## FAQ

### Q: Can binary data be obtained in the interceptor entry point?

No, in the interceptor entry point, both `req` and `rsp` are already serialized data structures. You can directly use the data; there is no binary data available.

### Q: How are multiple interceptors executed in order?

Multiple interceptors are executed in the order specified in the configuration file array. For example:

```yaml
server:
  filter:
    - filter1
    - filter2
  service:
    - name: trpc.app.server.service
      filter:
        - filter3
```

The execution order is as follows:

```
Request received -> filter1 pre-processing logic -> filter2 pre-processing logic -> filter3 pre-processing logic -> User's business logic -> filter3 post-processing logic -> filter2 post-processing logic -> filter1 post-processing logic -> Response sent
```

### Q: Is it necessary to set both server and client for an interceptor?

No, it's not necessary. If you only need a server-side interceptor, you can pass `nil` for the client-side interceptor, and vice versa. For example:

```golang
filter.Register("name1", serverFilter, nil) // In this case, the "name1" interceptor can only be configured in the server's filter list. Configuring it in the client will result in an RPC error.

filter.Register("name2", nil, clientFilter) // In this case, the "name2" interceptor can only be configured in the client's filter list. Configuring it in the server will cause the server to fail to start.
```
