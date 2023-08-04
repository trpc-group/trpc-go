// Package linkbuffer is a rich buffer to reuse underlying bytes.
package linkbuffer

import (
	"io"
	"sync"
)

var _ Buffer = (*Buf)(nil)

// NewBuf creates a new buf.
func NewBuf(a Allocator, minMallocSize int) *Buf {
	bytes := newBytes(nil, nil)
	return &Buf{
		a:             a,
		minMallocSize: minMallocSize,
		head:          bytes,
		tail:          bytes,
	}
}

// Buf is a rich buffer to reuse underlying bytes.
type Buf struct {
	a             Allocator
	minMallocSize int

	head, tail *bytes
	dirty      *bytes
}

// Write copies p to Buf and implements io.Writer.
func (b *Buf) Write(p []byte) (int, error) {
	if b.tail.release == nil {
		bts, release := b.a.Malloc(b.minMallocSize)
		b.tail.next = newBytes(bts[:0], release)
		b.tail = b.tail.next
	}
	available := cap(b.tail.bts) - len(b.tail.bts)
	if available >= len(p) {
		b.tail.bts = append(b.tail.bts, p...)
		return len(p), nil
	}
	b.tail.bts = append(b.tail.bts, p[:available]...)
	bts, release := b.a.Malloc(b.minMallocSize)
	b.tail.next = newBytes(bts[:0], release)
	b.tail = b.tail.next
	n, err := b.Write(p[available:])
	return available + n, err
}

// Append appends a slice of bytes to Buf.
// Buf owns these bs, but won't release them to underlying allocator.
func (b *Buf) Append(bs ...[]byte) {
	for _, bts := range bs {
		b.append(bts)
	}
}

func (b *Buf) append(bts []byte) {
	if b.tail.release == nil || cap(b.tail.bts) == len(b.tail.bts) {
		b.tail.next = newBytes(bts, nil)
		b.tail = b.tail.next
	} else {
		remains := b.tail.bts[len(b.tail.bts):]
		release := b.tail.release
		b.tail.release = nil
		b.tail.next = newBytes(bts, nil)
		b.tail = b.tail.next
		b.tail.next = newBytes(remains, release)
		b.tail = b.tail.next
	}
}

// Prepend prepends a slice to bytes to Buf. Next Read starts with the first bytes of slice.
// Buf owns these bs, but won't release them to underlying allocator.
func (b *Buf) Prepend(bs ...[]byte) {
	for i := len(bs) - 1; i >= 0; i-- {
		bytes := newBytes(bs[i], nil)
		bytes.next = b.head
		b.head = bytes
	}
}

// Alloc allocates a []byte with size n.
func (b *Buf) Alloc(n int) []byte {
	if b.tail.release != nil && cap(b.tail.bts)-len(b.tail.bts) >= n {
		l := len(b.tail.bts)
		b.tail.bts = b.tail.bts[:l+n]
		return b.tail.bts[l : l+n]
	}
	bts, release := b.a.Malloc(n)
	b.tail.next = newBytes(bts[:n], release)
	b.tail = b.tail.next
	return bts[:n]
}

// Prelloc allocates a []byte with size n at the beginning of Buf.
func (b *Buf) Prelloc(n int) []byte {
	bts, release := b.a.Malloc(n)
	bytes := newBytes(bts[:n], release)
	bytes.next = b.head
	b.head = bytes
	return bts[:n]
}

// Merge merges another Reader.
// If r is not *Buf, b does not own the bytes of r.
// If r is a *Buf, the ownership of r's bytes is changed to b, and the caller should not Release r.
func (b *Buf) Merge(r Reader) {
	bb, ok := r.(*Buf)
	if !ok {
		for _, bts := range r.ReadAll() {
			b.Append(bts)
		}
		return
	}
	b.tail.next = bb.head
	b.tail = bb.tail
}

// Read copies data to p, and returns the number of byte copied and an error.
// The io.EOF is returned if Buf has no unread bytes and len(p) is not zero.
func (b *Buf) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	defer b.ensureNotEmpty()
	var copied int
	for b.head != nil {
		curCopied := copy(p[copied:], b.head.bts)
		copied += curCopied
		b.head.bts = b.head.bts[curCopied:]
		b.dirtyEmptyHeads()
		if copied == len(p) {
			return copied, nil
		}
	}
	if copied > 0 {
		return copied, nil
	}
	return copied, io.EOF
}

// ReadN tries best to read all size into one []byte.
// The second return value may be smaller than size if underlying bytes is not continuous.
func (b *Buf) ReadN(size int) ([]byte, int) {
	defer b.ensureNotEmpty()
	b.dirtyEmptyHeads()
	for b.head != nil {
		if size >= len(b.head.bts) {
			bts := b.dirtyHead()
			b.dirtyEmptyHeads()
			return bts, len(bts)
		}
		bts := b.head.bts[:size]
		b.head.bts = b.head.bts[size:]
		return bts, size
	}
	return nil, 0
}

// ReadAll returns all underlying []byte in [][]byte.
func (b *Buf) ReadAll() [][]byte {
	defer b.ensureNotEmpty()
	var all [][]byte
	for b.head != nil {
		if bts := b.dirtyHead(); len(bts) != 0 {
			all = append(all, bts)
		}
	}
	return all
}

// ReadNext returns the next continuous []byte.
func (b *Buf) ReadNext() []byte {
	defer b.ensureNotEmpty()
	for b.head != nil {
		if bts := b.dirtyHead(); len(bts) != 0 {
			return bts
		}
	}
	return nil
}

// Release releases the read bytes to allocator.
func (b *Buf) Release() {
	for b.dirty != nil {
		b.a.Free(b.dirty.release)
		dirty := b.dirty
		b.dirty = b.dirty.next
		dirty.release = nil
		dirty.bts = nil
		dirty.next = nil
		bytesPool.Put(dirty)
	}
}

// Len returns the total len of underlying bytes.
func (b *Buf) Len() int {
	var l int
	for bytes := b.head; bytes != nil; bytes = bytes.next {
		l += len(bytes.bts)
	}
	return l
}

func (b *Buf) dirtyEmptyHeads() {
	for b.head != nil && len(b.head.bts) == 0 {
		b.dirtyHead()
	}
}

func (b *Buf) dirtyHead() []byte {
	bts := b.head.bts
	head := b.head
	b.head = head.next
	if head.release == nil {
		head.bts = nil
		head.next = nil
		bytesPool.Put(head)
	} else {
		head.next = b.dirty
		b.dirty = head
	}
	return bts
}

func (b *Buf) ensureNotEmpty() {
	if b.head == nil {
		b.head = newBytes(nil, nil)
		b.tail = b.head
	}
}

// Allocator is the interface to Malloc or Free bytes.
type Allocator interface {
	// Malloc mallocs a []byte with specific size.
	// The second return value is the consequence for go's escape analysis.
	// See ClassAllocator and https://github.com/golang/go/issues/8618 for details.
	Malloc(int) ([]byte, interface{})
	// Free frees the allocated bytes. It accepts the second return value of Malloc.
	Free(interface{})
}

type bytes struct {
	bts     []byte
	release interface{}
	next    *bytes
}

var bytesPool = sync.Pool{New: func() interface{} { return &bytes{} }}

func newBytes(bts []byte, release interface{}) *bytes {
	bytes := bytesPool.Get().(*bytes)
	bytes.bts = bts
	bytes.release = release
	bytes.next = nil
	return bytes
}
