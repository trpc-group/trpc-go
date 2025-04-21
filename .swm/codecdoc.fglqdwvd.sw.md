---
title: codec.doc
---
# Introduction

This document will walk you through the implementation of the codec package in the <SwmPath>[codec/codec.go](/codec/codec.go)</SwmPath> file. The codec package is responsible for defining the business communication protocol for packing and unpacking messages.

We will cover:

1. The purpose and structure of the codec package.
2. The definition and role of <SwmToken path="/codec/codec.go" pos="24:2:2" line-data="// RequestType is the type of client request, such as SendAndRecv，SendOnly.">`RequestType`</SwmToken>.
3. The <SwmToken path="/codec/codec.go" pos="34:2:2" line-data="// Codec defines the interface of business communication protocol,">`Codec`</SwmToken> interface and its methods.
4. The registration and retrieval of codecs.

# Codec package overview

<SwmSnippet path="/codec/codec.go" line="14">

---

The codec package is designed to handle the packing and unpacking of messages in a business communication protocol. It allows for the encoding and decoding of message bodies, which can be in various formats such as protobuf or JSON. This flexibility is achieved by allowing the registration of custom serializers.

```
// Package codec defines the business communication protocol of
// packing and unpacking.
package codec

import (
	"sync"

	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"
)
```

---

</SwmSnippet>

# <SwmToken path="/codec/codec.go" pos="24:2:2" line-data="// RequestType is the type of client request, such as SendAndRecv，SendOnly.">`RequestType`</SwmToken> definition

<SwmSnippet path="/codec/codec.go" line="24">

---

The <SwmToken path="/codec/codec.go" pos="24:2:2" line-data="// RequestType is the type of client request, such as SendAndRecv，SendOnly.">`RequestType`</SwmToken> is an enumeration that defines the type of client requests. It distinguishes between requests that expect a response and those that do not.

```
// RequestType is the type of client request, such as SendAndRecv，SendOnly.
type RequestType int

const (
	// SendAndRecv means send one request and receive one response.
	SendAndRecv = RequestType(trpcpb.TrpcCallType_TRPC_UNARY_CALL)
	// SendOnly means only send request, no response.
	SendOnly = RequestType(trpcpb.TrpcCallType_TRPC_ONEWAY_CALL)
)
```

---

</SwmSnippet>

# Codec interface

<SwmSnippet path="/codec/codec.go" line="34">

---

The <SwmToken path="/codec/codec.go" pos="34:2:2" line-data="// Codec defines the interface of business communication protocol,">`Codec`</SwmToken> interface is central to the package. It defines methods for encoding and decoding message bodies. The <SwmToken path="/codec/codec.go" pos="40:3:3" line-data="	// Encode pack the body into binary buffer.">`Encode`</SwmToken> method packs the body into a binary buffer, while the <SwmToken path="/codec/codec.go" pos="45:3:3" line-data="	// Decode unpack the body from binary buffer">`Decode`</SwmToken> method unpacks the body from a binary buffer. This separation of concerns allows for clear handling of message serialization and deserialization.

```
// Codec defines the interface of business communication protocol,
// which contains head and body. It only parses the body in binary,
// and then the business body struct will be handled by serializer.
// In common, the body's protocol is pb, json, etc. Specially,
// we can register our own serializer to handle other body type.
type Codec interface {
	// Encode pack the body into binary buffer.
	// client: Encode(msg, reqBody)(request-buffer, err)
	// server: Encode(msg, rspBody)(response-buffer, err)
	Encode(message Msg, body []byte) (buffer []byte, err error)
```

---

</SwmSnippet>

<SwmSnippet path="/codec/codec.go" line="45">

---

The <SwmToken path="/codec/codec.go" pos="45:3:3" line-data="	// Decode unpack the body from binary buffer">`Decode`</SwmToken> method complements <SwmToken path="/codec/codec.go" pos="40:3:3" line-data="	// Encode pack the body into binary buffer.">`Encode`</SwmToken> by providing the logic to unpack the binary buffer back into a message body.

```
	// Decode unpack the body from binary buffer
	// server: Decode(msg, request-buffer)(reqBody, err)
	// client: Decode(msg, response-buffer)(rspBody, err)
	Decode(message Msg, buffer []byte) (body []byte, err error)
}
```

---

</SwmSnippet>

# Codec registration and retrieval

The package maintains maps for client and server codecs, allowing for the registration and retrieval of codecs by name. This is crucial for supporting different communication protocols within the same application.

<SwmSnippet path="/codec/codec.go" line="51">

---

The <SwmToken path="/codec/codec.go" pos="57:2:2" line-data="// Register defines the logic of register a codec by name. It will be">`Register`</SwmToken> function is responsible for adding codecs to the maps. It uses a lock to ensure thread safety during the registration process.

```
var (
	clientCodecs = make(map[string]Codec)
	serverCodecs = make(map[string]Codec)
	lock         sync.RWMutex
)

// Register defines the logic of register a codec by name. It will be
// called by init function defined by third package. If there is no server codec,
// the second param serverCodec can be nil.
func Register(name string, serverCodec Codec, clientCodec Codec) {
	lock.Lock()
	serverCodecs[name] = serverCodec
	clientCodecs[name] = clientCodec
	lock.Unlock()
}
```

---

</SwmSnippet>

<SwmSnippet path="/codec/codec.go" line="67">

---

The <SwmToken path="/codec/codec.go" pos="67:2:2" line-data="// GetServer returns the server codec by name.">`GetServer`</SwmToken> and <SwmToken path="/codec/codec.go" pos="75:2:2" line-data="// GetClient returns the client codec by name.">`GetClient`</SwmToken> functions provide access to the registered server and client codecs, respectively. They use read locks to ensure safe concurrent access.

```
// GetServer returns the server codec by name.
func GetServer(name string) Codec {
	lock.RLock()
	c := serverCodecs[name]
	lock.RUnlock()
	return c
}

// GetClient returns the client codec by name.
func GetClient(name string) Codec {
	lock.RLock()
	c := clientCodecs[name]
	lock.RUnlock()
	return c
}
```

---

</SwmSnippet>

This concludes the walkthrough of the codec package. The design decisions made here ensure flexibility and extensibility in handling various communication protocols.

<SwmMeta version="3.0.0" repo-id="Z2l0aHViJTNBJTNBdHJwYy1nbyUzQSUzQXNoYWluZXNkdQ==" repo-name="trpc-go"><sup>Powered by [Swimm](https://app.swimm.io/)</sup></SwmMeta>
