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

package filter_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"

	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/client/mockclient"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/rpcz"
	pb "trpc.group/trpc-go/trpc-go/testdata"
	"trpc.group/trpc-go/trpc-go/testdata/restful/bookstore"
)

func echoServerHandle(ctx context.Context, req interface{}) (interface{}, error) {
	return req, nil
}

func echoClientHandle(ctx context.Context, req interface{}, rsp interface{}) error {
	if reply, ok := rsp.(*pb.HelloReply); ok {
		reply.Msg = "echo client handle"
	}
	return nil
}

func echoHandle(ctx context.Context, req interface{}, rsp interface{}) error {
	preq := req.(*string)
	prsp := rsp.(*string)
	*prsp = *preq

	return nil
}

func makeLabelFilter(name string) filter.Filter {
	return func(ctx context.Context, req interface{}, rsp interface{}, f filter.HandleFunc) (err error) {
		// pre-logic: rewrite req to name->req
		*(req.(*string)) = name + "->" + *req.(*string)
		f(ctx, req, rsp)
		// post-logic: rewrite rsp to rsp<-name
		*(rsp.(*string)) = *rsp.(*string) + "<-" + name
		return nil
	}
}

func TestNoopFilter(t *testing.T) {
	req := "echo"
	rsp := ""
	err := filter.NoopFilter(context.Background(), &req, &rsp, echoHandle)
	assert.Nil(t, err)
	assert.Equal(t, rsp, req)
	rspIntf, err := filter.NoopServerFilter(context.Background(), &req, echoServerHandle)
	assert.Nil(t, err)
	assert.Equal(t, rsp, *rspIntf.(*string))
}

func TestFilterChain_Handle(t *testing.T) {
	req := "echo"
	rsp := ""
	// noopFilter
	{
		fc := filter.Chain{}
		err := fc.Handle(context.Background(), &req, &rsp, echoHandle)
		assert.Nil(t, err)
		assert.Equal(t, rsp, req)
		svrfc := filter.ServerChain{}
		err = svrfc.Handle(context.Background(), &req, &rsp, echoHandle)
		assert.Nil(t, err)
		assert.Equal(t, rsp, req)
		rspIntf, err := svrfc.Filter(context.Background(), &req, echoServerHandle)
		assert.Nil(t, err)
		assert.Equal(t, rsp, *rspIntf.(*string))
	}

	// oneFilter
	{
		fc := filter.Chain{filter.NoopFilter}
		err := fc.Handle(context.Background(), &req, &rsp, echoHandle)
		assert.Nil(t, err)
		assert.Equal(t, rsp, req)
		svrfc := filter.ServerChain{filter.NoopServerFilter}
		err = svrfc.Handle(context.Background(), &req, &rsp, echoHandle)
		assert.Nil(t, err)
		assert.Equal(t, rsp, req)
		rspIntf, err := svrfc.Filter(context.Background(), &req, echoServerHandle)
		assert.Nil(t, err)
		assert.Equal(t, rsp, *rspIntf.(*string))
	}

	// multiFilter
	{
		fc := filter.Chain{filter.NoopFilter, filter.NoopFilter, filter.NoopFilter}
		err := fc.Handle(context.Background(), &req, &rsp, echoHandle)
		assert.Nil(t, err)
		assert.Equal(t, rsp, req)
	}

	// one labelFilter
	{
		fc := filter.Chain{makeLabelFilter("x")}
		err := fc.Handle(context.Background(), &req, &rsp, echoHandle)
		assert.Nil(t, err)
		assert.Equal(t, req, "x->echo")
		assert.Equal(t, rsp, "x->echo<-x")
	}

	// multiple hybrid filters
	{
		req = "echo"
		rsp = ""
		fc := filter.Chain{
			makeLabelFilter("x"),
			filter.NoopFilter,
			makeLabelFilter("y"),
			filter.NoopFilter,
			makeLabelFilter("z"),
		}
		err := fc.Handle(context.Background(), &req, &rsp, echoHandle)
		assert.Nil(t, err)
		assert.Equal(t, req, "z->y->x->echo")
		assert.Equal(t, rsp, "z->y->x->echo<-z<-y<-x")
	}
}

func TestClientChain_Filter(t *testing.T) {
	oldGlobalRPCZ := rpcz.GlobalRPCZ
	defer func() {
		rpcz.GlobalRPCZ = oldGlobalRPCZ
	}()
	rpcz.GlobalRPCZ = rpcz.NewRPCZ(&rpcz.Config{Fraction: 1.0, Capacity: 1000})
	s, ender, ctx := rpcz.NewSpanContext(context.Background(), "before filter")
	defer ender.End()
	s.SetAttribute(rpcz.TRPCAttributeFilterNames, []string{"filter1", "filter2"})
	type args struct {
		ctx  context.Context
		req  interface{}
		rsp  interface{}
		next filter.ClientHandleFunc
	}
	tests := []struct {
		name    string
		c       filter.ClientChain
		args    args
		wantErr assert.ErrorAssertionFunc
		wantRsp interface{}
	}{
		{
			name: "len(FilterNames) greater than len(ClientChain)",
			c: filter.ClientChain{
				func(ctx context.Context, req, rsp interface{}, next filter.ClientHandleFunc) error {
					{
						rsp := rsp.(*[]string)
						*rsp = append(*rsp, rpcz.SpanFromContext(ctx).Name())
					}
					return next(ctx, req, rsp)
				},
			},
			args: args{
				ctx: ctx,
				rsp: &[]string{},
				next: func(ctx context.Context, req, rsp interface{}) error {
					return nil
				},
			},
			wantErr: assert.NoError,
			wantRsp: &[]string{"filter1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.wantErr(
				t,
				tt.c.Filter(tt.args.ctx, tt.args.req, tt.args.rsp, tt.args.next),
				fmt.Sprintf("Filter(%v, %v, %v)", tt.args.ctx, tt.args.req, tt.args.rsp),
			)
			assert.Equal(t, tt.wantRsp, tt.args.rsp)
		})
	}
}

func TestServerServer_Filter(t *testing.T) {
	oldGlobalRPCZ := rpcz.GlobalRPCZ
	defer func() {
		rpcz.GlobalRPCZ = oldGlobalRPCZ
	}()
	rpcz.GlobalRPCZ = rpcz.NewRPCZ(&rpcz.Config{Fraction: 1.0, Capacity: 1000})
	s, ender, ctx := rpcz.NewSpanContext(context.Background(), "before filter")
	defer ender.End()
	s.SetAttribute(rpcz.TRPCAttributeFilterNames, []string{"filter1", "filter2"})

	filterNames := func(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (rsp interface{}, err error) {
		rsp, err = next(ctx, req)
		return append([]string{rpcz.SpanFromContext(ctx).Name()}, rsp.([]string)...), err
	}

	type args struct {
		ctx  context.Context
		req  interface{}
		next filter.ServerHandleFunc
	}
	tests := []struct {
		name    string
		c       filter.ServerChain
		args    args
		want    interface{}
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "len(FilterNames) greater than len(ServerChain)",
			c: filter.ServerChain{
				filterNames,
			},
			args: args{
				ctx: ctx,
				next: func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
					return []string{}, nil
				},
			},
			want:    []string{"filter1"},
			wantErr: assert.NoError,
		},
		{
			name: "len(FilterNames) less than len(ServerChain)",
			c: filter.ServerChain{
				filterNames, filterNames, filterNames,
			},
			args: args{
				ctx: ctx,
				next: func(ctx context.Context, req interface{}) (rsp interface{}, err error) {
					return []string{}, nil
				},
			},
			want:    []string{"filter1", "filter2", "unknown"},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.c.Filter(tt.args.ctx, tt.args.req, tt.args.next)
			if !tt.wantErr(t, err, fmt.Sprintf("Filter(%v, %v)", tt.args.ctx, tt.args.req)) {
				return
			}
			assert.Equalf(t, tt.want, got, "Filter(%v, %v)", tt.args.ctx, tt.args.req)
		})
	}
}

func TestChain_ConcurrentHandle(t *testing.T) {
	const concurrentN = 4
	var calledTimes [concurrentN]int32
	cc := filter.ClientChain{
		func(ctx context.Context, req interface{}, rsp interface{}, f filter.ClientHandleFunc) error {
			atomic.AddInt32(&calledTimes[0], 1)
			return f(ctx, req, rsp)
		},
		func(ctx context.Context, req interface{}, rsp interface{}, f filter.ClientHandleFunc) error {
			atomic.AddInt32(&calledTimes[1], 1)
			var eg errgroup.Group
			for i := 0; i < concurrentN; i++ {
				eg.Go(func() error {
					return f(ctx, req, rsp)
				})
			}
			return eg.Wait()
		},
		func(ctx context.Context, req interface{}, rsp interface{}, f filter.ClientHandleFunc) error {
			atomic.AddInt32(&calledTimes[2], 1)
			return f(ctx, req, rsp)
		},
		func(ctx context.Context, req interface{}, rsp interface{}, f filter.ClientHandleFunc) error {
			atomic.AddInt32(&calledTimes[3], 1)
			return f(ctx, req, rsp)
		},
	}

	if err := cc.Filter(context.Background(), nil, nil,
		func(ctx context.Context, req, rsp interface{}) (err error) {
			return nil
		}); err != nil {
		t.Errorf("cc.Filter(%v, %v, ...) gotErr = %v, wantErr = %v", nil, nil, err, nil)
	}
	if got, want := atomic.LoadInt32(&calledTimes[0]), int32(1); got != want {
		t.Errorf("atomic.LoadInt32(%p) got = %d, want = %d", &calledTimes[0], got, want)
	}
	if got, want := atomic.LoadInt32(&calledTimes[1]), int32(1); got != want {
		t.Errorf("atomic.LoadInt32(%p) got = %d, want = %d", &calledTimes[1], got, want)
	}
	if got, want := atomic.LoadInt32(&calledTimes[2]), int32(concurrentN); got != want {
		t.Errorf("atomic.LoadInt32(%p) got = %d, want = %d", &calledTimes[2], got, want)
	}
	if got, want := atomic.LoadInt32(&calledTimes[3]), int32(concurrentN); got != want {
		t.Errorf("atomic.LoadInt32(%p) got = %d, want = %d", &calledTimes[3], got, want)
	}
}

func TestGetClient(t *testing.T) {
	filter.Register("noop", filter.NoopFilter, filter.NoopFilter)
	f := filter.GetClient("noop")
	assert.NotNil(t, f)
}

func TestGetServer(t *testing.T) {
	filter.Register("noop", filter.NoopFilter, filter.NoopFilter)
	f := filter.GetServer("noop")
	assert.NotNil(t, f)
}

func serverFilter(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (interface{}, error) {
	return next(ctx, req)
}
func svrFilter() filter.ServerFilter {
	return serverFilter
}
func clientFilter(ctx context.Context, req, rsp interface{}, next filter.ClientHandleFunc) error {
	return next(ctx, req, rsp)
}
func cliFilter() filter.ClientFilter {
	return clientFilter
}
func nilRspClientFilter(ctx context.Context, req, rsp interface{}, next filter.ClientHandleFunc) error {
	if rsp == nil {
		return errors.New("client filter rsp is nil")
	}
	return next(ctx, req, rsp)
}
func TestConvert(t *testing.T) {
	req := "echo"
	rsp := "echo"
	f := filter.ConvertToServerFilter("svr1", serverFilter)
	assert.NotNil(t, f)
	rspIntf, err := f(context.Background(), &req, echoServerHandle)
	assert.Nil(t, err)
	assert.Equal(t, rsp, *rspIntf.(*string))

	f = filter.ConvertToServerFilter("svr2", svrFilter())
	assert.NotNil(t, f)

	f = filter.ConvertToServerFilter("cli1", clientFilter)
	assert.NotNil(t, f)
	rspIntf, err = f(context.Background(), &req, echoServerHandle)
	assert.Nil(t, err)
	assert.Equal(t, rsp, *rspIntf.(*string))

	f = filter.ConvertToServerFilter("cli2", cliFilter())
	assert.NotNil(t, f)

	f = filter.ConvertToServerFilter("cli3", nilRspClientFilter)
	assert.NotNil(t, f)
	rspIntf, err = f(context.Background(), &req, echoServerHandle)
	assert.NotNil(t, err)

	f = filter.ConvertToServerFilter("nil", nil)
	assert.Nil(t, f)

	defer func() {
		err := recover()
		assert.NotNil(t, err)
	}()
	f = filter.ConvertToServerFilter("unsupported", echoHandle)
	assert.Nil(t, f)
}

func serverNewPbRspFilter(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (interface{}, error) {
	return &pb.HelloReply{Msg: "server new pb rsp filter"}, nil
}

type Reply struct {
	Msg string `json:"msg"`
}

func serverNewRspFilter(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (interface{}, error) {
	return &Reply{Msg: "server new rsp filter"}, nil
}

// ReplyChan is used to simulate json marshal failures.
type ReplyChan struct {
	Msg string `json:"msg"`
	C   chan int
}

func serverNewRspJSONFailFilter(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (interface{}, error) {
	return &ReplyChan{Msg: "server new json rsp filter"}, nil
}

func serverNewRspJSONStrFailFilter(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (interface{}, error) {
	return string(`{xxx}`), nil
}

func serverNewRspJSONStrFilter(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (interface{}, error) {
	r := &Reply{Msg: "server new json str rsp filter"}
	rStr, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	return string(rStr), nil
}

func serverNewRspJSONByteFilter(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (interface{}, error) {
	r := &Reply{Msg: "server new json byte rsp filter"}
	rStr, err := json.Marshal(r)
	if err != nil {
		return nil, err
	}
	return rStr, nil
}

type ReplyCopier struct {
	Msg string `json:"msg"`
}

func (r *ReplyCopier) CopyTo(dst interface{}) error {
	dst.(*ReplyOmit).Msg = r.Msg
	return nil
}

func serverNewRspCopierFilter(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (interface{}, error) {
	return &ReplyCopier{Msg: "server new copier rsp filter"}, nil
}

type ReplyOmit struct {
	Msg  string `json:"msg"`
	Code int    `json:"code,omitempty"`
}

func serverNewRspOmitFilter(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (interface{}, error) {
	return &ReplyOmit{Msg: "server new json omit rsp filter"}, nil
}

func genRspHijackedServerFilter(cli client.Client) filter.ServerFilter {
	return func(_ context.Context, req interface{}, _ filter.ServerHandleFunc) (interface{}, error) {
		// calldown
		hijackedRsp := &json.RawMessage{}
		_ = cli.Invoke(context.Background(), req, hijackedRsp,
			client.WithSerializationType(codec.SerializationTypeJSON))
		return hijackedRsp, nil
	}
}

func TestServerChainHandleCopyRsp(t *testing.T) {
	req := &pb.HelloRequest{}
	rsp := &pb.HelloReply{}
	fc := filter.ServerChain{filter.NoopServerFilter}
	err := fc.Handle(context.Background(), req, rsp, echoClientHandle)
	assert.Nil(t, err)
	assert.Equal(t, "echo client handle", rsp.GetMsg())

	fc = filter.ServerChain{filter.NoopServerFilter, filter.ServerFilter(serverNewPbRspFilter)}
	err = fc.Handle(context.Background(), req, rsp, echoClientHandle)
	assert.Nil(t, err)
	assert.Equal(t, "server new pb rsp filter", rsp.GetMsg())

	fc = filter.ServerChain{filter.NoopServerFilter, filter.ServerFilter(serverNewRspFilter)}
	err = fc.Handle(context.Background(), req, rsp, echoClientHandle)
	assert.Nil(t, err)
	assert.Equal(t, "server new rsp filter", rsp.GetMsg())

	fc = filter.ServerChain{filter.NoopServerFilter, filter.ServerFilter(serverNewRspJSONFailFilter)}
	err = fc.Handle(context.Background(), req, rsp, echoClientHandle)
	assert.NotNil(t, err)

	rspOmit := &ReplyOmit{Msg: "test rsp msg", Code: 100}
	fc = filter.ServerChain{filter.NoopServerFilter, filter.ServerFilter(serverNewRspOmitFilter)}
	err = fc.Handle(context.Background(), req, rspOmit, echoClientHandle)
	assert.Nil(t, err)
	assert.Equal(t, "server new json omit rsp filter", rspOmit.Msg)
	assert.Equal(t, 100, rspOmit.Code)

	rspCopierOmit := &ReplyOmit{Msg: "test rsp copier msg", Code: 100}
	fc = filter.ServerChain{filter.NoopServerFilter, filter.ServerFilter(serverNewRspCopierFilter)}
	err = fc.Handle(context.Background(), req, rspCopierOmit, echoClientHandle)
	assert.Nil(t, err)
	assert.Equal(t, "server new copier rsp filter", rspCopierOmit.Msg)
	assert.Equal(t, 100, rspCopierOmit.Code)
}

func TestHijackServerRsp(t *testing.T) {
	// in new server filter, you dont know the rsp type, because rsp is the returned val.
	// func(ctx context.Context, req interface{}, next ServerHandleFunc) (rsp interface{}, err error)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockCli := mockclient.NewMockClient(ctrl)
	mockCli.EXPECT().Invoke(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, _, reqBody interface{}, _ ...client.Option) error {
			book := &bookstore.Book{Id: 123}
			data, _ := json.Marshal(book) // because call with json
			return codec.Unmarshal(codec.SerializationTypeJSON, data, reqBody)
		})
	oriReq := &bookstore.GetBookRequest{Shelf: 100, Book: 123}
	// cur svr filter chain proc
	oriRsp := &bookstore.Book{}
	fc := filter.ServerChain{filter.NoopServerFilter, filter.ServerFilter(genRspHijackedServerFilter(mockCli))}
	err := fc.Handle(context.Background(), oriReq, oriRsp, echoClientHandle)
	assert.Nil(t, err)
	assert.Equal(t, oriRsp.Id, int64(123))
}
