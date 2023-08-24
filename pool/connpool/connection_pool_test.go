// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package connpool

import (
	"context"
	"errors"
	"io"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"trpc.group/trpc-go/trpc-go/codec"
)

var (
	ErrFrameSet       = errors.New("framer not set")
	ErrReamFrame      = errors.New("ReadFrame failed")
	ErrRead           = errors.New("Read failed")
	ErrWrite          = errors.New("Write failed")
	ErrSyscallConn    = errors.New("SyscallConn Failed")
	ErrUnexpectedRead = errors.New("unexpected read from socket")

	mockChecker = func(*PoolConn, bool) bool { return true }
)

func TestInitialMinIdle(t *testing.T) {
	var established int32
	p := NewConnectionPool(
		WithMinIdle(10),
		WithDialFunc(func(*DialOptions) (net.Conn, error) {
			atomic.AddInt32(&established, 1)
			return &noopConn{closeFunc: func() {}}, nil
		}),
		WithHealthChecker(mockChecker))
	defer closePool(t, p)

	pc, err := p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
	require.Nil(t, err)
	require.Nil(t, pc.Close())

	start := time.Now()
	for time.Since(start) < time.Second {
		if established := atomic.LoadInt32(&established); established == 10 || established == 11 {
			return
		}
		runtime.Gosched()
	}
	require.FailNow(t, "expected 10/11 established connections for fresh pool")
}

func TestKeepMinIdle(t *testing.T) {
	var established int32
	minIdle := 10
	p := NewConnectionPool(
		WithMinIdle(minIdle),
		WithDialFunc(func(*DialOptions) (net.Conn, error) {
			atomic.AddInt32(&established, 1)

			return &noopConn{closeFunc: func() {}}, nil
		}),
		WithHealthChecker(mockChecker))
	defer closePool(t, p)

	// clear idle conns
	pc, err := p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
	require.Nil(t, err)
	require.Nil(t, pc.Close())
	start := time.Now()
	for time.Since(start) < time.Second {
		if established := atomic.LoadInt32(&established); established == 10 || established == 11 {
			break
		}
		runtime.Gosched()
	}
	cnt := (int)(atomic.LoadInt32(&established))
	for i := 0; i < cnt; i++ {
		pc, err := p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
		assert.Nil(t, err)
		defer pc.Close()
	}

	// wait for create idle conns background
	start = time.Now()
	target := (int32)(cnt + minIdle)
	for time.Since(start) < defaultCheckInterval*2 {
		if established := atomic.LoadInt32(&established); established == target {
			return
		}
		runtime.Gosched()
	}
	require.FailNow(t, "expected 20 established connections for fresh pool")
}

func TestGetTokenWithoutMaxActive(t *testing.T) {
	p := NewConnectionPool(
		WithDialFunc(func(*DialOptions) (net.Conn, error) {
			return &noopConn{closeFunc: func() {}}, nil
		}),
		WithHealthChecker(mockChecker))
	defer closePool(t, p)

	pc, err := p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
	require.Nil(t, err)
	require.Nil(t, pc.Close())
}

func TestGetTokenWait(t *testing.T) {
	maxActive := 10
	p := NewConnectionPool(
		WithMaxActive(maxActive),
		WithWait(true),
		WithDialFunc(func(*DialOptions) (net.Conn, error) {
			return &noopConn{closeFunc: func() {}}, nil
		}),
		WithHealthChecker(mockChecker))
	defer closePool(t, p)

	pcs := make([]net.Conn, 0, maxActive)
	for i := 0; i < maxActive; i++ {
		pc, err := p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
		assert.Nil(t, err)
		pcs = append(pcs, pc)
	}

	_, err := p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
	require.Equal(t, err, context.DeadlineExceeded)

	for _, pc := range pcs {
		require.Nil(t, pc.Close())
	}
}

func TestGetTokenNoWait(t *testing.T) {
	maxActive := 10
	p := NewConnectionPool(
		WithMaxActive(maxActive),
		WithWait(false),
		WithDialFunc(func(*DialOptions) (net.Conn, error) {
			return &noopConn{closeFunc: func() {}}, nil
		}),
		WithHealthChecker(mockChecker))
	defer closePool(t, p)

	pcs := make([]net.Conn, 0, maxActive)
	for i := 0; i < maxActive; i++ {
		pc, err := p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
		assert.Nil(t, err)
		pcs = append(pcs, pc)
	}

	_, err := p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
	require.Equal(t, err, ErrPoolLimit)

	for _, pc := range pcs {
		require.Nil(t, pc.Close())
	}

	pc, err2 := p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
	require.Nil(t, err2)
	require.Nil(t, pc.Close())
}

func TestIdleTimeout(t *testing.T) {
	var established int32
	idleTimeout := time.Millisecond * 100
	p := NewConnectionPool(
		WithIdleTimeout(idleTimeout),
		WithDialFunc(func(*DialOptions) (net.Conn, error) {
			atomic.AddInt32(&established, 1)
			return &noopConn{closeFunc: func() {
				atomic.AddInt32(&established, -1)
			}}, nil
		}))
	defer closePool(t, p)

	cnt := 3
	pcs := make([]net.Conn, 0, cnt)
	for i := 0; i < cnt; i++ {
		pc, err := p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
		require.Nil(t, err)
		pcs = append(pcs, pc)
	}
	require.Equal(t, atomic.LoadInt32(&established), int32(cnt))
	for _, pc := range pcs {
		require.Nil(t, pc.Close())
	}

	start := time.Now()
	for time.Since(start) < defaultCheckInterval*2 {
		if established := atomic.LoadInt32(&established); established == 0 {
			break
		}
		runtime.Gosched()
	}
	pc, err := p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
	require.Nil(t, err)
	require.Equal(t, atomic.LoadInt32(&established), int32(1))
	require.Nil(t, pc.Close())
}

func TestMaxConnLifetime(t *testing.T) {
	var established int32
	maxConnLifetime := time.Millisecond * 100
	p := NewConnectionPool(
		WithMaxConnLifetime(maxConnLifetime),
		WithDialFunc(func(*DialOptions) (net.Conn, error) {
			atomic.AddInt32(&established, 1)
			return &noopConn{closeFunc: func() {
				atomic.AddInt32(&established, -1)
			}}, nil
		}))
	defer closePool(t, p)

	cnt := 3
	pcs := make([]net.Conn, 0, cnt)
	for i := 0; i < cnt; i++ {
		pc, err := p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
		require.Nil(t, err)
		pcs = append(pcs, pc)
	}
	require.Equal(t, atomic.LoadInt32(&established), int32(cnt))
	for _, pc := range pcs {
		require.Nil(t, pc.Close())
	}

	start := time.Now()
	for time.Since(start) < defaultCheckInterval*2 {
		if established := atomic.LoadInt32(&established); established == 0 {
			break
		}
		runtime.Gosched()
	}
	pc, err := p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
	require.Nil(t, err)
	require.Equal(t, atomic.LoadInt32(&established), int32(1))
	require.Nil(t, pc.Close())
}

func TestConcurrencyGet(t *testing.T) {
	p := NewConnectionPool(
		WithDialFunc(func(*DialOptions) (net.Conn, error) {
			return &noopConn{closeFunc: func() {}}, nil
		}),
		WithHealthChecker(mockChecker))
	defer closePool(t, p)

	cnt := 5
	pcs := make([]net.Conn, cnt)
	var wg sync.WaitGroup
	for i := 0; i < cnt; i++ {
		wg.Add(1)
		idx := i
		go func() {
			pc, err := p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
			assert.Nil(t, err)
			pcs[idx] = pc
			wg.Done()
		}()
	}
	wg.Wait()

	for _, pc := range pcs {
		require.Nil(t, pc.Close())
	}
}

func TestPutForceClose(t *testing.T) {
	var established int32
	p := NewConnectionPool(
		WithForceClose(true),
		WithDialFunc(func(*DialOptions) (net.Conn, error) {
			atomic.AddInt32(&established, 1)
			return &noopConn{closeFunc: func() {
				atomic.AddInt32(&established, -1)
			}}, nil
		}),
		WithHealthChecker(mockChecker))
	defer closePool(t, p)

	cnt := 5
	pcs := make([]net.Conn, 0, cnt)
	for i := 0; i < cnt; i++ {
		pc, err := p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
		require.Nil(t, err)
		pcs = append(pcs, pc)
	}
	for _, pc := range pcs {
		require.Nil(t, pc.Close())
	}
	require.Equal(t, atomic.LoadInt32(&established), int32(0))
}

func TestIdleFifo(t *testing.T) {
	p := NewConnectionPool(
		WithPushIdleConnToTail(true),
		WithDialFunc(func(*DialOptions) (net.Conn, error) {
			return &noopConn{closeFunc: func() {}}, nil
		}),
		WithHealthChecker(mockChecker))
	defer closePool(t, p)

	cnt := 5
	pcs := make([]net.Conn, 0, cnt)
	for i := 0; i < cnt; i++ {
		pc, err := p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
		require.Nil(t, err)
		pcs = append(pcs, pc)
	}
	for _, pc := range pcs {
		time.Sleep(10 * time.Millisecond)
		require.Nil(t, pc.Close())
	}

	pcs = make([]net.Conn, 0, cnt)
	pc, err := p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
	require.Nil(t, err)
	pcs = append(pcs, pc)
	created := pc.(*PoolConn).t
	for i := 1; i < cnt; i++ {
		pc, err := p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
		require.Nil(t, err)
		pcs = append(pcs, pc)
		require.True(t, created.Before(pc.(*PoolConn).t))
		created = pc.(*PoolConn).t
	}

	for _, pc := range pcs {
		require.Nil(t, pc.Close())
	}
}

func TestIdleLifo(t *testing.T) {
	p := NewConnectionPool(
		WithPushIdleConnToTail(false),
		WithDialFunc(func(*DialOptions) (net.Conn, error) {
			return &noopConn{closeFunc: func() {}}, nil
		}),
		WithHealthChecker(mockChecker))
	defer closePool(t, p)

	cnt := 5
	pcs := make([]net.Conn, 0, cnt)
	for i := 0; i < cnt; i++ {
		pc, err := p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
		require.Nil(t, err)
		pcs = append(pcs, pc)
	}
	for _, pc := range pcs {
		time.Sleep(10 * time.Millisecond)
		require.Nil(t, pc.Close())
	}

	pcs = make([]net.Conn, 0, cnt)
	pc, err := p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
	require.Nil(t, err)
	pcs = append(pcs, pc)
	created := pc.(*PoolConn).t
	for i := 1; i < cnt; i++ {
		pc, err := p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
		require.Nil(t, err)
		pcs = append(pcs, pc)
		require.True(t, created.After(pc.(*PoolConn).t))
		created = pc.(*PoolConn).t
	}

	for _, pc := range pcs {
		require.Nil(t, pc.Close())
	}
}

func TestOverMaxIdle(t *testing.T) {
	var established int32
	maxIdle := 5
	p := NewConnectionPool(
		WithMaxIdle(maxIdle),
		WithDialFunc(func(*DialOptions) (net.Conn, error) {
			atomic.AddInt32(&established, 1)
			return &noopConn{closeFunc: func() {
				atomic.AddInt32(&established, -1)
			}}, nil
		}),
		WithHealthChecker(mockChecker))
	defer closePool(t, p)

	cnt := 10
	pcs := make([]net.Conn, 0, cnt)
	for i := 0; i < cnt; i++ {
		pc, err := p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
		assert.Nil(t, err)
		pcs = append(pcs, pc)
	}
	for _, pc := range pcs {
		require.Nil(t, pc.Close())
	}

	require.Equal(t, atomic.LoadInt32(&established), int32(maxIdle))
}

func TestPoolClose(t *testing.T) {
	var established int32
	p := NewConnectionPool(
		WithDialFunc(func(*DialOptions) (net.Conn, error) {
			atomic.AddInt32(&established, 1)
			return &noopConn{closeFunc: func() {
				atomic.AddInt32(&established, -1)

			}}, nil
		}),
		WithHealthChecker(mockChecker))

	cnt := 10
	pcs := make([]net.Conn, 0, cnt)
	for i := 0; i < cnt; i++ {
		pc, err := p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
		assert.Nil(t, err)
		pcs = append(pcs, pc)
	}
	for _, pc := range pcs {
		require.Nil(t, pc.Close())
	}

	closePool(t, p)
	require.Equal(t, atomic.LoadInt32(&established), int32(0))
}

func TestGetAfterPoolClose(t *testing.T) {
	p := NewConnectionPool(
		WithDialFunc(func(*DialOptions) (net.Conn, error) {
			return &noopConn{closeFunc: func() {}}, nil
		}),
		WithHealthChecker(mockChecker))

	pc, err := p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
	require.Nil(t, err)
	require.Nil(t, pc.Close())

	closePool(t, p)

	_, err2 := p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
	require.Equal(t, err2, ErrPoolClosed)
}

func TestCloseConnAfterPoolClose(t *testing.T) {
	var established int32
	p := NewConnectionPool(
		WithDialFunc(func(*DialOptions) (net.Conn, error) {
			atomic.AddInt32(&established, 1)
			return &noopConn{closeFunc: func() {
				atomic.AddInt32(&established, -1)
			}}, nil
		}),
		WithHealthChecker(mockChecker))
	defer closePool(t, p)

	pc, err := p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
	require.Nil(t, err)

	closePool(t, p)
	require.Nil(t, pc.Close())
	require.Equal(t, atomic.LoadInt32(&established), int32(0))
}

func TestCloseConnAfterConnCloseWithForceClose(t *testing.T) {
	var established int32
	p := NewConnectionPool(
		WithDialFunc(func(*DialOptions) (net.Conn, error) {
			atomic.AddInt32(&established, 1)
			return &noopConn{closeFunc: func() {
				atomic.AddInt32(&established, -1)
			}}, nil
		}),
		WithHealthChecker(mockChecker),
		WithForceClose(true))
	defer closePool(t, p)

	pc, err := p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
	require.Nil(t, err)
	require.Nil(t, pc.Close())
	require.Equal(t, atomic.LoadInt32(&established), int32(0))
	require.Equal(t, pc.Close(), ErrConnClosed)
}

func TestCloseConnAfterConnCloseWithoutForceClose(t *testing.T) {
	var established int32
	p := NewConnectionPool(
		WithDialFunc(func(*DialOptions) (net.Conn, error) {
			atomic.AddInt32(&established, 1)
			return &noopConn{closeFunc: func() {
				atomic.AddInt32(&established, -1)
			}}, nil
		}),
		WithHealthChecker(mockChecker))
	defer closePool(t, p)

	pc, err := p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
	require.Nil(t, err)
	require.Nil(t, pc.Close())
	require.Equal(t, pc.Close(), ErrConnInPool)
}

func TestReadFrameAfterClosed(t *testing.T) {
	p := NewConnectionPool(
		WithDialFunc(func(*DialOptions) (net.Conn, error) {
			return &noopConn{closeFunc: func() {}}, nil
		}),
		WithHealthChecker(mockChecker),
		WithForceClose(true))
	defer closePool(t, p)

	pc, err := p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
	require.Nil(t, err)
	require.Nil(t, pc.Close())

	_, err2 := pc.(codec.Framer).ReadFrame()
	require.Equal(t, err2, ErrConnClosed)
}

func TestReadFrameWithoutFramer(t *testing.T) {
	p := NewConnectionPool(
		WithDialFunc(func(*DialOptions) (net.Conn, error) {
			return &noopConn{closeFunc: func() {}}, nil
		}),
		WithHealthChecker(mockChecker),
		WithForceClose(true))
	defer closePool(t, p)

	pc, err := p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
	require.Nil(t, err)
	_, err2 := pc.(codec.Framer).ReadFrame()
	require.Equal(t, err2, ErrFrameSet)
	require.Equal(t, pc.Close(), ErrConnClosed)
}

func TestReadFrameFailed(t *testing.T) {
	p := NewConnectionPool(
		WithDialFunc(func(*DialOptions) (net.Conn, error) {
			return &noopConn{closeFunc: func() {}}, nil
		}),
		WithHealthChecker(mockChecker),
		WithForceClose(true))
	defer closePool(t, p)

	pc, err := p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader,
		DialTimeout:   time.Second,
		FramerBuilder: &noopFramerBuilder{false},
	})
	require.Nil(t, err)
	_, err2 := pc.(codec.Framer).ReadFrame()
	require.Equal(t, err2, ErrReamFrame)
	require.Equal(t, pc.Close(), ErrConnClosed)
}

func TestReadFrameWithCopyFrame(t *testing.T) {
	p := NewConnectionPool(
		WithDialFunc(func(*DialOptions) (net.Conn, error) {
			return &noopConn{closeFunc: func() {}}, nil
		}),
		WithHealthChecker(mockChecker),
		WithForceClose(true))

	pc, err := p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader,
		DialTimeout:   time.Second,
		FramerBuilder: &noopFramerBuilder{true},
	})
	require.Nil(t, err)
	_, err2 := pc.(codec.Framer).ReadFrame()
	require.Nil(t, err2)
	require.Nil(t, pc.Close())
}

func TestWriteAfterClosed(t *testing.T) {
	p := NewConnectionPool(
		WithDialFunc(func(*DialOptions) (net.Conn, error) {
			return &noopConn{closeFunc: func() {}}, nil
		}),
		WithHealthChecker(mockChecker),
		WithForceClose(true))
	defer closePool(t, p)

	pc, err := p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
	require.Nil(t, err)
	require.Nil(t, pc.Close())

	buf := make([]byte, 1)
	_, err2 := pc.Write(buf)
	require.Equal(t, err2, ErrConnClosed)
	require.Equal(t, pc.Close(), ErrConnClosed)
}

func TestWriteFailed(t *testing.T) {
	p := NewConnectionPool(
		WithDialFunc(func(*DialOptions) (net.Conn, error) {
			return &noopConn{suc: false, closeFunc: func() {}}, nil
		}),
		WithHealthChecker(mockChecker),
		WithForceClose(true))
	defer closePool(t, p)

	pc, err := p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
	require.Nil(t, err)

	buf := make([]byte, 1)
	_, err2 := pc.Write(buf)
	require.Equal(t, err2, ErrWrite)
	require.Equal(t, pc.Close(), ErrConnClosed)
}

func TestReadAfterClosed(t *testing.T) {
	p := NewConnectionPool(
		WithDialFunc(func(*DialOptions) (net.Conn, error) {
			return &noopConn{closeFunc: func() {}}, nil
		}),
		WithHealthChecker(mockChecker),
		WithForceClose(true))
	defer closePool(t, p)

	pc, err := p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
	require.Nil(t, err)
	require.Nil(t, pc.Close())

	buf := make([]byte, 1)
	_, err2 := pc.Read(buf)
	require.Equal(t, err2, ErrConnClosed)
	require.Equal(t, pc.Close(), ErrConnClosed)

}

func TestReadFailed(t *testing.T) {
	p := NewConnectionPool(
		WithDialFunc(func(*DialOptions) (net.Conn, error) {
			return &noopConn{suc: false, closeFunc: func() {}}, nil
		}),
		WithHealthChecker(mockChecker),
		WithForceClose(true))
	defer closePool(t, p)

	pc, err := p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
	require.Nil(t, err)

	buf := make([]byte, 1)
	_, err2 := pc.Read(buf)
	require.Equal(t, err2, ErrRead)
	require.Equal(t, pc.Close(), ErrConnClosed)
}

func TestReadFailedFreeToken(t *testing.T) {
	p := NewConnectionPool(
		WithMaxActive(5),
		WithDialFunc(func(*DialOptions) (net.Conn, error) {
			return &noopConn{suc: false, closeFunc: func() {}}, nil
		}),
		WithHealthChecker(mockChecker),
		WithForceClose(true))
	defer closePool(t, p)

	pc, err := p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
	require.Nil(t, err)
	require.Equal(t, 1, len(pc.(*PoolConn).pool.token))

	buf := make([]byte, 1)
	_, err2 := pc.Read(buf)
	require.Equal(t, err2, ErrRead)
	require.Equal(t, pc.Close(), ErrConnClosed)
	require.Equal(t, 0, len(pc.(*PoolConn).pool.token))
}

func TestConnPoolIdleTimeout(t *testing.T) {
	idleTimeout := time.Millisecond * 100
	poolIdleTimeout := time.Millisecond * 100
	getSize := func(p Pool) int {
		pool, ok := p.(*pool)
		assert.Equal(t, true, ok)
		var count int
		pool.connectionPools.Range(func(key, value interface{}) bool {
			count++
			return true
		})
		return count
	}
	p := NewConnectionPool(
		WithDialFunc(func(*DialOptions) (net.Conn, error) {
			return &noopConn{suc: false, closeFunc: func() {}}, nil
		}),
		WithIdleTimeout(idleTimeout),
		WithMinIdle(5),
		WithPoolIdleTimeout(poolIdleTimeout))

	assert.Equal(t, 0, getSize(p))

	c, err := p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
	assert.Nil(t, err)
	assert.NotNil(t, c)
	assert.Nil(t, c.Close())

	assert.Equal(t, 1, getSize(p))
	time.Sleep(poolIdleTimeout + defaultCheckInterval)
	assert.Equal(t, 0, getSize(p))

	//get again
	c, err = p.Get(t.Name(), t.Name(), GetOptions{CustomReader: codec.NewReader, DialTimeout: time.Second})
	assert.Nil(t, err)
	assert.NotNil(t, c)
	assert.Nil(t, c.Close())
}

func TestConnPoolTokenFreeOnReadFrameError(t *testing.T) {
	const maxActive = 1
	p := NewConnectionPool(
		WithDialFunc(func(*DialOptions) (net.Conn, error) {
			return &noopConn{suc: false, closeFunc: func() {}}, nil
		}),
		WithMaxActive(maxActive),
	)
	c, err := p.Get(t.Name(), t.Name(), GetOptions{DialTimeout: time.Second})
	require.Nil(t, err)
	pc, ok := c.(*PoolConn)
	require.True(t, ok)
	require.Equal(t, 1, len(pc.pool.token))
	_, err = pc.ReadFrame() // Error of ReadFrame will put back (*PoolConn) to pool with forceClose=true.
	require.NotNil(t, err)
	require.Equal(t, 0, len(pc.pool.token))
	ec := make(chan error)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	go func() {
		ec <- pc.Close() // Close will try to put back (*PoolConn) to pool again.
	}()
	errTimeout := errors.New("pc.Close reaches its timeout, probably somewhere hangs")
	select {
	case err = <-ec:
	case <-ctx.Done():
		err = errTimeout
	}
	require.False(t, errors.Is(err, errTimeout))
	go func() {
		ec <- pc.pool.put(pc, true) // Put a closed pc to pool should not hang neither.
	}()
	select {
	case err = <-ec:
	case <-ctx.Done():
		err = errTimeout
	}
	require.False(t, errors.Is(err, errTimeout))
}

func closePool(t *testing.T, p Pool) {
	v, ok := p.(*pool)
	if !ok {
		return
	}
	key := getNodeKey(t.Name(), t.Name(), "")
	if pool, ok := v.connectionPools.Load(key); ok {
		pool.(*ConnectionPool).Close()
	}
}

type noopConn struct {
	closeFunc func()
	suc       bool
}

func (c *noopConn) Read(bs []byte) (int, error) {
	if !c.suc {
		return len(bs), ErrRead
	}
	return len(bs), nil
}
func (c *noopConn) Write(bs []byte) (int, error) {
	if !c.suc {
		return len(bs), ErrWrite
	}
	return len(bs), nil
}

func (c *noopConn) Close() error {
	c.closeFunc()
	return nil
}

func (c *noopConn) LocalAddr() net.Addr              { return nil }
func (c *noopConn) RemoteAddr() net.Addr             { return nil }
func (c *noopConn) SetDeadline(time.Time) error      { return nil }
func (c *noopConn) SetReadDeadline(time.Time) error  { return nil }
func (c *noopConn) SetWriteDeadline(time.Time) error { return nil }

type noopFramerBuilder struct {
	suc bool
}

func (fb *noopFramerBuilder) New(io.Reader) codec.Framer {
	return &noopFramer{fb.suc}
}

type noopFramer struct {
	suc bool
}

func (fr *noopFramer) ReadFrame() ([]byte, error) {
	if !fr.suc {
		return make([]byte, 1), ErrReamFrame
	}
	return make([]byte, 1), nil
}

func (fr *noopFramer) IsSafe() bool {
	return false
}
