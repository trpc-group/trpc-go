package codec

import (
	"encoding/xml"
)

func init() {
	RegisterSerializer(SerializationTypeXML, &XMLSerialization{})
	RegisterSerializer(SerializationTypeTextXML, &XMLSerialization{})
}

// XMLSerialization provides xml serialization mode.
type XMLSerialization struct{}

// Unmarshal deserializes the in bytes into body.
func (*XMLSerialization) Unmarshal(in []byte, body interface{}) error {
	return xml.Unmarshal(in, body)
}

// Marshal returns the serialized bytes in xml protocol.
func (*XMLSerialization) Marshal(body interface{}) ([]byte, error) {
	return xml.Marshal(body)
}
