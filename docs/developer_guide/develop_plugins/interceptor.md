[TOC]

# Developing Interceptor Plugins for tRPC-Go



## 1. Introduction

This article introduces how to develop interceptors (also known as filters) for the tRPC-Go framework. The tRPC framework utilizes the interceptor mechanism to modularize and plug-in specific logical components related to interface requests, thereby decoupling them from specific business logic and achieving the purpose of reuse. Examples of interceptors include monitoring interceptors, distributed tracing interceptors, logging interceptors, authentication interceptors, and so on.

## 2. Principles

Understanding the `trigger timing` and `order` of interceptors is key to understanding their principles.

Trigger timing: Interceptors can intercept interface requests and responses, and process requests, responses and contexts (in simple terms, they can do something before the request is received and after the request is processed). Therefore interceptors are functionally divided into two parts: pre-processing (before business processing) and post-processing (after business logic processing).

Order: As shown in the figure below, interceptors have a clear order, and the pre-processing logic is executed in order according to the registration order of the interceptors, and the post-processing part of the interceptors is executed in reverse order.

![The Order of Interceptors](/.resources/developer_guide/develop_plugins/interceptor/interceptor.png)

## 3. Implementation

To understand implementation of interceptors, you need to master the following parts:

* `Invocation timing`: The invocation logic of interceptors is called in the generated stub code, so it cannot seen the framework code. The framework passes the interceptor's processing function to the generated code for use.
* Interceptors are implemented recursively.

If there is only one interceptor, it is sufficient to return the interceptor to the generated code for invocation.
If there are multiple interceptors, multiple interceptors need to be encapsulated as the next one interceptor.

The following is an implementation of abstracting multiple interceptor processing into one interceptor. The core is `chainFunc`, which is a recursive closure that allows sequentially registered interceptors to be executed in order. It can be difficult to understand this function by looking at it, but it be understood through debugging with unit tests.

```go
func (fc Chain) Handle(ctx context.Context, req interface{}, rsp interface{}, f HandleFunc) (err error) {
    n := len(fc)
    curI := -1
    // multiple filters are consumed recursively 
    var chainFunc HandleFunc
    chainFunc = func(ctx context.Context, req interface{}, rsp interface{}) error {
        if curI == n-1 {
            return f(ctx, req, rsp)
        }
        curI++
        return fc[curI](ctx, req, rsp, chainFunc)
    }
    return chainFunc(ctx, req, rsp)
}
```

When implementing business logic, you only need to implement the Filter interface.

```go
type Filter func(ctx context.Context, req, rsp interface{}, handler filter.HandleFunc) error
```

## 4. Examples


Next, let me give an example of how to develop an interceptor for reporting the time consumption of an RPC.

Step 1: The following is the function prototype for implementing the interceptor.

```go
// ServerFilter server: time consumption statistics, from the time of receiving the request to the time of returning the response.
func ServerFilter() filter.ServerFilter {
    return func(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (rsp interface{}, err error) {
    }
}
```

```go
// ClientFilter client: time consumption statistics, from the time of initiating the network request to the time of receiving the response.
func ClientFilter() filter.ClientFilter {
    return func(ctx context.Context, req, rsp interface{}, next filter.HandleFunc) (err error) {
    }
}
```

Step 2: Implementation.

```go
func ServerFilter() filter.ServerFilter {
    return func(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (interface{}, error) {
        begin := time.Now() // Timestamp recorded
        rsp, err := next(ctx, req) //  Call the next interceptor themselves unless there is a specific purpose to return directly.
        cost := time.Since(begin) // Calculate the elapsed time
        _ = cost
        // reportxxx reporting to a specific monitoring platform
        return rsp, err //  It's important to return both the rsp and err from the next function, and to be careful not to overwrite them with your own logic.
    }
}
func ClientFilter() filter.ClientFilter {
    return func(ctx context.Context, req, rsp interface{}, next filter.HandleFunc) (err error) {
        begin := time.Now() // Timestamp recorded
        err = next(ctx, req, rsp)
        cost := time.Since(begin) // Calculate the elapsed time
        // reportxxx reporting to a specific monitoring platform
        return err
    }
}
```

Step 3: Register the interceptor with the framework.

```go
filter1 := ServerFilter()
filter2 := ClientFilter()
filter.Register("name", filter1, filter2) // Define the name of the interceptor as you like, which will be used in the configuration file later. It's important to register the interceptor before trpc.NewServer().
```

Step 4: Configure and use the interceptor.

```yaml
server:
 filter:  # Effective for all services.
   - name1  #  The name of the server interceptor registered with the framework in the step 3 above.
 service:
   - name: trpc.app.server.service
     filter:  # Only effective for the current service.
       - name2  
client:
 ...
 filter:
  ...
  - name
```

## Streaming Interceptor

Because streaming services have a very different calling interface from simple RPCs. For example, a simple RPC client initiates an RPC call through proxy.SayHello, but a streaming client creates a stream through proxy.ClientStreamSayHello. Once the stream is created, the interaction of the stream is done through SendMsg, RecvMsg, and CloseSend. Therefore, for streaming services, a different interceptor interface is provided.

Although the exposed interfaces are different, the underlying implementation of streaming services is similar to simple RPCs, and the principles are based on the principles of simple RPC interceptors.

### Client Configuration

To configure streaming interceptors on the client, you need to implement client.StreamFilter.

```go
type StreamFilter (ctx context.Context, desc *client.ClientStreamDesc, streamer client.Streamer) (client.ClientStream, error)
```

Let's take the example of developing a streaming interceptor for reporting the time consumption during the streaming interaction.

Step 1: Implement client.StreamFilter.

```go
func streamClientFilter(ctx context.Context, desc *client.ClientStreamDesc, streamer client.Streamer) (client.ClientStream, error) {
    begin := time.Now() // Timestamp recorded
    s, err := streamer(ctx, desc) // call the next interceptor themselves unless there is a specific purpose to return directly.
    cost := time.Since(begin) // Calculate the elapsed time
    // reportxxx reporting to a specific monitoring platform
    return newWrappedStream(s), err // newWrappedStream creates a wrapped stream structure that is used to intercept interfaces such as SendMsg and RecvMsg. Note that newWrappedStream must return the error of the streamer.
}
```

Step 2: Implement the wrapped structure and override client.ClientStream method.

In the streaming service interaction, the client has interfaces such as SendMsg, RecvMsg, and CloseSend. To intercept these interactions, the concept of a wrapped structure needs to be introduced. You need to implement all interfaces of client.ClientStream for this structure. When the framework calls the interface of client.ClientStream, it first executes the corresponding method of this structure, thus achieving interception.

Because you may not need to intercept all interfaces of client.ClientStream, client.ClientStream can be set as an anonymous field of the structure. In this way, interfaces that do not need to be intercepted will directly use the underlying methods. You can override the interfaces they want to intercept in this structure.

For example, if you only want to intercept the process of sending data, just override the SendMsg method, and leave alone other interfaces of client.ClientStream. Here, for demonstration purposes, all interfaces of client.ClientStream are implemented (except Context).

```go
// wrappedStream is a wrapped stream structure, and the interfaces that need to be intercepted should be overridden in this structure.
type wrappedStream struct {
    client.ClientStream // The wrappedStream structure must include the client.ClientStream field.
}
// newWrappedStream creates a wrapped stream structure.
func newWrappedStream(s client.ClientStream) client.ClientStream {
    return &wrappedStream{s}
}
// Override RecvMsg to intercept all RecvMsg calls of the stream.
func (w *wrappedStream) RecvMsg(m interface{}) error {
    begin := time.Now() // Before receiving data, record a timestamp for tracking purposes.
    err := w.ClientStream.RecvMsg(m) // Note that must call RecvMsg themselves to allow the underlying stream to receive data, unless there is a specific purpose to directly return.
    cost := time.Since(begin) // After receiving the data, calculate the elapsed time.
    // reportxxx reporting to a specific monitoring platform
    return err // Note that here the err generated earlier must be returned.
}
// Override SendMsg to intercept all SendMsg calls of the stream.
func (w *wrappedStream) SendMsg(m interface{}) error {
    begin := time.Now() // Before sending data, record a timestamp for tracking purposes.
    err := w.ClientStream.SendMsg(m) // Note that call SendMsg themselves to allow the underlying stream to send data, unless there is a specific purpose to directly return.
    cost := time.Since(begin) // After sending the data, calculate the elapsed time.
    // reportxxx reporting to a specific monitoring platform
    return err // Note that here the err generated earlier must be returned.
}
// Override CloseSend to intercept all CloseSend calls of the stream.
func (w *wrappedStream) CloseSend() error {
    begin := time.Now() // Before closing the local end, record a timestamp for tracking purposes.
    err := w.ClientStream.CloseSend() // Note that call CloseSend themselves to allow the underlying stream to close the local end, unless there is a specific purpose to directly return.
    cost := time.Since(begin) // After closing the local end, calculate the elapsed time.
    // reportxxx reporting to a specific monitoring platform
    return err // Note that here the err generated earlier must be returned.
}
```
Step 3: Configure the interceptor to the client, which can be done through a configuration file or in the code.

Method 1: Configuration file

First, register the interceptor with the framework.

```go
streamFilter := ClientStreamFilter()
client.RegisterStreamFilter("name1", streamFilter)    // The interceptor name can be defined arbitrarily for later use in the configuration file, but it must be placed before trpc.NewServer().
```

Then configure it in the configuration file.

```yaml
client:
 stream_filter:  # Effective for all services.
   - name1        # The name of the client stream interceptor registered with the framework above.
 service:
   - name: trpc.app.server.service
     stream_filter:  # Effective only for the current service.
       - name2
```

Option 2: Configure in code.

```go
// Add the interceptor through client.WithStreamFilters
proxy := pb.NewGreeterClientProxy(client.WithStreamFilters(streamClientFilter))
// Create a stream
cstream, err := proxy.ClientStreamSayHello(ctx)
// Stream interaction process...
cstream.Send(...)
cstream.Recv()
```

### Server Configuration

To configure a streaming interceptor on the server, implementation of server.StreamFilter is required.

```go
type StreamFilter func(ss Stream, info *StreamServerInfo, handler StreamHandler) error
```

As an example of how to develop a streaming interceptor for reporting time-consuming statistics during streaming interactions:

Step 1: Implement server.StreamFilter.

```go
func streamServerFilter(ss server.Stream, si *server.StreamServerInfo,
    handler server.StreamHandler) error {
    begin := time.Now() // Record timestamp before entering streaming processing
    // newWrappedStream creates a wrapped stream structure for subsequent interception of SendMsg, RecvMsg, and other interfaces
    ws := newWrappedStream(ss)
    // Note that the handler must be called to execute the next interceptor, unless there is a specific purpose to return directly.
    err := handler(ws) 
    cost := time.Since(begin) // Calculate the time-consuming after the processing function exits
    // reportxxx reports to a specific monitoring platform
    return err // Note that the err of the handler must be returned here
}
```

Step 2: Implement the wrapped structure and override the server.Stream method.

Because there are interfaces such as SendMsg and RecvMsg on the server side during the interaction process of streaming services, in order to intercept these interaction processes, the concept of a wrapped structure needs to be introduced. You need to implement all the server.Stream interfaces for this structure. When the framework calls the server.Stream interface, the corresponding method of this structure is executed first, thus achieving interception.

Because you may not need to intercept all the server.Stream interfaces, server.Stream can be set as an anonymous field of the structure. In this way, the interfaces that do not need to be intercepted will directly use the underlying methods. You can override the interfaces they want to intercept in this structure.

For example, if you only want to intercept the process of sending data, then just override the SendMsg method, and leave alone other server.Stream interfaces. Here, for demonstration purposes, all server.Stream interfaces are implemented (except for Context).

```go
// wrappedStream is the wrapped structure for streaming interception. Override the interfaces that need to be intercepted.
type wrappedStream struct {
    server.Stream // The wrapped type must contain the `server.Stream` field.
}

// newWrappedStream creates a wrapped stream structure.
func newWrappedStream(s server.Stream) server.Stream {
    return &wrappedStream{s}
}

// Override `RecvMsg` to intercept all `RecvMsg` calls of the stream.
func (w *wrappedStream) RecvMsg(m interface{}) error {
    begin := time.Now() // Record timestamp before receiving data
    err := w.Stream.RecvMsg(m) // Note that call `RecvMsg` to let the underlying stream receive data, unless there is a specific purpose to return directly.
    cost := time.Since(begin) // Calculate the time-consuming after receiving data
    // reportxxx reports to a specific monitoring platform
    return err // Note that the err generated earlier must be returned here.
}

// Override `SendMsg` to intercept all `SendMsg` calls of the stream.
func (w *wrappedStream) SendMsg(m interface{}) error {
    begin := time.Now() // Record timestamp before sending data
    err := w.Stream.SendMsg(m) // Note that call `SendMsg` to let the underlying stream send data, unless there is a specific purpose to return directly.
    cost := time.Since(begin) // Calculate the time-consuming after sending data
    // reportxxx reports to a specific monitoring platform
    return err // Note that the err generated earlier must be returned here.
}
```

Step 3: Configure the interceptor on the server. It can be configured through a configuration file or in the code.

Option 1: Configure in a configuration file.

First, register the interceptor with the framework.

```go
streamFilter := ServerStreamFilter()
server.RegisterStreamFilter("name1", streamFilter)    // The interceptor name can be defined arbitrarily for subsequent configuration file use. It must be placed before `trpc.NewServer()`.
```

Then configure it in the configuration file:
```yaml
server:
 stream_filter:  # Applies to all services
   - name1        # The name of the server streaming interceptor registered with the framework
 service:
   - name: trpc.app.server.service
     stream_filter:  # Only applies to the current service
       - name2
```

Option 2: Configure in the code.

First, register the interceptor with the framework.

```go
// Add the interceptor through `server.WithStreamFilters`.
s := trpc.NewServer(server.WithStreamFilters(streamServerFilter))
pb.RegisterGreeterService(s, &greeterServiceImpl{})
if err := s.Serve(); err != nil {
    log.Fatal(err)
}
```

## FAQ

### Q1: Can binary data be obtained in the interceptor entry?

No, the req and rsp in the interceptor entry are already serialized structures, and the data can be used directly without binary data.

### Q2: What is the execution order of multiple interceptors?

The execution order of multiple interceptors is based on the order of the array in the configuration file, such as:

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

```shell
Receive request -> filter1 pre-processing logic -> filter2 pre-processing logic -> filter3 pre-processing logic -> your business processing logic -> filter3 post-processing logic -> filter2 post-processing logic -> filter1 post-processing logic -> Return response
```

### Q3: Does an interceptor need to be set for both server and client at the same time?

No, it is not necessary. When only the server is needed, pass nil to the client, and vice versa. For example:

```go
filter.Register("name1", serverFilter, nil)  // Note that the name1 interceptor can only be configured in the server's filter list. Configuring it in the client will cause an RPC error.
filter.Register("name2", nil, clientFilter)  // Note that the name2 interceptor can only be configured in the client's filter list. Configuring it in the server will cause the server to fail to start.
```

