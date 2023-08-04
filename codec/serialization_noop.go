package codec

import (
	"errors"
	"fmt"
)

func init() {
	RegisterSerializer(SerializationTypeNoop, &NoopSerialization{})
}

// BytesBodyIn is used to check if Body implements BytesBodyIn
// method when compile.
var _ BytesBodyIn = &Body{}

// BytesBodyIn is used to check if Body implements BytesBodyOut
// method when compile.
var _ BytesBodyOut = &Body{}

// BytesBodyOut is used to receive custom type body.
type BytesBodyOut interface {
	Bytes() ([]byte, error)
}

// BytesBodyIn is used to receive custom type body.
type BytesBodyIn interface {
	SetBytes([]byte) error
}

// Body is bytes pack layer, it is not need serialized
// and used in gateway service generally.
type Body struct {
	Data []byte
}

// String returns body data as string.
func (b *Body) String() string {
	return fmt.Sprintf("%v", b.Data)
}

// SetBytes sets body data and implements ByteBodyIn interface.
func (b *Body) SetBytes(p []byte) error {
	if b == nil {
		return errors.New("body nil")
	}
	b.Data = p
	return nil
}

// Bytes returns body data and implements ByteBodyOut interface.
func (b *Body) Bytes() ([]byte, error) {
	if b == nil {
		return nil, errors.New("body nil")
	}
	return b.Data, nil
}

// NoopSerialization provides empty serialization, it is
// used to serialize bytes.
type NoopSerialization struct {
}

// Unmarshal deserializes the in bytes into body, body should be a Body or implements
// BytesBodyIn interface.
func (s *NoopSerialization) Unmarshal(in []byte, body interface{}) error {
	bytesBodyIn, ok := body.(BytesBodyIn)
	if ok {
		return bytesBodyIn.SetBytes(in)
	}
	noop, ok := body.(*Body)
	if !ok {
		return errors.New("body type invalid")
	}
	if noop == nil {
		return errors.New("body nil")
	}
	noop.Data = in
	return nil
}

// Marshal returns the serialized bytes. body should be a Body or implements
// BytesBodyOut interface.
func (s *NoopSerialization) Marshal(body interface{}) ([]byte, error) {
	bytesBody, ok := body.(BytesBodyOut)
	if ok {
		return bytesBody.Bytes()
	}
	noop, ok := body.(*Body)
	if !ok {
		return nil, errors.New("body type invalid")
	}
	if noop == nil {
		return nil, errors.New("body nil")
	}
	return noop.Data, nil
}
