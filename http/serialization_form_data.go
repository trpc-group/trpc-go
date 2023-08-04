package http

import (
	"errors"
	"net/url"

	"trpc.group/trpc-go/trpc-go/codec"
)

var (
	// tagJSON uses same tag with json.
	tagJSON = "json"
	// FormDataMarshalType the serialization method of the response data,
	// default is json serialization.
	FormDataMarshalType = codec.SerializationTypeJSON
)

func init() {
	codec.RegisterSerializer(
		codec.SerializationTypeFormData,
		NewFormDataSerialization(tagJSON),
	)
}

// getFormDataContentType returns content type for form data.
func getFormDataContentType() string {
	return serializationTypeContentType[FormDataMarshalType]
}

// NewFormDataSerialization initializes from serialized object.
func NewFormDataSerialization(tag string) codec.Serializer {
	return &FormDataSerialization{
		tagName: tag,
	}
}

// FormDataSerialization packages kv structure of http request.
type FormDataSerialization struct {
	tagName string
}

// Unmarshal unpacks kv structure.
func (j *FormDataSerialization) Unmarshal(in []byte, body interface{}) error {
	values, err := url.ParseQuery(string(in))
	if err != nil {
		return err
	}
	return unmarshalValues(j.tagName, values, body)
}

// Marshal serializes.
func (j *FormDataSerialization) Marshal(body interface{}) ([]byte, error) {
	serializer := codec.GetSerializer(FormDataMarshalType)
	if serializer == nil {
		return nil, errors.New("empty json serializer")
	}
	return serializer.Marshal(body)
}
