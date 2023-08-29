// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package restful

import (
	"compress/gzip"
	"io"
	"sync"
)

func init() {
	RegisterCompressor(&GZIPCompressor{})
}

var readerPool sync.Pool
var writerPool sync.Pool

// GZIPCompressor is the compressor for Content-Encoding: gzip.
type GZIPCompressor struct{}

// wrappedWriter wraps gzip.Writer for pooling.
type wrappedWriter struct {
	*gzip.Writer
}

// Close rewrites the underlying gzip.Writer's Close method.
// The wrapped writer will be put back to the pool after its underlying gzip.Writer is closed.
func (w *wrappedWriter) Close() error {
	defer writerPool.Put(w)
	return w.Writer.Close()
}

// wrappedReader wraps gzip.Reader for pooling.
type wrappedReader struct {
	*gzip.Reader
}

// Read rewrites the underlying gzip.Reader's Read method.
// The wrapped reader will be put back to the pool after its underlying gzip.Reader is read.
func (r *wrappedReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	if err == io.EOF {
		readerPool.Put(r)
	}
	return n, err
}

// Compress implements Compressor.
func (*GZIPCompressor) Compress(w io.Writer) (io.WriteCloser, error) {
	z, ok := writerPool.Get().(*wrappedWriter)
	if !ok {
		z = &wrappedWriter{
			Writer: gzip.NewWriter(w),
		}
	}
	z.Writer.Reset(w)
	return z, nil
}

// Decompress implements Compressor.
func (g *GZIPCompressor) Decompress(r io.Reader) (io.Reader, error) {
	z, ok := readerPool.Get().(*wrappedReader)
	if !ok {
		gzipReader, err := gzip.NewReader(r)
		if err != nil {
			return nil, err
		}
		return &wrappedReader{
			Reader: gzipReader,
		}, nil
	}
	if err := z.Reader.Reset(r); err != nil {
		readerPool.Put(z)
		return nil, err
	}
	return z, nil
}

// Name implements Compressor.
func (*GZIPCompressor) Name() string {
	return "gzip"
}

// ContentEncoding implements Compressor.
func (*GZIPCompressor) ContentEncoding() string {
	return "gzip"
}
