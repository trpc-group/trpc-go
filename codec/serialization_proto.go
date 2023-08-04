package codec

import (
	"errors"

	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterSerializer(SerializationTypePB, &PBSerialization{})
}

// PBSerialization provides protobuf serialization mode.
type PBSerialization struct{}

// Unmarshal deserializes the in bytes into body.
func (s *PBSerialization) Unmarshal(in []byte, body interface{}) error {
	msg, ok := body.(proto.Message)
	if !ok {
		return errors.New("unmarshal fail: body not protobuf message")
	}
	return proto.Unmarshal(in, msg)
}

// Marshal returns the serialized bytes in protobuf protocol.
func (s *PBSerialization) Marshal(body interface{}) ([]byte, error) {
	msg, ok := body.(proto.Message)
	if !ok {
		return nil, errors.New("marshal fail: body not protobuf message")
	}
	return proto.Marshal(msg)
}
