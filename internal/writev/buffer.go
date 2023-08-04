// Package writev provides Buffer and uses the writev() system call to send packages.
package writev

import (
	"errors"
	"io"
	"net"
	"runtime"

	"trpc.group/trpc-go/trpc-go/internal/ring"
)

const (
	// default buffer queue length.
	defaultBufferSize = 128
	// The maximum number of data packets that can be sent by writev (from Go source code definition).
	maxWritevBuffers = 1024
)

var (
	// ErrAskQuit sends a close request externally.
	ErrAskQuit = errors.New("writev goroutine is asked to quit")
	// ErrStopped Buffer stops receiving data.
	ErrStopped = errors.New("writev buffer stop to receive data")
)

// QuitHandler Buffer goroutine exit handler.
type QuitHandler func(*Buffer)

// Buffer records the messages to be sent and sends them in batches using goroutines.
type Buffer struct {
	opts           *Options           // configuration items.
	w              io.Writer          // The underlying io.Writer that sends data.
	queue          *ring.Ring[[]byte] // queue for buffered messages.
	wakeupCh       chan struct{}      // used to wake up the sending goroutine.
	done           chan struct{}      // notify the sending goroutine to exit.
	err            error              // record error message.
	errCh          chan error         // internal error notification.
	isQueueStopped bool               // whether the cache queue stops receiving packets.
}

var defaultQuitHandler = func(b *Buffer) {
	b.SetQueueStopped(true)
}

// NewBuffer creates a Buffer and starts the sending goroutine.
func NewBuffer(opt ...Option) *Buffer {
	opts := &Options{
		bufferSize: defaultBufferSize,
		handler:    defaultQuitHandler,
	}
	for _, o := range opt {
		o(opts)
	}

	b := &Buffer{
		queue:    ring.New[[]byte](uint32(opts.bufferSize)),
		opts:     opts,
		wakeupCh: make(chan struct{}, 1),
		errCh:    make(chan error, 1),
	}
	return b
}

// Start starts the sending goroutine, you need to set writer and done at startup.
func (b *Buffer) Start(writer io.Writer, done chan struct{}) {
	b.w = writer
	b.done = done
	go b.start()
}

// Restart recreates a Buffer when restarting, reusing the buffer queue and configuration of the original Buffer.
func (b *Buffer) Restart(writer io.Writer, done chan struct{}) *Buffer {
	buffer := &Buffer{
		queue:    b.queue,
		opts:     b.opts,
		wakeupCh: make(chan struct{}, 1),
		errCh:    make(chan error, 1),
	}
	buffer.Start(writer, done)
	return buffer
}

// SetQueueStopped sets whether the buffer queue stops receiving packets.
func (b *Buffer) SetQueueStopped(stopped bool) {
	b.isQueueStopped = stopped
	if b.err == nil {
		b.err = ErrStopped
	}
}

// Write writes p to the buffer queue and returns the length of the data written.
// How to write a packet smaller than len(p), err returns the specific reason.
func (b *Buffer) Write(p []byte) (int, error) {
	if b.opts.dropFull {
		return b.writeNoWait(p)
	}
	return b.writeOrWait(p)
}

// Error returns the reason why the sending goroutine exited.
func (b *Buffer) Error() error {
	return b.err
}

// Done return to exit the Channel.
func (b *Buffer) Done() chan struct{} {
	return b.done
}

func (b *Buffer) wakeUp() {
	// Based on performance optimization considerations: due to the poor
	// efficiency of concurrent select write channel, check here first
	// whether wakeupCh already has a wakeup message, reducing the chance
	// of concurrently writing to the channel.
	if len(b.wakeupCh) > 0 {
		return
	}
	// try to send wakeup signal, don't wait.
	select {
	case b.wakeupCh <- struct{}{}:
	default:
	}
}

func (b *Buffer) writeNoWait(p []byte) (int, error) {
	// The buffer queue stops receiving packets and returns directly.
	if b.isQueueStopped {
		return 0, b.err
	}
	// return directly when the queue is full.
	if err := b.queue.Put(p); err != nil {
		return 0, err
	}
	// Write the buffer queue successfully, wake up the sending goroutine.
	b.wakeUp()
	return len(p), nil
}

func (b *Buffer) writeOrWait(p []byte) (int, error) {
	for {
		// The buffer queue stops receiving packets and returns directly.
		if b.isQueueStopped {
			return 0, b.err
		}
		// Write the buffer queue successfully, wake up the sending goroutine.
		if err := b.queue.Put(p); err == nil {
			b.wakeUp()
			return len(p), nil
		}
		// The queue is full, send the package directly.
		if err := b.writeDirectly(); err != nil {
			return 0, err
		}
	}
}

func (b *Buffer) writeDirectly() error {
	if b.queue.IsEmpty() {
		return nil
	}
	vals := make([][]byte, 0, maxWritevBuffers)
	size, _ := b.queue.Gets(&vals)
	if size == 0 {
		return nil
	}
	bufs := make(net.Buffers, 0, maxWritevBuffers)
	for _, v := range vals {
		bufs = append(bufs, v)
	}
	if _, err := bufs.WriteTo(b.w); err != nil {
		// Notify the sending goroutine setting error and exit.
		select {
		case b.errCh <- err:
		default:
		}
		return err
	}
	return nil
}

func (b *Buffer) getOrWait(values *[][]byte) error {
	for {
		// Check whether to be notified to close the outgoing goroutine.
		select {
		case <-b.done:
			return ErrAskQuit
		case err := <-b.errCh:
			return err
		default:
		}
		// Bulk receive packets from the cache queue.
		size, _ := b.queue.Gets(values)
		if size > 0 {
			return nil
		}

		// Fast Path: Due to the poor performance of using select
		// to wake up the goroutine, it is preferred here to use Gosched()
		// to delay checking the queue, improving the hit rate and
		// the efficiency of obtaining packets in batches, thereby reducing
		// the probability of using select to wake up the goroutine.
		runtime.Gosched()
		if !b.queue.IsEmpty() {
			continue
		}
		// Slow Path: There are still no packets after the delayed check queue,
		// indicating that the system is relatively idle. goroutine uses
		// the select mechanism to wait for wakeup. The advantage of hibernation
		// is to reduce CPU idling loss when the system is idle.
		select {
		case <-b.done:
			return ErrAskQuit
		case err := <-b.errCh:
			return err
		case <-b.wakeupCh:
		}
	}
}

func (b *Buffer) start() {
	initBufs := make(net.Buffers, 0, maxWritevBuffers)
	vals := make([][]byte, 0, maxWritevBuffers)
	bufs := initBufs

	defer b.opts.handler(b)
	for {
		if err := b.getOrWait(&vals); err != nil {
			b.err = err
			break
		}

		for _, v := range vals {
			bufs = append(bufs, v)
		}
		vals = vals[:0]

		if _, err := bufs.WriteTo(b.w); err != nil {
			b.err = err
			break
		}
		// Reset bufs to the initial position to prevent `append` from generating new memory allocations.
		bufs = initBufs
	}
}
