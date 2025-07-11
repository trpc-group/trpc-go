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

//go:build linux || darwin || dragonfly || freebsd || netbsd || openbsd
// +build linux darwin dragonfly freebsd netbsd openbsd

package reuseport

import (
	"fmt"
	"html"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	httpServerOneResponse = "1"
	httpServerTwoResponse = "2"
)

var (
	httpServerOne = NewHTTPServer(httpServerOneResponse)
	httpServerTwo = NewHTTPServer(httpServerTwoResponse)
)

func NewHTTPServer(resp string) *httptest.Server {
	return httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, resp)
	}))
}

func TestNewReusablePortListener(t *testing.T) {
	listenerOne, err := NewReusablePortListener("tcp4", "localhost:10081")
	assert.Nil(t, err)
	defer listenerOne.Close()

	listenerTwo, err := NewReusablePortListener("tcp", "127.0.0.1:10081")
	assert.Nil(t, err)
	defer listenerTwo.Close()

	// devcloud ipv6地址无效
	_, err = NewReusablePortListener("tcp6", "[::x]:10081")
	if err == nil {
		t.Errorf("expect err, err[%v]", err)
	}

	listenerFour, err := NewReusablePortListener("tcp6", ":10081")
	assert.Nil(t, err)
	defer listenerFour.Close()

	listenerFive, err := NewReusablePortListener("tcp4", ":10081")
	assert.Nil(t, err)
	defer listenerFive.Close()

	listenerSix, err := NewReusablePortListener("tcp", ":10081")
	assert.Nil(t, err)
	defer listenerSix.Close()

	// proto invalid 非法协议
	_, err = NewReusablePortListener("xxx", "")
	if err == nil {
		t.Errorf("expect err")
	}
}

func TestListen(t *testing.T) {
	listenerOne, err := Listen("tcp4", "localhost:10081")
	assert.Nil(t, err)
	defer listenerOne.Close()

	listenerTwo, err := Listen("tcp", "127.0.0.1:10081")
	assert.Nil(t, err)
	defer listenerTwo.Close()

	listenerThree, err := Listen("tcp6", ":10081")
	assert.Nil(t, err)
	defer listenerThree.Close()

	listenerFour, err := Listen("tcp6", ":10081")
	assert.Nil(t, err)
	defer listenerFour.Close()

	listenerFive, err := Listen("tcp4", ":10081")
	assert.Nil(t, err)
	defer listenerFive.Close()

	listenerSix, err := Listen("tcp", ":10081")
	assert.Nil(t, err)
	defer listenerSix.Close()
}

func TestNewReusablePortServers(t *testing.T) {
	listenerOne, err := NewReusablePortListener("tcp4", "localhost:10081")
	assert.Nil(t, err)
	defer listenerOne.Close()

	// listenerTwo, err := NewReusablePortListener("tcp6", ":10081")
	listenerTwo, err := NewReusablePortListener("tcp", "localhost:10081")
	assert.Nil(t, err)
	defer listenerTwo.Close()

	httpServerOne.Listener = listenerOne
	httpServerTwo.Listener = listenerTwo

	httpServerOne.Start()
	httpServerTwo.Start()

	// Server One — First Response
	httpGet(httpServerOne.URL, httpServerOneResponse, httpServerTwoResponse, t)

	// Server Two — First Response
	httpGet(httpServerTwo.URL, httpServerOneResponse, httpServerTwoResponse, t)
	httpServerTwo.Close()

	// Server One — Second Response
	httpGet(httpServerOne.URL, httpServerOneResponse, "", t)

	// Server One — Third Response
	httpGet(httpServerOne.URL, httpServerOneResponse, "", t)
	httpServerOne.Close()
}

func httpGet(url string, expected1 string, expected2 string, t *testing.T) {
	resp, err := http.Get(url)
	assert.Nil(t, err)
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	assert.Nil(t, err)
	if string(body) != expected1 && string(body) != expected2 {
		t.Errorf("Expected %#v or %#v, got %#v.", expected1, expected2, string(body))
	}
}

func BenchmarkNewReusablePortListener(b *testing.B) {
	for i := 0; i < b.N; i++ {
		listener, err := NewReusablePortListener("tcp", ":10081")

		if err != nil {
			b.Error(err)
		} else {
			listener.Close()
		}
	}
}

func ExampleNewReusablePortListener() {
	listener, err := NewReusablePortListener("tcp", ":8881")
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	server := &http.Server{}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(os.Getgid())
		fmt.Fprintf(w, "Hello, %q\n", html.EscapeString(r.URL.Path))
	})

	panic(server.Serve(listener))
}

// TestBoundaryCase 一些边界条件覆盖
func TestBoundaryCase(t *testing.T) {
	proto, err := determineTCPProto("tcp", &net.TCPAddr{})
	if proto != "tcp" {
		t.Errorf("proto not tcp")
	}
	assert.Nil(t, err)
	_, err = determineTCPProto("udp", &net.TCPAddr{})
	if err == nil {
		t.Errorf("expect error")
	}

	// getTCPAddr 边界
	if _, _, err := getTCPAddr("udp", "localhost:8001"); err == nil {
		t.Error("expect error")
	}

	// ipv6 zone id，不存在的网卡
	addr := &net.TCPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Zone: "ethx",
	}
	_, _, err = getTCP6Sockaddr(addr)
	assert.NotNil(t, err)

	// udp ipv6
	udpAddr := &net.UDPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Zone: "ethx",
	}
	_, _, err = getUDP6Sockaddr(udpAddr)
	assert.NotNil(t, err)

	// ResolveUDPAddr failed
	_, _, err = getUDPSockaddr("xxx", ":10086")
	assert.NotNil(t, err)
}

func TestCreateReusableFd(t *testing.T) {
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, syscall.IPPROTO_TCP)
	assert.Nil(t, err)
	assert.NotZero(t, fd)

	// set opt failed, bad fd: -1
	sa := &syscall.SockaddrInet4{}
	err = createReusableFd(-1, sa)
	assert.NotNil(t, err)

	// set opt failed
	oldReusePort := reusePort
	defer func() {
		reusePort = oldReusePort
	}()
	reusePort = 0
	err = createReusableFd(fd, sa)
	assert.NotNil(t, err)

	// file descriptor invalid
	_, err = createReusableListener(10081, "tcp", "localhost:8001")
	assert.NotNil(t, err)
}
