// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package main

import (
	"context"
	"reflect"
	"testing"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld"

	"github.com/golang/mock/gomock"
)

func Test_greeterServiceImpl_SayHello(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cli := helloworld.NewMockGreeterClientProxy(ctrl)
	call := cli.EXPECT().SayHi(gomock.Any(), gomock.Any()).AnyTimes()
	ctx := trpc.BackgroundContext()
	type fields struct {
		proxy helloworld.GreeterClientProxy
	}
	type args struct {
		ctx context.Context
		req *helloworld.HelloRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *helloworld.HelloReply
		wantErr bool
		setup   func()
	}{
		{
			name: "test SayHi success",
			fields: fields{
				proxy: cli,
			},
			args: args{
				ctx: ctx,
				req: &helloworld.HelloRequest{
					Msg: "success hello req",
				},
			},
			wantErr: false,
			want: &helloworld.HelloReply{
				Msg: "Hello mock hi rsp",
			},
			setup: func() {
				call.Return(&helloworld.HelloReply{Msg: "mock hi rsp"}, nil)
			},
		},
		{
			name: "test SayHi fail",
			fields: fields{
				proxy: cli,
			},
			args: args{
				ctx: ctx,
				req: &helloworld.HelloRequest{
					Msg: "fail hello req",
				},
			},
			want:    nil,
			wantErr: true,
			setup: func() {
				call.Return(nil, errs.New(101, "timeout"))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			s := &greeterServiceImpl{
				proxy: tt.fields.proxy,
			}
			got, err := s.SayHello(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("greeterServiceImpl.SayHello() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("greeterServiceImpl.SayHello() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_greeterServiceImpl_SayHi(t *testing.T) {
	ctx := trpc.BackgroundContext()
	type fields struct {
		proxy helloworld.GreeterClientProxy
	}
	type args struct {
		ctx context.Context
		req *helloworld.HelloRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *helloworld.HelloReply
		wantErr bool
	}{
		{
			name: "test success",
			args: args{
				ctx: ctx,
				req: &helloworld.HelloRequest{
					Msg: "success hi req",
				},
			},
			want: &helloworld.HelloReply{
				Msg: "Hi success hi req",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &greeterServiceImpl{
				proxy: tt.fields.proxy,
			}
			got, err := s.SayHi(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("greeterServiceImpl.SayHi() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("greeterServiceImpl.SayHi() = %v, want %v", got, tt.want)
			}
		})
	}
}
