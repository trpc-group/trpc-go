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

package trpc

import (
	"context"
	"net"
	"runtime"
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"
	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/internal/report"
	"trpc.group/trpc-go/trpc-go/log"
)

// PanicBufLen is len of buffer used for stack trace logging
// when the goroutine panics, 1024 by default.
var PanicBufLen = 1024

// ----------------------- trpc util functions ------------------------------------ //

// Message returns msg from ctx.
func Message(ctx context.Context) codec.Msg {
	return codec.Message(ctx)
}

// BackgroundContext puts an initialized msg into background context and returns it.
func BackgroundContext() context.Context {
	cfg := GlobalConfig()
	ctx, msg := codec.WithNewMessage(context.Background())
	msg.WithCalleeContainerName(cfg.Global.ContainerName)
	msg.WithNamespace(cfg.Global.Namespace)
	msg.WithEnvName(cfg.Global.EnvName)
	if cfg.Global.EnableSet == "Y" {
		msg.WithSetName(cfg.Global.FullSetName)
	}
	if len(cfg.Server.Service) > 0 {
		msg.WithCalleeServiceName(cfg.Server.Service[0].Name)
	} else {
		msg.WithCalleeApp(cfg.Server.App)
		msg.WithCalleeServer(cfg.Server.Server)
	}
	return ctx
}

// GetMetaData returns metadata from ctx by key.
func GetMetaData(ctx context.Context, key string) []byte {
	msg := codec.Message(ctx)
	if len(msg.ServerMetaData()) > 0 {
		return msg.ServerMetaData()[key]
	}
	return nil
}

// SetMetaData sets metadata which will be returned to upstream.
// This method is not thread-safe.
// Notice: SetMetaData can only be called in the server side rpc entry goroutine,
// not in goroutine that calls the client.
func SetMetaData(ctx context.Context, key string, val []byte) {
	msg := codec.Message(ctx)
	if len(msg.ServerMetaData()) > 0 {
		msg.ServerMetaData()[key] = val
		return
	}
	md := make(map[string][]byte)
	md[key] = val
	msg.WithServerMetaData(md)
}

// Request returns RequestProtocol from ctx.
// If the RequestProtocol not found, a new RequestProtocol will be created and returned.
func Request(ctx context.Context) *trpcpb.RequestProtocol {
	msg := codec.Message(ctx)
	request, ok := msg.ServerReqHead().(*trpcpb.RequestProtocol)
	if !ok {
		return &trpcpb.RequestProtocol{}
	}
	return request
}

// Response returns ResponseProtocol from ctx.
// If the ResponseProtocol not found, a new ResponseProtocol will be created and returned.
func Response(ctx context.Context) *trpcpb.ResponseProtocol {
	msg := codec.Message(ctx)
	response, ok := msg.ServerRspHead().(*trpcpb.ResponseProtocol)
	if !ok {
		return &trpcpb.ResponseProtocol{}
	}
	return response
}

// CloneContext copies the context to get a context that retains the value and doesn't cancel.
// This is used when the handler is processed asynchronously to detach the original timeout control
// and retains the original context information.
//
// After the trpc handler function returns, ctx will be canceled, and put the ctx's Msg back into pool,
// and the associated Metrics and logger will be released.
//
// Before starting a goroutine to run the handler function asynchronously,
// this method must be called to copy context, detach the original timeout control,
// and retain the information in Msg for Metrics.
//
// Retain the logger context for printing the associated log,
// keep other value in context, such as tracing context, etc.
func CloneContext(ctx context.Context) context.Context {
	oldMsg := codec.Message(ctx)
	newCtx, newMsg := codec.WithNewMessage(detach(ctx))
	codec.CopyMsg(newMsg, oldMsg)
	return newCtx
}

type detachedContext struct{ parent context.Context }

func detach(ctx context.Context) context.Context { return detachedContext{ctx} }

// Deadline implements context.Deadline
func (v detachedContext) Deadline() (time.Time, bool) { return time.Time{}, false }

// Done implements context.Done
func (v detachedContext) Done() <-chan struct{} { return nil }

// Err implements context.Err
func (v detachedContext) Err() error { return nil }

// Value implements context.Value
func (v detachedContext) Value(key interface{}) interface{} { return v.parent.Value(key) }

// GoAndWait provides safe concurrent handling. Per input handler, it starts a goroutine.
// Then it waits until all handlers are done and will recover if any handler panics.
// The returned error is the first non-nil error returned by one of the handlers.
// It can be set that non-nil error will be returned if the "key" handler fails while other handlers always
// return nil error.
func GoAndWait(handlers ...func() error) error {
	var (
		wg   sync.WaitGroup
		once sync.Once
		err  error
	)
	for _, f := range handlers {
		wg.Add(1)
		go func(handler func() error) {
			defer func() {
				if e := recover(); e != nil {
					buf := make([]byte, PanicBufLen)
					buf = buf[:runtime.Stack(buf, false)]
					log.Errorf("[PANIC]%v\n%s\n", e, buf)
					report.PanicNum.Incr()
					once.Do(func() {
						err = errs.New(errs.RetServerSystemErr, "panic found in call handlers")
					})
				}
				wg.Done()
			}()
			if e := handler(); e != nil {
				once.Do(func() {
					err = e
				})
			}
		}(f)
	}
	wg.Wait()
	return err
}

// Goer is the interface that launches a testable and safe goroutine.
type Goer interface {
	Go(ctx context.Context, timeout time.Duration, handler func(context.Context)) error
}

type asyncGoer struct {
	panicBufLen   int
	shouldRecover bool
	pool          *ants.PoolWithFunc
}

type goerParam struct {
	ctx     context.Context
	cancel  context.CancelFunc
	handler func(context.Context)
}

// NewAsyncGoer creates a goer that executes handler asynchronously with a goroutine when Go() is called.
func NewAsyncGoer(workerPoolSize int, panicBufLen int, shouldRecover bool) Goer {
	g := &asyncGoer{
		panicBufLen:   panicBufLen,
		shouldRecover: shouldRecover,
	}
	if workerPoolSize == 0 {
		return g
	}

	pool, err := ants.NewPoolWithFunc(workerPoolSize, func(args interface{}) {
		p := args.(*goerParam)
		g.handle(p.ctx, p.handler, p.cancel)
	})
	if err != nil {
		panic(err)
	}
	g.pool = pool
	return g
}

func (g *asyncGoer) handle(ctx context.Context, handler func(context.Context), cancel context.CancelFunc) {
	defer func() {
		if g.shouldRecover {
			if err := recover(); err != nil {
				buf := make([]byte, g.panicBufLen)
				buf = buf[:runtime.Stack(buf, false)]
				log.ErrorContextf(ctx, "[PANIC]%v\n%s\n", err, buf)
				report.PanicNum.Incr()
			}
		}
		cancel()
	}()
	handler(ctx)
}

func (g *asyncGoer) Go(ctx context.Context, timeout time.Duration, handler func(context.Context)) error {
	oldMsg := codec.Message(ctx)
	newCtx, newMsg := codec.WithNewMessage(detach(ctx))
	codec.CopyMsg(newMsg, oldMsg)
	newCtx, cancel := context.WithTimeout(newCtx, timeout)
	if g.pool != nil {
		p := &goerParam{
			ctx:     newCtx,
			cancel:  cancel,
			handler: handler,
		}
		return g.pool.Invoke(p)
	}
	go g.handle(newCtx, handler, cancel)
	return nil
}

type syncGoer struct {
}

// NewSyncGoer creates a goer that executes handler synchronously without cloning ctx when Go() is called.
// it's usually used for testing.
func NewSyncGoer() Goer {
	return &syncGoer{}
}

func (g *syncGoer) Go(ctx context.Context, timeout time.Duration, handler func(context.Context)) error {
	newCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	handler(newCtx)
	return nil
}

// DefaultGoer is an async goer without worker pool.
var DefaultGoer = NewAsyncGoer(0, PanicBufLen, true)

// Go launches a safer goroutine for async task inside rpc handler.
// it clones ctx and msg before the goroutine, and will recover and report metrics when the goroutine panics.
// you should set a suitable timeout to control the lifetime of the new goroutine to prevent goroutine leaks.
func Go(ctx context.Context, timeout time.Duration, handler func(context.Context)) error {
	return DefaultGoer.Go(ctx, timeout, handler)
}

// --------------- the following code is IP Config related -----------------//

// nicIP defines the parameters used to record the ip address (ipv4 & ipv6) of the nic.
type nicIP struct {
	nic  string
	ipv4 []string
	ipv6 []string
}

// netInterfaceIP maintains the nic name to nicIP mapping.
type netInterfaceIP struct {
	once sync.Once
	ips  map[string]*nicIP
}

// enumAllIP returns the nic name to nicIP mapping.
func (p *netInterfaceIP) enumAllIP() map[string]*nicIP {
	p.once.Do(func() {
		p.ips = make(map[string]*nicIP)
		interfaces, err := net.Interfaces()
		if err != nil {
			return
		}
		for _, i := range interfaces {
			p.addInterface(i)
		}
	})
	return p.ips
}

func (p *netInterfaceIP) addInterface(i net.Interface) {
	addrs, err := i.Addrs()
	if err != nil {
		return
	}
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		if ipNet.IP.To4() != nil {
			p.addIPv4(i.Name, ipNet.IP.String())
		} else if ipNet.IP.To16() != nil {
			p.addIPv6(i.Name, ipNet.IP.String())
		}
	}
}

// addIPv4 append ipv4 address
func (p *netInterfaceIP) addIPv4(nic string, ip4 string) {
	ips := p.getNicIP(nic)
	ips.ipv4 = append(ips.ipv4, ip4)
}

// addIPv6 append ipv6 address
func (p *netInterfaceIP) addIPv6(nic string, ip6 string) {
	ips := p.getNicIP(nic)
	ips.ipv6 = append(ips.ipv6, ip6)
}

// getNicIP returns nicIP by nic name.
func (p *netInterfaceIP) getNicIP(nic string) *nicIP {
	if _, ok := p.ips[nic]; !ok {
		p.ips[nic] = &nicIP{nic: nic}
	}
	return p.ips[nic]
}

// getIPByNic returns ip address by nic name.
// If the ipv4 addr is not empty, it will be returned.
// Otherwise, the ipv6 addr will be returned.
func (p *netInterfaceIP) getIPByNic(nic string) string {
	p.enumAllIP()
	if len(p.ips) <= 0 {
		return ""
	}
	if _, ok := p.ips[nic]; !ok {
		return ""
	}
	ip := p.ips[nic]
	if len(ip.ipv4) > 0 {
		return ip.ipv4[0]
	}
	if len(ip.ipv6) > 0 {
		return ip.ipv6[0]
	}
	return ""
}

// localIP records the local nic name->nicIP mapping.
var localIP = &netInterfaceIP{}

// getIP returns ip addr by nic name.
func getIP(nic string) string {
	ip := localIP.getIPByNic(nic)
	return ip
}

// deduplicate merges two slices.
// Order will be kept and duplication will be removed.
func deduplicate(a, b []string) []string {
	r := make([]string, 0, len(a)+len(b))
	m := make(map[string]bool)
	for _, s := range append(a, b...) {
		if _, ok := m[s]; !ok {
			m[s] = true
			r = append(r, s)
		}
	}
	return r
}
