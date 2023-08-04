package codec

import (
	"errors"
)

// Compressor is body compress and decompress interface.
type Compressor interface {
	Compress(in []byte) (out []byte, err error)
	Decompress(in []byte) (out []byte, err error)
}

// CompressType is the mode of body compress or decompress.
const (
	CompressTypeNoop = iota
	CompressTypeGzip
	CompressTypeSnappy
	CompressTypeZlib
	CompressTypeStreamSnappy
	CompressTypeBlockSnappy
)

var compressors = make(map[int]Compressor)

// RegisterCompressor register a specific compressor, which will
// be called by init function defined in third package.
func RegisterCompressor(compressType int, s Compressor) {
	compressors[compressType] = s
}

// GetCompressor returns a specific compressor by type.
func GetCompressor(compressType int) Compressor {
	return compressors[compressType]
}

// Compress returns the compressed data, the data is compressed
// by a specific compressor.
func Compress(compressorType int, in []byte) ([]byte, error) {
	if len(in) == 0 {
		return nil, nil
	}
	compressor := GetCompressor(compressorType)
	if compressor == nil {
		return nil, errors.New("compressor not registered")
	}
	return compressor.Compress(in)
}

// Decompress returns the decompressed data, the data is decompressed
// by a specific compressor.
func Decompress(compressorType int, in []byte) ([]byte, error) {
	if len(in) == 0 {
		return nil, nil
	}
	compressor := GetCompressor(compressorType)
	if compressor == nil {
		return nil, errors.New("compressor not registered")
	}
	return compressor.Decompress(in)
}
