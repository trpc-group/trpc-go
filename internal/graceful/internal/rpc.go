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

//go:build !windows

package graceful

import (
	"bufio"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"syscall"
)

// The kernel constant SCM_MAX_FD defines a limit on the number
// of file descriptors in the SCM_RIGHTS. See https://man7.org/linux/man-pages/man7/unix.7.html
// for more details.
// We use a small enough value and call send
// msg multiple times to avoid this restriction.
const maxSCMDataLen = 32

// NewRpcWriter creates a new Writer which can send messages and fds.
func NewRpcWriter(fd int) *Writer {
	w := writer{fd: fd}
	bw := bufio.NewWriter(&w)
	return &Writer{
		writer: &w,
		bw:     bw,
		enc:    gob.NewEncoder(bw),
	}
}

// NewRpcReader creates a new Reader which can receive messages and fds.
func NewRpcReader(fd int) *Reader {
	r := reader{
		fd:        fd,
		ancillary: make([]byte, syscall.CmsgSpace(maxSCMDataLen)),
	}
	return &Reader{
		reader: &r,
		dec:    gob.NewDecoder(&r),
	}
}

// Reader receives messages and fds.
type Reader struct {
	*reader
	dec *gob.Decoder
}

// Decode decodes the msg into struct v.
func (r *Reader) Decode(v interface{}) error {
	return r.dec.Decode(v)
}

// GetFds returns the fds which is carried by ancillary message.
func (r *Reader) GetFds() []int {
	fds := r.fds
	r.fds = nil
	return fds
}

// Writer sends messages and fds.
type Writer struct {
	*writer
	bw  *bufio.Writer
	enc *gob.Encoder
}

// Encode encodes the struct v and stores them waiting for Flush.
func (w *Writer) Encode(v interface{}) error {
	return w.enc.Encode(v)
}

// Flush sends the Encode messages with ancillary fds.
func (w *Writer) Flush(fds []int) error {
	if len(fds) > maxSCMDataLen {
		return fmt.Errorf("exceeded max scm data len %d", maxSCMDataLen)
	}
	if w.fds != nil {
		return errors.New("remain un-sent fds")
	}
	w.fds = fds
	return w.bw.Flush()
}

type reader struct {
	fd        int
	fds       []int
	ancillary []byte
}

// Read reads data into p and ancillary data into fds.
func (r *reader) Read(p []byte) (n int, err error) {
	pn, an, _, _, err := syscall.Recvmsg(r.fd, p, r.ancillary, 0)
	if err != nil {
		return 0, err
	}
	if pn == 0 && an == 0 {
		return 0, io.EOF
	}

	scms, err := syscallParseSocketControlMessage(r.ancillary[:an])
	if err != nil {
		return 0, fmt.Errorf("failed to parse socket control message: %w", err)
	}
	if len(scms) > 1 {
		return 0, errors.New("expect at most one scm at a time")
	}

	if len(scms) == 1 {
		if r.fds != nil {
			return 0, fmt.Errorf("unconsumed fds")
		}
		fds, err := syscall.ParseUnixRights(&scms[0])
		if err != nil {
			return 0, fmt.Errorf("failed to parse unix rights: %w", err)
		}
		r.fds = append(r.fds, fds...)
	}

	return pn, nil
}

type writer struct {
	fd  int
	fds []int
}

// Write writes p into socket with stored ancillary fds.
func (w *writer) Write(p []byte) (n int, err error) {
	var oob []byte
	if w.fds != nil {
		oob = syscall.UnixRights(w.fds...)
		w.fds = nil
	}

	if err := syscall.Sendmsg(w.fd, p, oob, nil, 0); err != nil {
		return 0, err
	}
	return len(p), nil
}

// This is only used to pass meaningless code coverage test.
var syscallParseSocketControlMessage = syscall.ParseSocketControlMessage
