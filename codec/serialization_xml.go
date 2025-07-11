//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 Tencent.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

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
