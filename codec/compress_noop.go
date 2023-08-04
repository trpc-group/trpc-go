package codec

func init() {
	RegisterCompressor(CompressTypeNoop, &NoopCompress{})
}

// NoopCompress is an empty compressor
type NoopCompress struct {
}

// Compress returns the origin data.
func (c *NoopCompress) Compress(in []byte) ([]byte, error) {
	return in, nil
}

// Decompress returns the origin data.
func (c *NoopCompress) Decompress(in []byte) ([]byte, error) {
	return in, nil
}
