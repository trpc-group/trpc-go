//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 Tencent.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

package transport

import (
	"context"
	"fmt"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/internal/addrutil"
)

// serverStreamTransport implements ServerStreamTransport and keeps backward compatibility with the
// original serverTransport.
type serverStreamTransport struct {
	// Keep backward compatibility with original serverTransport.
	serverTransport
}

// NewServerStreamTransport creates a new ServerTransport, which is wrapped in serverStreamTransport
// as the return ServerStreamTransport interface.
func NewServerStreamTransport(opt ...ServerTransportOption) ServerStreamTransport {
	s := newServerTransport(opt...)
	return &serverStreamTransport{s}
}

// DefaultServerStreamTransport is the default ServerStreamTransport.
var DefaultServerStreamTransport = NewServerStreamTransport()

// ListenAndServe implements ServerTransport.
// To be compatible with common RPC and stream RPC, we use serverTransport.ListenAndServe function.
func (st *serverStreamTransport) ListenAndServe(ctx context.Context, opts ...ListenServeOption) error {
	return st.serverTransport.ListenAndServe(ctx, opts...)
}

// Send is the method to send stream messages.
func (st *serverStreamTransport) Send(ctx context.Context, req []byte) error {
	msg := codec.Message(ctx)
	raddr := msg.RemoteAddr()
	laddr := msg.LocalAddr()
	if raddr == nil || laddr == nil {
		return errs.NewFrameError(errs.RetServerSystemErr,
			fmt.Sprintf("Address is invalid, local: %s, remote: %s", laddr, raddr))
	}
	key := addrutil.AddrToKey(laddr, raddr)
	st.serverTransport.m.RLock()
	tc, ok := st.serverTransport.addrToConn[key]
	st.serverTransport.m.RUnlock()
	if ok && tc != nil {
		if _, err := tc.rwc.Write(req); err != nil {
			tc.close()
			st.Close(ctx)
			return err
		}
		return nil
	}
	return errs.NewFrameError(errs.RetServerSystemErr, "Can't find conn by addr")
}

// Close closes ServerStreamTransport, it also cleans up cached connections.
func (st *serverStreamTransport) Close(ctx context.Context) {
	msg := codec.Message(ctx)
	key := addrutil.AddrToKey(msg.LocalAddr(), msg.RemoteAddr())
	st.m.Lock()
	delete(st.serverTransport.addrToConn, key)
	st.m.Unlock()
}
