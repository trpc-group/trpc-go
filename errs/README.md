English | [中文](README.zh_CN.md)

# tRPC-Go Error Code Definition


## Introduction

All languages of tRPC frameworks use a unified error definition consisting of an error code `code` and an error description `msg`. This differs from the Golang standard library's error, which contains only a string. Therefore, in tRPC-Go, the `errs` package is used to encapsulate error types, making it easier for users to work with errors. When a user needs to return an error, use `errs.New(code, msg)` to create the error instead of using the standard library's `errors.New(msg)` as shown below:

```golang
func (s *greeterServerImpl) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
    if failed { // Business logic failure
        // Define an error code and return the error message to the caller
        return nil, errs.New(int(your-err-code), "your business error message")
    }
    return &pb.HelloRepley{}, nil // Business logic succeeded
}
```

## Error Code Type

tRPC-Go error codes are divided into `framework` errors, `callee framework` errors, and `business` errors.

### Framework Errors

These are automatically returned by the current service's framework, such as timeouts, decoding errors, etc. All framework error codes used by tRPC are defined in [trpc.proto](https://github.com/trpc-group/trpc/blob/main/trpc/trpc.proto).

- 0-100: Server errors, indicating errors that occur before entering the business logic, such as when the framework encounters an error before processing the request from the network layer. It doesn't affect the business logic.
- 101-200: Client errors, which means errors that occur at the client level when invoking downstream services.
- 201-400: Streaming errors.

Typical log representation:

```golang
type:framework, code:101, msg:xxx timeout
```

### Callee Framework Errors

These are the error codes returned by the framework of the callee service. They may be transparent to the business development of the callee service, but they are clear indicators of errors returned by the callee service, and they have no direct relation to the current service. Typically, these errors are caused by parameter issues in the current service, and solving them requires checking request parameters and collaborating with the callee service.

Typical log representation:

```golang
type:callee framework, code:12, msg:rpcname:xxx invalid
```

### Business Errors

These are the error codes returned by the business logic of the callee service. Note that these error types are specific to the business logic of the callee service and are defined by the callee service itself. The specific meaning of these codes should be checked in the documentation of the callee service or by consulting the callee service's developers.

tRPC-Go recommends using `errs.New` to return business error codes when there is a business error, rather than defining error codes in response messages. Error codes and response messages are mutually exclusive, so if an error is returned, the framework will ignore the response message.

It is recommended that user-defined error codes start from 10000.

Typical log representation:

```golang
type:business, code:10000, msg:xxx fail
```

## Error Code Meanings

**Note: The following error codes refer to framework errors and callee framework errors. Business error codes are defined by the callee service and should be checked in the callee service's documentation or by consulting the callee service's developers. Error codes provide a general categorization of error types; specific error causes should be examined in the error message.**

| Error Code | Meanings                                                                                                                                                                                                                                                               |
| :--------: | :--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
|     0      | Success                                                                                                                                                                                                                                                                |
|     1      | Server decoding error, usually caused by misalignment or lack of synchronization of pb fields between caller and callee services, leading to failed unpacking. To resolve, ensure that both services are updated to the latest version of pb and keep pb synchronized. |
|     2      | Server encoding error, serialization of response packets failed, typically due to issues with pb fields, such as setting binary data with invisible characters to a string field. Check the error message for details.                                                 |
|     11     | Server doesn't have the corresponding service implementation.                                                                                                                                                                                                          |
|     12     | Server doesn't have the corresponding interface implementation, calling function was incorrect.                                                                                                                                                                        |
|     21     | Server-side business logic processing time exceeded the timeout, exceeding the link timeout or message timeout.                                                                                                                                                        |
|     22     | Server is overloaded, typically because the callee server used a overload control plugin.                                                                                                                                                                              |
|     23     | The request is rate-limited by the server.                                                                                                                                                                                                                             |
|     24     | Server full-link timeout, i.e., the timeout given by the caller was too short, and it did not even enter the business logic of this service.                                                                                                                           |
|     31     | Server system error, typically caused by panic, most likely a null pointer or array out of bounds error in the called service.                                                                                                                                         |
|     41     | Authentication failed.                                                                                                                                                                                                                                                 |
|     51     | Request parameters validates failed.                                                                                                                                                                                                                                   |
|    101     | Timeout when making a client call.                                                                                                                                                                                                                                     |
|    102     | Full-link timeout on the client side, meaning the timeout given for this RPC call was too short, potentially because previous RPC calls had already consumed most of the time.                                                                                         |
|    111     | Client connection error, typically because the callee service is not listening on the specified IP:Port or the callee service failed to start.                                                                                                                         |
|    121     | Client encoding error, serialization of request packets failed.                                                                                                                                                                                                        |
|    122     | Client decoding error, typically due to misalignment of pb.                                                                                                                                                                                                            |
|    123     | Rate limit exceeded by the client.                                                                                                                                                                                                                                     |
|    124     | Client overload error.                                                                                                                                                                                                                                                 |
|    131     | Client IP routing error, typically due to a misspelled service name or no available instances under that service name.                                                                                                                                                 |
|    141     | Client network error.                                                                                                                                                                                                                                                  |
|    151     | Response parameters validates failed.                                                                                                                                                                                                                                  |
|    161     | Request canceled prematurely by the caller caller.                                                                                                                                                                                                                     |
|    171     | Client failed to read frame data.                                                                                                                                                                                                                                      |
|    201     | Server-side streaming network error.                                                                                                                                                                                                                                   |
|    351     | Client streaming data reading failed.                                                                                                                                                                                                                                  |
|    999     | Unknown error, typically occurs when the callee service uses `errors.New(msg)` from the Golang standard library to return an error without a numeric code.                                                                                                             |

## Implementation

The specific implementation structure of error is as follows:

```golang
type Error struct {
    Type int    // Error code type: 1 for framework errors, 2 for business errors, 3 for callee framework errors
    Code int32  // Error code
    Msg  string // Error message
    Desc string // Additional error description, mainly used for monitoring prefixes, such as "trpc" for tRPC framework errors and "http" for HTTP protocol errors. Users can capture this error by implementing filters and change this field to report monitoring data with any desired prefix.
}
```

Error handling process:

- When the server returns a business error using `errs.New`, the framework fills this error into the `func_ret` field in the tRPC protocol header.
- When the server returns a framework error using `errs.NewFrameError`, the framework fills this error into the `ret` field in the tRPC protocol header.
- When the server framework responds, it checks whether there is an error. If there is an error, the response data is discarded. Therefore, when an error is returned, do not attempt to return data in the response.
- When the client invokes an RPC call, the framework needs to encode the request before sending the request downstream. If an error occurs before sending the network data, it directly returns `Framework` error to the user. If the request is successfully sent and a response is received, but a framework error is detected in the response's protocol header, it returns `Callee Framework` error. If a business error is detected during parsing, it returns `Business` error.
