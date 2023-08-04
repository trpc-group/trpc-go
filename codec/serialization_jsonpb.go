package codec

import (
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterSerializer(SerializationTypeJSON, &JSONPBSerialization{})
}

// Marshaler is jsonpb marshal object, users can change its params.
var Marshaler = protojson.MarshalOptions{EmitUnpopulated: true, UseProtoNames: true, UseEnumNumbers: true}

// Unmarshaler is jsonpb unmarshal object, users can chang its params.
var Unmarshaler = protojson.UnmarshalOptions{DiscardUnknown: false}

// JSONPBSerialization provides jsonpb serialization mode. It is based on
// protobuf/jsonpb.
type JSONPBSerialization struct{}

// Unmarshal deserialize the in bytes into body.
func (s *JSONPBSerialization) Unmarshal(in []byte, body interface{}) error {
	input, ok := body.(proto.Message)
	if !ok {
		return JSONAPI.Unmarshal(in, body)
	}
	return Unmarshaler.Unmarshal(in, input)
}

// Marshal returns the serialized bytes in jsonpb protocol.
func (s *JSONPBSerialization) Marshal(body interface{}) ([]byte, error) {
	input, ok := body.(proto.Message)
	if !ok {
		return JSONAPI.Marshal(body)
	}
	return Marshaler.Marshal(input)
}
