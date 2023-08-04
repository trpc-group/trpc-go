package codec

import (
	"errors"

	"trpc.group/trpc-go/jce"
)

func init() {
	RegisterSerializer(SerializationTypeJCE, &JCESerialization{})
}

// JCESerialization provides jce serialization mode.
type JCESerialization struct{}

// Unmarshal deserializes in bytes into body, body should implement
// jce.Message interface.
func (j *JCESerialization) Unmarshal(in []byte, body interface{}) error {
	if _, ok := body.(jce.Message); !ok {
		return errors.New("not jce.Message")
	}
	return jce.Unmarshal(in, body.(jce.Message))
}

// Marshal returns the bytes serialized in jce protocol.
func (j *JCESerialization) Marshal(body interface{}) ([]byte, error) {
	if _, ok := body.(jce.Message); !ok {
		return nil, errors.New("not jce.Message")
	}
	return jce.Marshal(body.(jce.Message))
}
