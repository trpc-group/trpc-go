[TOC]

# Introduction

Based on the design principles of the tRPC framework, the framework needs plugin support for commonly used protocols in other business scenarios. To meet this requirement, the framework has designed a codec module that supports protocol registration.

This module is primarily designed to allow users to focus on implementing codecs to apply their own business protocols to the framework. The following mainly introduces the design principles of the plugin protocol.

Regardless of the specific protocol, it ultimately serves as a representation of requests and responses, mainly to enable users to transmit the required information more securely and efficiently. For the tRPC framework based on the client-server (C/S) architecture, the call flow of the plugin during the request and response handling process is shown in the diagram below:

![ 'tRPC Plugin Flowchart'](/.resources/developer_guide/develop_plugins/protocol/tRPC%20Plugin%20Flowchart_EN.png)

From the diagram, you can see that the tRPC-Go framework calls the Codec(Client) to encode the requests from client users. When the requests arrive at the server, the framework calls the Codec(Server) on the server side to decode the requests from users and pass them to the server-side business processing code. Finally, the server generates the corresponding response data.

tRPC uses this model to standardize various RPC protocols and client components. By adopting a unified invocation model and interceptors, it can easily implement monitoring, distributed tracing, and logging functionalities, achieving a seamless integration with the business.

Currently, tRPC-Go provides encapsulated implementations of protocols such as SSO, WNS, OIDB Proto, ILive, NRPC, as well as clients for MySQL, Redis, CKV, and other databases. You can find them in the trpc-go/trpc-codec and trpc-go/trpc-database repositories.

# Principles

## Interfaces for Protocol Design

```go
// FramerBuilder is usually used to build a Framer for each connection, which is responsible for continuously reading complete business frames from a connection.
type FramerBuilder interface {
New(io.Reader) Framer
}
```

```go
// Framer is responsible for reading and writing data frames. It reads a complete business frame from a TCP stream, copies it, and passes it to the subsequent decoding process.
type Framer interface {
ReadFrame() ([]byte, error)
}
```

```go
// Codec is the interface for packing and unpacking business protocols. The business protocol consists of a header (head) and a body. 
// Here, only the binary body is parsed, and the specific structure of the business body is handled by the serializer.
// Usually, the body is encoded in formats like protocol buffers (pb), JSON, or JCE. In special cases, the business can register its own serializer.
type Codec interface {
// Pack the body into a binary buffer
// client: Encode(msg, reqbody)(request-buffer, err)
// server: Encode(msg, rspbody)(response-buffer, err)
Encode(message Msg, body []byte) (buffer []byte, err error)
// Unpack the body from the binary buffer
// server: Decode(msg, request-buffer)(reqbody, err)
// client: Decode(msg, response-buffer)(rspbody, err)
Decode(message Msg, buffer []byte) (body []byte, err error)
}
```

## Server-side Protocol Plugin Principle

![ 'Server-side Protocol Plugin Flowchart'](/.resources/developer_guide/develop_plugins/protocol/Server-side%20Protocol%20Plugin%20Flowchart.png)

The general process for protocol handling in tRPC-Go is as follows:

1. After importing the custom protocol, register the `codec` and `FramerBuilder` in the init function.

2. Use `trpc.NewServer` to retrieve the corresponding `FramerBuilder` and `Codec` from the plugin manager based on the protocol configuration of the service, and set them up.

3. Register the RPC name and corresponding business functions.

4. Start listening.

5. The service on the server side receives a connection.

6. Build a Framer based on the `FramerBuilder`.

7. Call ReadFrame on the Framer to read a complete business frame.

8. Use the configured Codec to decode the header and business body (still as []byte). During this process, the SerializationType is usually set to obtain the serializer, and the ServerRPCName is also set to retrieve the handling method from the registered business methods.

9. Get a filter.Chain and use the serializer's Unmarshal method to deserialize the business body into the corresponding structure (e.g., protocol buffers, JCE, etc.), and pass it to the business logic code for processing.

10. The business logic returns an rsp struct.

11. Call Marshal on the serializer to serialize the rsp struct into []byte and write it back to the client.

## Client-side Protocol Plugin Principle

The process for the client's handling flow is similar to that of the server, but the steps are essentially reversed.

![ 'Client-side Protocol Plugin Flowchart'](/.resources/developer_guide/develop_plugins/protocol/Client-side%20Protocol%20Plugin%20Flowchart.png)

# Implementation

## Setting the msg Field

When setting the msg field, there are a few points to keep in mind (some values may not need to be set):

For server-side codec's Decode method after receiving a request packet, the following interfaces need to be called (values that are not applicable can be skipped):

- msg.WithServerRPCName is used to inform tRPC how to dispatch the route, e.g., /trpc.app.server.service/method.
    - `msg.WithRequestTimeout` can be used to specify the remaining timeout for the upstream service.
    - `msg.WithSerializationType` is used to specify the serialization method.
    - `msg.WithCompressType` can be used to specify the compression method.
    - `msg.WithCallerServiceName` is used to set the upstream service name, e.g., trpc.app.server.service.
    - `msg.WithCalleeServiceName` is used to set the name of the self-service.
    - `msg.WithServerReqHead` msg.WithServerRspHead can be used to set the business protocol packet header.


- For server-side codec's Encode method before sending a response packet, the following interfaces need to be called:

    - `msg.ServerRspHead` is used to retrieve the response packet header and send it back to the client.
    - `msg.ServerRspErr` is used to convert errors returned by the handler processing function into specific business protocol header error codes.


- For client-side codec's Encode method before sending a request packet, the following interfaces need to be called:

    - `msg.ClientRPCName` is used to specify the request route.
    - `msg.RequestTimeout` is used to inform the downstream service about the remaining timeout.
    - `msg.WithCalleeServiceName` is used to set the downstream service, e.g., app server service method.

- For client-side codec's Decode method after receiving a response packet, the following interfaces need to be called:
    - `errs.New` is used to convert specific business protocol error codes into an error and return it to the user calling function.
    - `msg.WithSerializationType` is used to specify the serialization method.
    - `msg.WithCompressType` can be used to specify the compression method.

## Supporting RPC Service Description for Legacy Protocols with Numeric Command Codes

Some legacy protocols, such as OIDB, use numeric command codes (command/service type) to dispatch different methods, rather than using strings like RPC does.

However, tRPC treats all services as RPC services. For non-RPC protocols that use numeric command codes, they can be converted into RPC services using annotation aliases. You can define your own service as shown below:

```go
syntax = "proto2";
package tencent.im.oidb.cmd0x110;
option go_package="git.woa.com/trpc-go/trpc-codec/oidb/examples/helloworld/cmd0x110";
message ReqBody {
optional bytes req = 1;
}
message RspBody {
optional bytes rsp = 1;
}
service Greeter {
rpc SayHello(ReqBody) returns (RspBody); // @alias=/0x110/1
}
```
- By default, tRPC services use the RPC name format /packagename.Service/Method, for example /tencent.im.oidb.cmd0x110.Greeter/SayHello. This format is not compatible with legacy protocols that use numeric command codes.
- To address this issue, the trpc tool provides a new implementation method. You can simply add // @alias=/0x110/1 after the method name, and the trpc tool will automatically replace the RPC name with the content in the comment. This way, the framework will find the corresponding method for processing based on the RPC name set in the server's Decode method.
- Any protocol with a body in protobuf or JSON format can be converted into an RPC format service.
- By running the command trpc create -protofile=xxx.proto -alias, you can create a service.

To implement this:

you can follow the example of the OIDB protocol.

# Examples

## oidb

https://git.woa.com/trpc-go/trpc-codec/tree/master/oidb

## tars

https://git.woa.com/trpc-go/trpc-codec/tree/master/tars

# Summary

To implement a business protocol, you need to implement a Framer to extract complete business packets from TCP. You also need to implement the server codec and client codec interfaces, as well as the serializer if necessary. Additionally, make sure to set some metadata in the encode and decode methods to find the processing methods or perform marshal and unmarshal operations.

