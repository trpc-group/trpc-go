---
title: message.doc
---
# Introduction

This document will walk you through the implementation of message handling in the <SwmPath>[codec/message.go](/codec/message.go)</SwmPath> file. The purpose of this code is to define and manage message data structures and operations for multi-protocol communication within the <SwmToken path="/codec/message.go" pos="3:24:24" line-data="// Tencent is pleased to support the open source community by making tRPC available.">`tRPC`</SwmToken> framework.

We will cover:

1. The rationale behind defining custom types for context keys and metadata.
2. The use of <SwmToken path="/codec/message.go" pos="33:6:8" line-data="var msgPool = sync.Pool{">`sync.Pool`</SwmToken> for efficient message handling.
3. The design of the <SwmToken path="/codec/message.go" pos="74:2:2" line-data="// Msg defines core message data for multi protocol, business protocol">`Msg`</SwmToken> interface for message operations.
4. The implementation of the <SwmToken path="/codec/message.go" pos="344:2:2" line-data="// CopyMsg copy src Msg to dst.">`CopyMsg`</SwmToken> function for duplicating message data.

# Custom types for context keys and metadata

<SwmSnippet path="/codec/message.go" line="25">

---

The code defines custom types for context keys and metadata to avoid conflicts and ensure type safety. This is crucial for managing context-specific data and metadata in a structured manner.

```
// ContextKey is trpc context key type, the specific value is judged
// by interface, the interface will both judge value and type. Defining
// a new type can avoid string value conflict.
type ContextKey string

// MetaData is request penetrate message.
type MetaData map[string][]byte
```

---

</SwmSnippet>

# Efficient message handling with <SwmToken path="/codec/message.go" pos="33:6:8" line-data="var msgPool = sync.Pool{">`sync.Pool`</SwmToken>

<SwmSnippet path="/codec/message.go" line="33">

---

A <SwmToken path="/codec/message.go" pos="33:6:8" line-data="var msgPool = sync.Pool{">`sync.Pool`</SwmToken> is used to manage message instances efficiently. This approach reduces the overhead of repeatedly allocating and deallocating memory for message objects, improving performance in high-load scenarios.

```
var msgPool = sync.Pool{
	New: func() interface{} {
		return &msg{}
	},
}

// Clone returns a copied meta data.
func (m MetaData) Clone() MetaData {
	if m == nil {
		return nil
	}
	md := MetaData{}
	for k, v := range m {
		md[k] = v
	}
	return md
}
```

---

</SwmSnippet>

# Design of the Msg interface

<SwmSnippet path="/codec/message.go" line="74">

---

The <SwmToken path="/codec/message.go" pos="74:2:2" line-data="// Msg defines core message data for multi protocol, business protocol">`Msg`</SwmToken> interface is central to the message handling logic. It provides methods for setting and retrieving various attributes related to message data, such as addresses, namespaces, and metadata. This design allows for flexible and consistent message operations across different protocols.

```
// Msg defines core message data for multi protocol, business protocol
// should set this message when packing and unpacking data.
type Msg interface {
	// Context returns rpc context
	Context() context.Context

	// WithRemoteAddr sets upstream address for server,
	// or downstream address for client.
	WithRemoteAddr(addr net.Addr)
```

---

</SwmSnippet>

# Implementation of the <SwmToken path="/codec/message.go" pos="344:2:2" line-data="// CopyMsg copy src Msg to dst.">`CopyMsg`</SwmToken> function

<SwmSnippet path="/codec/message.go" line="344">

---

The <SwmToken path="/codec/message.go" pos="344:2:2" line-data="// CopyMsg copy src Msg to dst.">`CopyMsg`</SwmToken> function is implemented to copy all fields from a source message to a destination message. This is important for scenarios where message data needs to be duplicated, ensuring that all relevant attributes are transferred accurately.

```
// CopyMsg copy src Msg to dst.
// All fields of src msg will be copied to dst msg.
func CopyMsg(dst, src Msg) {
	if dst == nil || src == nil {
		return
	}
	dst.WithFrameHead(src.FrameHead())
	dst.WithRequestTimeout(src.RequestTimeout())
	dst.WithSerializationType(src.SerializationType())
	dst.WithCompressType(src.CompressType())
	dst.WithStreamID(src.StreamID())
	dst.WithDyeing(src.Dyeing())
	dst.WithDyeingKey(src.DyeingKey())
	dst.WithServerRPCName(src.ServerRPCName())
	dst.WithClientRPCName(src.ClientRPCName())
	dst.WithServerMetaData(src.ServerMetaData().Clone())
	dst.WithClientMetaData(src.ClientMetaData().Clone())
	dst.WithCallerServiceName(src.CallerServiceName())
	dst.WithCalleeServiceName(src.CalleeServiceName())
	dst.WithCalleeContainerName(src.CalleeContainerName())
	dst.WithServerRspErr(src.ServerRspErr())
	dst.WithClientRspErr(src.ClientRspErr())
	dst.WithServerReqHead(src.ServerReqHead())
	dst.WithServerRspHead(src.ServerRspHead())
	dst.WithClientReqHead(src.ClientReqHead())
	dst.WithClientRspHead(src.ClientRspHead())
	dst.WithLocalAddr(src.LocalAddr())
	dst.WithRemoteAddr(src.RemoteAddr())
	dst.WithLogger(src.Logger())
	dst.WithCallerApp(src.CallerApp())
	dst.WithCallerServer(src.CallerServer())
	dst.WithCallerService(src.CallerService())
	dst.WithCallerMethod(src.CallerMethod())
	dst.WithCalleeApp(src.CalleeApp())
	dst.WithCalleeServer(src.CalleeServer())
	dst.WithCalleeService(src.CalleeService())
	dst.WithCalleeMethod(src.CalleeMethod())
	dst.WithNamespace(src.Namespace())
	dst.WithSetName(src.SetName())
	dst.WithEnvName(src.EnvName())
	dst.WithEnvTransfer(src.EnvTransfer())
	dst.WithRequestID(src.RequestID())
	dst.WithStreamFrame(src.StreamFrame())
	dst.WithCalleeSetName(src.CalleeSetName())
	dst.WithCommonMeta(src.CommonMeta().Clone())
	dst.WithCallType(src.CallType())
}
```

---

</SwmSnippet>

This document has outlined the key design decisions and their rationale in the message handling code. Each section highlights the importance of the respective implementation in achieving efficient and structured message operations within the <SwmToken path="/codec/message.go" pos="3:24:24" line-data="// Tencent is pleased to support the open source community by making tRPC available.">`tRPC`</SwmToken> framework.

<SwmMeta version="3.0.0" repo-id="Z2l0aHViJTNBJTNBdHJwYy1nbyUzQSUzQXNoYWluZXNkdQ==" repo-name="trpc-go"><sup>Powered by [Swimm](https://app.swimm.io/)</sup></SwmMeta>
