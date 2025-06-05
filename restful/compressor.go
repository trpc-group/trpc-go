//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 THL A29 Limited, a Tencent company.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

package restful

import (
	"io"
	"net/http"
)

// Compressor is the interface for http body compression/decompression.
type Compressor interface {
	// Compress compresses http body.
	Compress(w io.Writer) (io.WriteCloser, error)
	// Decompress decompresses http body.
	Decompress(r io.Reader) (io.Reader, error)
	// Name returns name of the Compressor.
	Name() string
	// ContentEncoding returns the encoding indicated by Content-Encoding response header.
	ContentEncoding() string
}

// compressor related http headers
var (
	headerAcceptEncoding  = http.CanonicalHeaderKey("Accept-Encoding")
	headerContentEncoding = http.CanonicalHeaderKey("Content-Encoding")
)

var compressors = make(map[string]Compressor)

// RegisterCompressor registers a Compressor.
// This function is not thread-safe, it should only be called in init() function.
func RegisterCompressor(c Compressor) {
	if c == nil || c.Name() == "" {
		panic("tried to register nil or anonymous compressor")
	}
	compressors[c.Name()] = c
}

// MustRegisterCompressor registers a Compressor.
// This function is not thread-safe, it should only be called in init() function.
// It will panic if the compressor has been registered.
//
// In most cases, the framework uses the init + RegisterCompressor method for registration. However, due to
// the unpredictable execution order of init functions, some unknown situations may arise. For example:
//
// If your code uses init + MustRegisterCompressor to forcibly register a component 'xxx', while the framework
// uses init + RegisterCompressor to register another component 'yyy', conflicts may occur. If the init function
// for MustRegisterCompressor is executed before the conflicting init function, MustRegisterCompressor might not raise
// an error or panic as expected.
//
// Therefore, it's important to be cautious when using MustRegisterCompressor and to carefully consider any
// potential conflicts or unintended consequences that may arise from its use.
func MustRegisterCompressor(c Compressor) {
	if GetCompressor(c.Name()) != nil {
		panic("compressor already registered: " + c.Name())
	}
	RegisterCompressor(c)
}

// GetCompressor returns a Compressor by name.
func GetCompressor(name string) Compressor {
	return compressors[name]
}

func compressor(contentOrAcceptEncodings []string) Compressor {
	for _, contentOrAcceptEncoding := range contentOrAcceptEncodings {
		if c, ok := compressors[contentOrAcceptEncoding]; ok {
			return c
		}
	}
	return nil
}
