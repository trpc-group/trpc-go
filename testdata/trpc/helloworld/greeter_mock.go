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

// Code generated by MockGen. DO NOT EDIT.
// Source: trpc.group/trpc-go/trpc-go/testdata/trpc/helloworld (interfaces: GreeterClientProxy)

// Package helloworld is a generated GoMock package.
package helloworld

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"

	client "trpc.group/trpc-go/trpc-go/client"
)

// MockGreeterClientProxy is a mock of GreeterClientProxy interface
type MockGreeterClientProxy struct {
	ctrl     *gomock.Controller
	recorder *MockGreeterClientProxyMockRecorder
}

// MockGreeterClientProxyMockRecorder is the mock recorder for MockGreeterClientProxy
type MockGreeterClientProxyMockRecorder struct {
	mock *MockGreeterClientProxy
}

// NewMockGreeterClientProxy creates a new mock instance
func NewMockGreeterClientProxy(ctrl *gomock.Controller) *MockGreeterClientProxy {
	mock := &MockGreeterClientProxy{ctrl: ctrl}
	mock.recorder = &MockGreeterClientProxyMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockGreeterClientProxy) EXPECT() *MockGreeterClientProxyMockRecorder {
	return m.recorder
}

// SayHello mocks base method
func (m *MockGreeterClientProxy) SayHello(arg0 context.Context, arg1 *HelloRequest, arg2 ...client.Option) (*HelloReply, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "SayHello", varargs...)
	ret0, _ := ret[0].(*HelloReply)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SayHello indicates an expected call of SayHello
func (mr *MockGreeterClientProxyMockRecorder) SayHello(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SayHello", reflect.TypeOf((*MockGreeterClientProxy)(nil).SayHello), varargs...)
}

// SayHi mocks base method
func (m *MockGreeterClientProxy) SayHi(arg0 context.Context, arg1 *HelloRequest, arg2 ...client.Option) (*HelloReply, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "SayHi", varargs...)
	ret0, _ := ret[0].(*HelloReply)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SayHi indicates an expected call of SayHi
func (mr *MockGreeterClientProxyMockRecorder) SayHi(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SayHi", reflect.TypeOf((*MockGreeterClientProxy)(nil).SayHi), varargs...)
}
