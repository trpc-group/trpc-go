package codec

import (
	jsoniter "github.com/json-iterator/go"
)

// JSONAPI is json packing and unpacking object, users can change
// the internal parameter.
var JSONAPI = jsoniter.ConfigCompatibleWithStandardLibrary

// JSONSerialization provides json serialization mode.
// golang official json package is implemented by reflection,
// and has very low performance. So trpc-go choose json-iterator package
// to implement json serialization.
type JSONSerialization struct{}

// Unmarshal deserializes the in bytes into body.
func (s *JSONSerialization) Unmarshal(in []byte, body interface{}) error {
	return JSONAPI.Unmarshal(in, body)
}

// Marshal returns the serialized bytes in json protocol.
func (s *JSONSerialization) Marshal(body interface{}) ([]byte, error) {
	return JSONAPI.Marshal(body)
}
