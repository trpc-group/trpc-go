package linkbuffer_test

import (
	stdbytes "bytes"
	"io"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
	"trpc.group/trpc-go/trpc-go/internal/allocator"
	. "trpc.group/trpc-go/trpc-go/internal/linkbuffer"
)

func BenchmarkBuf(b *testing.B) {
	bigBts := make([]byte, 1<<10)
	b.Run("link_buffer_bigBytes", func(b *testing.B) {
		b.ReportAllocs()
		bb := NewBuf(allocator.NewClassAllocator(), 1<<9)
		for i := 0; i < b.N; i++ {
			bb.Append(bigBts)
			bb.ReadNext()
			bb.Release()
		}
	})
	b.Run("copy_each_bigBytes", func(b *testing.B) {
		b.ReportAllocs()
		var bb []byte
		for i := 0; i < b.N; i++ {
			bb = append(bb, bigBts...)
		}
	})
	b.Run("link_buffer_reuse", func(b *testing.B) {
		b.ReportAllocs()
		r := rand.New(rand.NewSource(1))
		bb := NewBuf(allocator.NewClassAllocator(), 1<<10)
		for i := 0; i < b.N; i++ {
			copy(bb.Alloc(16), bigBts)
			copy(bb.Alloc(int(r.Int31()%1<<20+1)), bigBts)
			bb.ReadNext()
			bb.ReadNext()
			bb.Release()
		}
	})
	b.Run("std_buffer", func(b *testing.B) {
		b.ReportAllocs()
		r := rand.New(rand.NewSource(1))
		for i := 0; i < b.N; i++ {
			bb := stdbytes.Buffer{}
			bb.Write(bigBts[:16])
			bb.Write(bigBts[:r.Int31()%1<<20+1])
		}
	})
	b.Run("bytes_cannot_reuse", func(b *testing.B) {
		b.ReportAllocs()
		r := rand.New(rand.NewSource(1))
		for i := 0; i < b.N; i++ {
			bts := make([]byte, r.Int31()%1<<20+16)
			copy(bts[:16], bigBts)
			copy(bts[16:], bigBts)
		}
	})
}

func TestBuf_Write(t *testing.T) {
	wa := newWrappedAllocator(t, allocator.NewClassAllocator())
	b := NewBuf(wa, 4)

	n, err := b.Write([]byte("123"))
	require.Nil(t, err)
	require.Equal(t, 3, n)

	n, err = b.Write([]byte("45"))
	require.Nil(t, err)
	require.Equal(t, 2, n)

	wa.MustMallocTimes(2)
}

func TestBuf_Read(t *testing.T) {
	wa := newWrappedAllocator(t, allocator.NewClassAllocator())
	b := NewBuf(wa, 4)

	n, err := b.Write([]byte("1234567890"))
	require.Nil(t, err)
	require.Equal(t, 10, n)

	bts := make([]byte, 3)
	n, err = b.Read(bts)
	require.Nil(t, err)
	require.Equal(t, 3, n)
	require.Equal(t, "123", string(bts))

	n, err = b.Read(bts)
	require.Nil(t, err)
	n, err = b.Read(bts)
	require.Nil(t, err)

	n, err = b.Read(bts)
	require.Nil(t, err)
	require.Equal(t, 1, n)
	require.Equal(t, "089", string(bts))

	n, err = b.Read(bts)
	require.ErrorIs(t, err, io.EOF)
	require.Equal(t, 0, n)

	b.Release()
	wa.MustMallocTimes(3)
	wa.MustAllFreed()
}

func TestBuf_Append(t *testing.T) {
	wa := newWrappedAllocator(t, allocator.NewClassAllocator())
	b := NewBuf(wa, 4)

	n, err := b.Write([]byte("12"))
	require.Nil(t, err)
	require.Equal(t, 2, n)

	b.Append([]byte("345"))

	// the remaining two bytes is available, no need to malloc new bytes.
	n, err = b.Write([]byte("67"))
	require.Nil(t, err)
	require.Equal(t, 2, n)

	bts := make([]byte, 7)
	n, err = b.Read(bts)
	require.Nil(t, err)
	require.Equal(t, 7, n)
	require.Equal(t, "1234567", string(bts))

	b.Release()
	wa.MustMallocTimes(1)
	wa.MustAllFreed()
}

func TestBuf_Prepend(t *testing.T) {
	wa := newWrappedAllocator(t, allocator.NewClassAllocator())
	b := NewBuf(wa, 4)

	n, err := b.Write([]byte("12"))
	require.Nil(t, err)
	require.Equal(t, 2, n)

	b.Prepend([]byte("345"))

	bts := make([]byte, 5)
	n, err = b.Read(bts)
	require.Nil(t, err)
	require.Equal(t, 5, n)
	require.Equal(t, "34512", string(bts))

	b.Release()
	wa.MustMallocTimes(1)
	wa.MustAllFreed()
}

func TestBuf_Alloc(t *testing.T) {
	wa := newWrappedAllocator(t, allocator.NewClassAllocator())
	b := NewBuf(wa, 4)

	n, err := b.Write([]byte("12"))
	require.Nil(t, err)
	require.Equal(t, 2, n)

	bts := b.Alloc(1)
	require.Len(t, bts, 1)
	n, err = b.Write([]byte("456"))
	require.Nil(t, err)
	require.Equal(t, 3, n)
	bts[0] = '3'

	bts = b.Alloc(3)
	require.Len(t, bts, 3)
	copy(bts, "789")

	bts = make([]byte, 9)
	n, err = b.Read(bts)
	require.Nil(t, err)
	require.Equal(t, 9, n)
	require.Equal(t, "123456789", string(bts))

	b.Release()
	wa.MustMallocTimes(3)
	wa.MustAllFreed()
}

func TestBuf_Prelloc(t *testing.T) {
	wa := newWrappedAllocator(t, allocator.NewClassAllocator())
	b := NewBuf(wa, 4)

	n, err := b.Write([]byte("12"))
	require.Nil(t, err)
	require.Equal(t, 2, n)

	bts := b.Prelloc(1)
	require.Len(t, bts, 1)
	b.Prepend([]byte("456"))
	bts[0] = '3'

	bts = b.Prelloc(3)
	require.Len(t, bts, 3)
	copy(bts, "789")

	bts = make([]byte, 9)
	n, err = b.Read(bts)
	require.Nil(t, err)
	require.Equal(t, 9, n)
	require.Equal(t, "789456312", string(bts))

	b.Release()
	wa.MustMallocTimes(3)
	wa.MustAllFreed()
}

func TestBuf_Merge(t *testing.T) {
	wa := newWrappedAllocator(t, allocator.NewClassAllocator())

	b1 := NewBuf(wa, 4)
	b1.Append([]byte("123"))
	n, err := b1.Write([]byte("456"))
	require.Nil(t, err)
	require.Equal(t, 3, n)

	b2 := NewBuf(wa, 2)
	n, err = b2.Write([]byte("567"))
	require.Nil(t, err)
	require.Equal(t, 3, n)
	b2.Append([]byte("89"))

	bts := make([]byte, 2)
	n, err = b2.Read(bts)
	require.Nil(t, err)
	require.Equal(t, 2, n)
	require.Equal(t, "56", string(bts))
	b2.Release()

	b1.Merge(b2)
	bts = make([]byte, 9)
	n, err = b1.Read(bts)
	require.Nil(t, err)
	require.Equal(t, 9, n)
	require.Equal(t, "123456789", string(bts))

	b1.Release()
	wa.MustMallocTimes(1 + 2)
	wa.MustAllFreed()
}

func TestBuf_ReadN(t *testing.T) {
	wa := newWrappedAllocator(t, allocator.NewClassAllocator())
	b := NewBuf(wa, 4)

	n, err := b.Write([]byte("123"))
	require.Nil(t, err)
	require.Equal(t, 3, n)
	b.Append([]byte("456"))

	bts, n := b.ReadN(1)
	require.Equal(t, 1, n)
	require.Equal(t, "1", string(bts))

	bts, n = b.ReadN(3)
	require.Equal(t, 2, n)
	require.Equal(t, "23", string(bts))

	bts, n = b.ReadN(3)
	require.Equal(t, 3, n)
	require.Equal(t, "456", string(bts))

	bts, n = b.ReadN(3)
	require.Equal(t, 0, n)
	require.Nil(t, bts)

	b.Release()
	wa.MustMallocTimes(1)
	wa.MustAllFreed()
}

func TestBuf_ReadAll(t *testing.T) {
	wa := newWrappedAllocator(t, allocator.NewClassAllocator())
	b := NewBuf(wa, 4)

	n, err := b.Write([]byte("123"))
	require.Nil(t, err)
	require.Equal(t, 3, n)
	b.Append([]byte("45"))
	n, err = b.Write([]byte("67890"))
	require.Nil(t, err)
	require.Equal(t, 5, n)

	bs := b.ReadAll()
	require.Len(t, bs, 4)
	require.Equal(t, "123", string(bs[0]))
	require.Equal(t, "45", string(bs[1]))
	require.Equal(t, "6", string(bs[2]))
	require.Equal(t, "7890", string(bs[3]))

	b.Release()
	wa.MustMallocTimes(2)
	wa.MustAllFreed()
}

func TestBuf_ReadNext(t *testing.T) {
	wa := newWrappedAllocator(t, allocator.NewClassAllocator())
	b := NewBuf(wa, 4)

	n, err := b.Write([]byte("123"))
	require.Nil(t, err)
	require.Equal(t, 3, n)
	b.Append([]byte("45"))
	n, err = b.Write([]byte("678"))
	require.Nil(t, err)
	require.Equal(t, 3, n)

	require.Equal(t, "123", string(b.ReadNext()))
	require.Equal(t, "45", string(b.ReadNext()))
	require.Equal(t, "6", string(b.ReadNext()))
	require.Equal(t, "78", string(b.ReadNext()))
	require.Equal(t, "", string(b.ReadNext()))

	b.Release()
	wa.MustMallocTimes(2)
	wa.MustAllFreed()
}

func TestBuf_WriteBigData(t *testing.T) {
	wa := newWrappedAllocator(t, allocator.NewClassAllocator())
	b := NewBuf(wa, 2)

	n, err := b.Write([]byte("123"))
	require.Nil(t, err)
	require.Equal(t, 3, n)

	copy(b.Alloc(3), "456")

	n, err = b.Write([]byte("78"))
	require.Nil(t, err)
	require.Equal(t, 2, n)

	bts := make([]byte, 8)
	n, err = b.Read(bts)
	require.Nil(t, err)
	require.Equal(t, 8, n)
	require.Equal(t, "12345678", string(bts))

	b.Release()
	wa.MustMallocTimes(2 + 1 + 1)
	wa.MustAllFreed()
}

func TestBuf_PrependAfterRead(t *testing.T) {
	wa := newWrappedAllocator(t, allocator.NewClassAllocator())
	b := NewBuf(wa, 4)

	b.Append([]byte("123"))

	n, err := b.Write([]byte("456"))
	require.Nil(t, err)
	require.Equal(t, 3, n)

	bts := make([]byte, 4)
	n, err = b.Read(bts)
	require.Nil(t, err)
	require.Equal(t, 4, n)
	require.Equal(t, "1234", string(bts))

	b.Prepend([]byte("34"))
	copy(b.Prelloc(2), "12")

	bts = make([]byte, 6)
	n, err = b.Read(bts)
	require.Nil(t, err)
	require.Equal(t, 6, n)
	require.Equal(t, "123456", string(bts))

	b.Release()
	wa.MustMallocTimes(1 + 1)
	wa.MustAllFreed()
}

func TestBuf_UseBytesAfterRelease(t *testing.T) {
	ba := newBytesAllocator()

	b1 := NewBuf(ba, 4)
	n, err := b1.Write([]byte("123"))
	require.Nil(t, err)
	require.Equal(t, 3, n)
	b1.Append([]byte("45"))
	n, err = b1.Write([]byte("678"))
	require.Nil(t, err)
	require.Equal(t, 3, n)

	bts := b1.ReadNext()
	require.Equal(t, "123", string(bts))
	b1.Release()

	b2 := NewBuf(ba, 4)
	n, err = b2.Write([]byte("1234"))
	require.Nil(t, err)
	require.Equal(t, 4, n)
	require.Equal(t, "123", string(bts), "bts of b1 is not released now")

	b1.ReadNext() // read "45"
	b1.Release()  // bts's underlying buffer is still not released

	n, err = b2.Write([]byte("5678"))
	require.Nil(t, err)
	require.Equal(t, 4, n)
	require.Equal(t, "123", string(bts), "bts of b1 is not released now")

	b1.ReadNext() // read "6"
	b1.Release()  // bts's underlying buffer has been released

	n, err = b2.Write([]byte("5678"))
	require.Nil(t, err)
	require.Equal(t, 4, n)
	require.Equal(t, "567", string(bts), "bts of b1 has been changed by b2")
}

func newWrappedAllocator(t *testing.T, a Allocator) *wrappedAllocator {
	return &wrappedAllocator{t: t, a: a, malloced: make(map[*byte]struct{})}
}

type wrappedAllocator struct {
	t           *testing.T
	a           Allocator
	malloced    map[*byte]struct{}
	mallocTimes int
}

func (a *wrappedAllocator) Malloc(size int) ([]byte, interface{}) {
	bts, free := a.a.Malloc(size)
	a.malloced[&bts[0]] = struct{}{}
	a.mallocTimes++
	return bts, free
}

func (a *wrappedAllocator) Free(bts interface{}) {
	a.a.Free(bts)
	if _, ok := a.malloced[&bts.([]byte)[0]]; !ok {
		require.FailNow(a.t, "free unknown bytes")
	}
	delete(a.malloced, &bts.([]byte)[0])
}

func (a *wrappedAllocator) MustMallocTimes(n int) {
	require.Equal(a.t, n, a.mallocTimes)
}

func (a *wrappedAllocator) MustAllFreed() {
	require.Empty(a.t, a.malloced)
}

func newBytesAllocator() *bytesAllocator {
	return &bytesAllocator{pool: make(map[int][]interface{})}
}

type bytesAllocator struct {
	pool map[int][]interface{}
}

func (a *bytesAllocator) Malloc(n int) ([]byte, interface{}) {
	bs, ok := a.pool[n]
	if ok && len(bs) != 0 {
		bts := bs[len(bs)-1]
		a.pool[n] = bs[:len(bs)-1]
		return bts.([]byte), bts
	}
	bts := make([]byte, n)
	return bts, bts
}

func (a *bytesAllocator) Free(v interface{}) {
	bts := v.([]byte)
	bts = bts[:cap(bts)]
	a.pool[len(bts)] = append(a.pool[len(bts)], v)
}
