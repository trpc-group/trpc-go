package linkbuffer

import "io"

// Buffer is the interface of link buffer.
type Buffer interface {
	Reader
	Writer
	Release()
}

// Reader is the interface to read from link buffer.
type Reader interface {
	io.Reader
	ReadN(size int) ([]byte, int)
	ReadAll() [][]byte
	ReadNext() []byte
}

// Writer is the interface to write to link buffer.
type Writer interface {
	io.Writer
	Append(...[]byte)
	Prepend(...[]byte)
	Alloc(size int) []byte
	Prelloc(size int) []byte
	Len() int
	Merge(Reader)
}
