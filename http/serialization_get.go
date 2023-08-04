package http

import (
	"errors"
	"net/url"

	"trpc.group/trpc-go/trpc-go/codec"
)

func init() {
	codec.RegisterSerializer(codec.SerializationTypeGet, NewGetSerialization(tag))
}

// NewGetSerialization initializes the get serialized object.
func NewGetSerialization(tag string) codec.Serializer {

	return &GetSerialization{
		tagname: tag,
	}
}

// GetSerialization packages kv structure of the http get request.
type GetSerialization struct {
	tagname string
}

// Unmarshal unpacks kv structure.
func (s *GetSerialization) Unmarshal(in []byte, body interface{}) error {
	values, err := url.ParseQuery(string(in))
	if err != nil {
		return err
	}
	return unmarshalValues(s.tagname, values, body)
}

// Marshal packages kv structure.
func (s *GetSerialization) Marshal(body interface{}) ([]byte, error) {
	jsonSerializer := codec.GetSerializer(codec.SerializationTypeJSON)
	if jsonSerializer == nil {
		return nil, errors.New("empty json serializer")
	}
	return jsonSerializer.Marshal(body)
}
