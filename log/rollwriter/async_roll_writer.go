//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 THL A29 Limited, a Tencent company.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

package rollwriter

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/hashicorp/go-multierror"

	"trpc.group/trpc-go/trpc-go/internal/report"
)

const (
	defaultLogQueueSize    = 10000
	defaultWriteLogSize    = 4 * 1024 // 4KB
	defaultLogIntervalInMs = 100
	defaultDropLog         = false
)

// AsyncRollWriter is the asynchronous rolling log writer which implements zapcore.WriteSyncer.
type AsyncRollWriter struct {
	logger io.WriteCloser
	opts   *AsyncOptions

	logQueue chan []byte
	sync     chan struct{}
	syncErr  chan error
	close    chan struct{}
	closeErr chan error
}

// NewAsyncRollWriter creates a new AsyncRollWriter.
func NewAsyncRollWriter(logger io.WriteCloser, opt ...AsyncOption) *AsyncRollWriter {
	opts := &AsyncOptions{
		LogQueueSize:     defaultLogQueueSize,
		WriteLogSize:     defaultWriteLogSize,
		WriteLogInterval: defaultLogIntervalInMs,
		DropLog:          defaultDropLog,
	}

	for _, o := range opt {
		o(opts)
	}

	w := &AsyncRollWriter{
		logger:   logger,
		opts:     opts,
		logQueue: make(chan []byte, opts.LogQueueSize),
		sync:     make(chan struct{}),
		syncErr:  make(chan error),
		close:    make(chan struct{}),
		closeErr: make(chan error),
	}

	// Start a new goroutine to write batch logs.
	go w.batchWriteLog()
	return w
}

// Write writes logs. It implements io.Writer.
func (w *AsyncRollWriter) Write(data []byte) (int, error) {
	log := make([]byte, len(data))
	copy(log, data)
	if w.opts.DropLog {
		select {
		case w.logQueue <- log:
		default:
			report.LogQueueDropNum.Incr()
			return 0, errors.New("async roll writer: log queue is full")
		}
		return len(data), nil
	}
	w.logQueue <- log
	return len(data), nil
}

// Sync syncs logs. It implements zapcore.WriteSyncer.
func (w *AsyncRollWriter) Sync() error {
	w.sync <- struct{}{}
	return <-w.syncErr
}

// Close closes current log file. It implements io.Closer.
func (w *AsyncRollWriter) Close() error {
	err := w.Sync()
	close(w.close)
	return multierror.Append(err, <-w.closeErr).ErrorOrNil()
}

// batchWriteLog asynchronously writes logs in batches.
func (w *AsyncRollWriter) batchWriteLog() {
	buffer := bytes.NewBuffer(make([]byte, 0, w.opts.WriteLogSize*2))
	ticker := time.NewTicker(time.Millisecond * time.Duration(w.opts.WriteLogInterval))
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if buffer.Len() > 0 {
				_, err := w.logger.Write(buffer.Bytes())
				handleErr(err, "w.logger.Write on tick")
				buffer.Reset()
			}
		case data := <-w.logQueue:
			buffer.Write(data)
			if buffer.Len() >= w.opts.WriteLogSize {
				_, err := w.logger.Write(buffer.Bytes())
				handleErr(err, "w.logger.Write on log queue")
				buffer.Reset()
			}
		case <-w.sync:
			var err error
			if buffer.Len() > 0 {
				_, e := w.logger.Write(buffer.Bytes())
				err = multierror.Append(err, e).ErrorOrNil()
				buffer.Reset()
			}
			size := len(w.logQueue)
			for i := 0; i < size; i++ {
				v := <-w.logQueue
				_, e := w.logger.Write(v)
				err = multierror.Append(err, e).ErrorOrNil()
			}
			w.syncErr <- err
		case <-w.close:
			w.closeErr <- w.logger.Close()
			return
		}
	}
}

func handleErr(err error, msg string) {
	if err == nil {
		return
	}
	// Log writer has errors, so output to stdout directly.
	fmt.Printf("async roll writer err: %+v, msg: %s", err, msg)
}
