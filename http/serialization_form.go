package http

import (
	"fmt"
	"net/url"

	"trpc.group/trpc-go/trpc-go/codec"

	"github.com/go-playground/form/v4"
	"github.com/mitchellh/mapstructure"
)

// Uses the same tag as json.
var tag = "json"

func init() {
	codec.RegisterSerializer(
		codec.SerializationTypeForm,
		NewFormSerialization(tag),
	)
}

// NewFormSerialization initializes the form serialized object.
func NewFormSerialization(tag string) codec.Serializer {
	encoder := form.NewEncoder()
	encoder.SetTagName(tag)
	decoder := form.NewDecoder()
	decoder.SetTagName(tag)
	return &FormSerialization{
		tagname: tag,
		encoder: encoder,
		decoder: decoder,
	}
}

// FormSerialization packages the kv structure of http get request.
type FormSerialization struct {
	tagname string
	encoder *form.Encoder
	decoder *form.Decoder
}

// Unmarshal unpacks kv structure.
func (j *FormSerialization) Unmarshal(in []byte, body interface{}) error {
	values, err := url.ParseQuery(string(in))
	if err != nil {
		return err
	}
	switch body.(type) {
	// go-playground/form does not support map structure.
	case map[string]interface{}, *map[string]interface{}, map[string]string, *map[string]string:
		return unmarshalValues(j.tagname, values, body)
	default:
	}
	// First try using go-playground/form, it can handle nested struct.
	// But it cannot handle Chinese characters in byte slice.
	err = j.decoder.Decode(body, values)
	if err == nil {
		return nil
	}
	// Second try using mapstructure.
	if e := unmarshalValues(j.tagname, values, body); e != nil {
		return fmt.Errorf("unmarshal error: first try err = %+v, second try err = %w", err, e)
	}
	return nil
}

// unmarshalValues parses the corresponding fields in values according to tagname.
func unmarshalValues(tagname string, values url.Values, body interface{}) error {
	params := map[string]interface{}{}
	for k, v := range values {
		if len(v) == 1 {
			params[k] = v[0]
		} else {
			params[k] = v
		}
	}
	config := &mapstructure.DecoderConfig{TagName: tagname, Result: body, WeaklyTypedInput: true, Metadata: nil}
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}
	return decoder.Decode(params)
}

// Marshal packages kv structure.
func (j *FormSerialization) Marshal(body interface{}) ([]byte, error) {
	if req, ok := body.(url.Values); ok { // Used to send form urlencode post request to backend.
		return []byte(req.Encode()), nil
	}
	val, err := j.encoder.Encode(body)
	if err != nil {
		return nil, err
	}
	return []byte(val.Encode()), nil
}
