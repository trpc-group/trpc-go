package multiplexed

import (
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/hashicorp/go-multierror"
	"go.uber.org/atomic"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/internal/packetbuffer"
	"trpc.group/trpc-go/trpc-go/internal/report"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/pool/connpool"
)

// NewDialFunc returns a DialFunc which can dial a connections used in multiplexed pool.
func NewDialFunc(opts ...ConnOption) DialFunc {
	option := &connOptions{
		sendingQueueSize: defaultSendingQueueSize,
	}
	for _, opt := range opts {
		opt(option)
	}
	return func(fp FrameParser, opts *connpool.DialOptions) (Conn, error) {
		switch opts.Network {
		case "tcp", "tcp4", "tcp6", "unix":
			conn, err := connpool.Dial(opts)
			if err != nil {
				return nil, err
			}
			if c, ok := conn.(*net.TCPConn); ok {
				c.SetKeepAlive(true)
			}
			return &tcpConn{
				frameParser: fp,
				writeBuffer: make(chan []byte, option.sendingQueueSize),
				dropFull:    option.dropFull,
				conn:        conn,
				localAddr:   conn.LocalAddr(),
				remoteAddr:  conn.RemoteAddr(),
				reader:      codec.NewReaderSize(conn, defaultBufferSize),
				closeCh:     make(chan struct{}),
			}, nil
		case "udp", "udp4", "udp6":
			addr, err := net.ResolveUDPAddr(opts.Network, opts.Address)
			if err != nil {
				return nil, err
			}
			const defaultLocalAddr = ":"
			localAddr := defaultLocalAddr
			if opts.LocalAddr != "" {
				localAddr = opts.LocalAddr
			}
			conn, err := net.ListenPacket(opts.Network, localAddr)
			if err != nil {
				return nil, err
			}
			return &udpConn{
				frameParser: fp,
				packetConn:  conn,
				remoteAddr:  addr,
				localAddr:   conn.LocalAddr(),
				packetBuf:   packetbuffer.New(conn, defaultMaxBufferSize),
			}, nil
		default:
			return nil, fmt.Errorf("unsupported network: %s", opts.Network)
		}
	}
}

// ConnOption is the function to set connection option.
type ConnOption func(*connOptions)

// WithSendingQueueSize returns a ConnOption which sets the
// size of each Connection sending queue in the multiplexed pool.
func WithSendingQueueSize(size int) ConnOption {
	return func(o *connOptions) {
		o.sendingQueueSize = size
	}
}

// WithDropFull returns a ConnOption which enable to drop the request when the buffer chan is full.
func WithDropFull() ConnOption {
	return func(opts *connOptions) {
		opts.dropFull = true
	}
}

const (
	defaultBufferSize       = 128 * 1024
	defaultMaxBufferSize    = 65535
	defaultSendingQueueSize = 1023
)

// errSendingQueueFull indicates sending queue is full.
var errSendingQueueFull = errors.New("connection's send queue is full")

type connOptions struct {
	sendingQueueSize int
	dropFull         bool
}

type tcpConn struct {
	writeBuffer chan []byte
	dropFull    bool
	conn        net.Conn
	localAddr   net.Addr
	remoteAddr  net.Addr
	reader      io.Reader
	frameParser FrameParser
	isClosed    atomic.Bool
	closeCh     chan struct{}
}

// Start starts background read and write processes.
func (tc *tcpConn) Start(n Notifier) error {
	go tc.reading(n)
	go tc.writing(n)
	return nil
}

// Write writes data to the connection and returns the number of bytes written.
func (tc *tcpConn) Write(buf []byte) (int, error) {
	if tc.dropFull {
		select {
		case tc.writeBuffer <- buf:
			return len(buf), nil
		default:
			return 0, errSendingQueueFull
		}
	}
	select {
	case tc.writeBuffer <- buf:
		return len(buf), nil
	case <-tc.closeCh:
		return 0, ErrConnClosed
	}
}

// Close closes the connection.
func (tc *tcpConn) Close() error {
	if !tc.isClosed.CAS(false, true) {
		return nil
	}
	close(tc.closeCh)
	return tc.conn.Close()
}

// LocalAddr returns the local network address of the connection.
func (tc *tcpConn) LocalAddr() net.Addr {
	return tc.conn.LocalAddr()
}

// RemoteAddr returns the remote network address of the connection.
func (tc *tcpConn) RemoteAddr() net.Addr {
	return tc.conn.RemoteAddr()
}

// IsActive returns whether the connection is active.
func (tc *tcpConn) IsActive() bool {
	return !tc.isClosed.Load()
}

func (tc *tcpConn) reading(n Notifier) {
	var errToClose error
	for {
		vid, buf, err := tc.frameParser.Parse(tc.reader)
		if err != nil {
			// If there is an error in tcp unpacking, it may cause problems with
			// all subsequent parsing, so it is necessary to close the connection.
			errToClose = err
			report.MultiplexedTCPReadingErr.Incr()
			log.Trace("close connection when read err: ", err)
			break
		}
		n.Dispatch(vid, buf)
	}
	n.Close(multierror.Append(ErrConnClosed, errToClose).ErrorOrNil())
	tc.Close()
}

func (tc *tcpConn) writing(n Notifier) {
	loop := func() error {
		for {
			select {
			case <-tc.closeCh:
				return ErrConnClosed
			case buf := <-tc.writeBuffer:
				if err := tc.writeAll(buf); err != nil {
					return err
				}
			}
		}
	}
	if err := loop(); err != nil {
		report.MultiplexedTCPWritingErr.Incr()
		log.Trace("multiplexed close writing loop because err: ", err)
		n.Close(multierror.Append(ErrConnClosed, err).ErrorOrNil())
		tc.Close()
	}
}

func (tc *tcpConn) writeAll(buf []byte) error {
	var sentNum, num int
	var err error
	for sentNum < len(buf) {
		num, err = tc.conn.Write(buf[sentNum:])
		if err != nil {
			return err
		}
		sentNum += num
	}
	return nil
}

type udpConn struct {
	frameParser FrameParser
	packetBuf   *packetbuffer.PacketBuffer
	packetConn  net.PacketConn
	remoteAddr  net.Addr
	localAddr   net.Addr
	isClose     atomic.Bool
}

// Start starts background read process.
func (uc *udpConn) Start(n Notifier) error {
	go uc.reading(n)
	return nil
}

// Write writes data to the connection and returns the number of bytes written.
func (uc *udpConn) Write(buf []byte) (int, error) {
	n, err := uc.packetConn.WriteTo(buf, uc.remoteAddr)
	if err != nil {
		return 0, err
	}
	if n != len(buf) {
		return n, ErrWriteNotFinished
	}
	return n, nil
}

// Close closes the connection.
func (uc *udpConn) Close() error {
	if !uc.isClose.CAS(false, true) {
		return nil
	}
	return uc.packetConn.Close()
}

// LocalAddr returns the local network address of the connection.
func (uc *udpConn) LocalAddr() net.Addr {
	return uc.localAddr
}

// RemoteAddr returns the remote network address of the connection.
func (uc *udpConn) RemoteAddr() net.Addr {
	return uc.remoteAddr
}

// IsActive returns whether the connection is active.
func (uc *udpConn) IsActive() bool {
	return !uc.isClose.Load()
}

func (uc *udpConn) reading(n Notifier) {
	var errToClose error
	for {
		vid, buf, err := uc.parse()
		if err != nil {
			if errors.Is(err, packetbuffer.ErrReadFrom) {
				errToClose = err
				log.Tracef("parse udp packet fatal err: %+v", err)
				break
			}
			// udp is processed according to a single packet, receiving an illegal
			// packet does not affect the subsequent packet processing logic, and
			// can continue to receive packets.
			log.Tracef("parse udp packet err: %+v", err)
			continue
		}
		n.Dispatch(vid, buf)
	}
	uc.packetConn.Close()
	n.Close(multierror.Append(ErrConnClosed, errToClose).ErrorOrNil())
}

func (uc *udpConn) parse() (uint32, []byte, error) {
	vid, buf, err := uc.frameParser.Parse(uc.packetBuf)
	closeErr := uc.packetBuf.Next()
	return vid, buf, multierror.Append(err, closeErr).ErrorOrNil()
}
