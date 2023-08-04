// Package ring provides a concurrent-safe circular queue, supports multiple read/write.
package ring

import (
	"errors"
	"fmt"
	"runtime"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/cpu"
)

const (
	// cacheLinePadSize is the size of CPU cache line.
	cacheLinePadSize = unsafe.Sizeof(cpu.CacheLinePad{})
)

var (
	// ErrQueueFull happens when the queue is full.
	ErrQueueFull = errors.New("queue is full")
)

type ringItem[T any] struct {
	putSeq uint32 // sequence number expected to put.
	getSeq uint32 // sequence number expected to get.
	value  T
	_      [cacheLinePadSize - 8 - 16]byte
}

// Ring a concurrent-safe circular queue, based on the idea of Disruptor(simplified).
// https://lmax-exchange.github.io/disruptor/disruptor.html
type Ring[T any] struct {
	capacity uint32 // the capacity of circular queue, including Empty element, must be the power of 2.
	mask     uint32 // capacity mask of the circular queue.
	_        [cacheLinePadSize - 8]byte
	head     uint32 // the latest sequence number having been read.
	_        [cacheLinePadSize - 4]byte
	tail     uint32 // the latest sequence number having been written.
	_        [cacheLinePadSize - 4]byte
	data     []ringItem[T] // elements in the queue.
	_        [cacheLinePadSize - unsafe.Sizeof([]ringItem[T]{})]byte
}

// New creates a circular queue.
func New[T any](capacity uint32) *Ring[T] {
	capacity = roundUpToPower2(capacity)
	if capacity < 2 {
		capacity = 2
	}

	r := &Ring[T]{
		capacity: capacity,
		mask:     capacity - 1,
		data:     make([]ringItem[T], capacity),
	}
	// initialize every slot with read/write sequence number.
	for i := range r.data {
		r.data[i].getSeq = uint32(i)
		r.data[i].putSeq = uint32(i)
	}
	// starts from Index=1 to fill the package.
	r.data[0].getSeq = capacity
	r.data[0].putSeq = capacity
	return r
}

// Put puts element into the circular queue.
// Directly return if the queue is full.
func (r *Ring[T]) Put(val T) error {
	// acquire put sequence.
	seq, err := r.acquirePutSequence()
	if err != nil {
		return err
	}
	// write element.
	r.commit(seq, val)
	return nil
}

// Get gets an element from the circular queue, return element value and
// the remaining number of elements.
func (r *Ring[T]) Get() (T, uint32) {
	// acquire get sequence.
	head, size, left := r.acquireGetSequence(1)
	if size == 0 {
		var zero T
		return zero, 0
	}
	// read element.
	return r.consume(head), left
}

// Gets acquires elements from the circular queue and appends then into v,
// return the number of elements acquired and remained.
func (r *Ring[T]) Gets(val *[]T) (uint32, uint32) {
	// batch acquire get sequence.
	head, size, left := r.acquireGetSequence(uint32(cap(*val) - len(*val)))
	if size == 0 {
		return 0, 0
	}
	// batch read the elements.
	for seq, i := head, uint32(0); i < size; seq, i = seq+1, i+1 {
		*val = append(*val, r.consume(seq))
	}
	return size, left
}

// Cap retrieves the number of elements the circular queue can hold.
func (r *Ring[T]) Cap() uint32 {
	// the capacity is represented by mask.
	return r.mask
}

// Size retrieves the number of elements in the circular queue.
func (r *Ring[T]) Size() uint32 {
	head := atomic.LoadUint32(&r.head)
	tail := atomic.LoadUint32(&r.tail)
	return r.quantity(head, tail)
}

// IsEmpty checks whether the queue is empty.
func (r *Ring[T]) IsEmpty() bool {
	head := atomic.LoadUint32(&r.head)
	tail := atomic.LoadUint32(&r.tail)
	return head == tail
}

// IsFull checks whether the queue is full.
func (r *Ring[T]) IsFull() bool {
	head := atomic.LoadUint32(&r.head)
	next := atomic.LoadUint32(&r.tail) + 1
	return next-head > r.mask
}

// String prints the structure of Ring.
func (r *Ring[T]) String() string {
	head := atomic.LoadUint32(&r.head)
	tail := atomic.LoadUint32(&r.tail)
	return fmt.Sprintf("Ring: Cap=%v, Head=%v, Tail=%v, Size=%v\n",
		r.Cap(), head, tail, r.Size())
}

func (r *Ring[T]) quantity(head, tail uint32) uint32 {
	return tail - head
}

func (r *Ring[T]) acquirePutSequence() (uint32, error) {
	var tail, head, next uint32
	mask := r.mask
	for {
		head = atomic.LoadUint32(&r.head)
		tail = atomic.LoadUint32(&r.tail)
		next = tail + 1
		left := r.quantity(head, next)
		// the queue is full, return.
		if left > mask {
			return 0, ErrQueueFull
		}
		// got the sequence number, return.
		if atomic.CompareAndSwapUint32(&r.tail, tail, next) {
			return next, nil
		}
		// fails to get the sequence number, yields the CPU
		// to reduce busy loop of CPU.
		runtime.Gosched()
	}
}

func (r *Ring[T]) acquireGetSequence(ask uint32) (uint32, uint32, uint32) {
	var tail, head, size uint32
	for {
		head = atomic.LoadUint32(&r.head)
		tail = atomic.LoadUint32(&r.tail)
		left := r.quantity(head, tail)
		// the queue is empty, return.
		if left < 1 {
			return head, 0, 0
		}
		size = left
		if ask < left {
			size = ask
		}
		// got the sequence number, return.
		if atomic.CompareAndSwapUint32(&r.head, head, head+size) {
			return head + 1, size, left - size
		}
		// fails to get the sequence number, yields the CPU
		// to reduce busy loop of CPU.
		runtime.Gosched()
	}
}

func (r *Ring[T]) commit(seq uint32, val T) {
	item := &r.data[seq&r.mask]
	for {
		getSeq := atomic.LoadUint32(&item.getSeq)
		putSeq := atomic.LoadUint32(&item.putSeq)
		// Waiting for data to be ready for writing. Due to the separation of
		// obtaining the right to use the sequence number and reading and writing
		// data operations, there is a short period of time that the old data has
		// not been read, wait for the read operation to complete and set getSeq.
		if seq == putSeq && getSeq == putSeq {
			break
		}
		runtime.Gosched()
	}
	// Complete the write operation and set putSeq to the next expected write sequence number.
	item.value = val
	atomic.AddUint32(&item.putSeq, r.capacity)
}

func (r *Ring[T]) consume(seq uint32) T {
	item := &r.data[seq&r.mask]
	for {
		getSeq := atomic.LoadUint32(&item.getSeq)
		putSeq := atomic.LoadUint32(&item.putSeq)
		// Waiting for data to be ready to read. Due to the separation of
		// obtaining the right to use the sequence number and reading and writing
		// data operations, there is a short period of time that the writing data has
		// not been written yet, wait for the writing operation to complete and set putSeq.
		if seq == getSeq && getSeq == (putSeq-r.capacity) {
			break
		}
		runtime.Gosched()
	}
	// Complete the read operation and set getSeq to the next expected read sequence number.
	val := item.value
	var zero T
	item.value = zero
	atomic.AddUint32(&item.getSeq, r.capacity)
	return val
}

// roundUpToPower2 rounds the integer up to the Nth power of 2.
func roundUpToPower2(v uint32) uint32 {
	v--
	v |= v >> 1
	v |= v >> 2
	v |= v >> 4
	v |= v >> 8
	v |= v >> 16
	v++
	return v
}
