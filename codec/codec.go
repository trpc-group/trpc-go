// Package codec defines the business communication protocol of
// packing and unpacking.
package codec

import (
	"sync"

	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"
)

// RequestType is the type of client request, such as SendAndRecv，SendOnly.
type RequestType int

const (
	// SendAndRecv means send one request and receive one response.
	SendAndRecv = RequestType(trpcpb.TrpcCallType_TRPC_UNARY_CALL)
	// SendOnly means only send request, no response.
	SendOnly = RequestType(trpcpb.TrpcCallType_TRPC_ONEWAY_CALL)
)

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

	// Decode unpack the body from binary buffer
	// server: Decode(msg, request-buffer)(reqBody, err)
	// client: Decode(msg, response-buffer)(rspBody, err)
	Decode(message Msg, buffer []byte) (body []byte, err error)
}

var (
	clientCodecs = make(map[string]Codec)
	serverCodecs = make(map[string]Codec)
	lock         sync.RWMutex
)

// Register defines the logic of register an codec by name. It will be
// called by init function defined by third package. If there is no server codec,
// the second param serverCodec can be nil.
func Register(name string, serverCodec Codec, clientCodec Codec) {
	lock.Lock()
	serverCodecs[name] = serverCodec
	clientCodecs[name] = clientCodec
	lock.Unlock()
}

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
