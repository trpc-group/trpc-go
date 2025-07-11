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

// GetCompressor returns a Compressor by name.
func GetCompressor(name string) Compressor {
	return compressors[name]
}

// compressorForTranscoding returns inbound/outbound Compressors for transcoding.
func compressorForTranscoding(contentEncodings []string, acceptEncodings []string) (Compressor, Compressor) {
	var reqCompressor, respCompressor Compressor // both could be nil

	for _, contentEncoding := range contentEncodings {
		if c, ok := compressors[contentEncoding]; ok {
			reqCompressor = c
			break
		}
	}

	for _, acceptEncoding := range acceptEncodings {
		if c, ok := compressors[acceptEncoding]; ok {
			respCompressor = c
			break
		}
	}

	return reqCompressor, respCompressor
}
