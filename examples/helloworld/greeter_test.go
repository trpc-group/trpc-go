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

package main

import (
	"context"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/errs"
	pb "trpc.group/trpc-go/trpc-go/testdata"
)

func Test_greeterServiceImpl_SayHello(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	cli := pb.NewMockGreeterClientProxy(ctrl)
	call := cli.EXPECT().SayHi(gomock.Any(), gomock.Any()).AnyTimes()
	ctx := trpc.BackgroundContext()
	type fields struct {
		proxy pb.GreeterClientProxy
	}
	type args struct {
		ctx context.Context
		req *pb.HelloRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.HelloReply
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
				req: &pb.HelloRequest{
					Msg: "success hello req",
				},
			},
			wantErr: false,
			want: &pb.HelloReply{
				Msg: "Hello mock hi rsp",
			},
			setup: func() {
				call.Return(&pb.HelloReply{Msg: "mock hi rsp"}, nil)
			},
		},
		{
			name: "test SayHi fail",
			fields: fields{
				proxy: cli,
			},
			args: args{
				ctx: ctx,
				req: &pb.HelloRequest{
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
		proxy pb.GreeterClientProxy
	}
	type args struct {
		ctx context.Context
		req *pb.HelloRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *pb.HelloReply
		wantErr bool
	}{
		{
			name: "test success",
			args: args{
				ctx: ctx,
				req: &pb.HelloRequest{
					Msg: "success hi req",
				},
			},
			want: &pb.HelloReply{
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
