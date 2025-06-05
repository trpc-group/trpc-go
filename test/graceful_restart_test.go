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

package test

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/server"
	testpb "trpc.group/trpc-go/trpc-go/test/protocols"
)

type gracefulRestartTestData struct {
	network    string
	target     string
	sourceFile string
	configFile string
	binaryFile string
}

func (s *TestSuite) TestServerGracefulRestart() {
	tests := []gracefulRestartTestData{
		{
			network:    "tcp",
			target:     "ip://127.0.0.1:17777",
			sourceFile: "./gracefulrestart/trpc/server.go",
			configFile: "./gracefulrestart/trpc/trpc_go_tcp.yaml",
			binaryFile: "./gracefulrestart/trpc/server.o",
		},
		{
			network:    "udp",
			target:     "ip://127.0.0.1:17777",
			sourceFile: "./gracefulrestart/trpc/server.go",
			configFile: "./gracefulrestart/trpc/trpc_go_udp.yaml",
			binaryFile: "./gracefulrestart/trpc/server.o",
		},
	}
	for _, tt := range tests {
		s.Run("ServerGracefulRestartIsIdempotent"+tt.network, func() {
			s.testServerGracefulRestartIsIdempotent(tt)
		})
		s.Run("SendNonGracefulRestartSignal"+tt.network, func() {
			s.testSendNonGracefulRestartSignal(tt)
		})
		s.Run("ServerGracefulRestartContinuesHandling"+tt.network, func() {
			s.testServerGracefulRestartContinuesHandling(tt)
		})
	}
	tests = []gracefulRestartTestData{
		{
			network:    "tcp",
			target:     "ip://127.0.0.1:17777",
			sourceFile: "./gracefulrestart/trpc/server.go",
			configFile: "./gracefulrestart/trpc/trpc_go_emptyip_tcp.yaml",
			binaryFile: "./gracefulrestart/trpc/server.o",
		},
		{
			network:    "udp",
			target:     "ip://127.0.0.1:17777",
			sourceFile: "./gracefulrestart/trpc/server.go",
			configFile: "./gracefulrestart/trpc/trpc_go_emptyip_udp.yaml",
			binaryFile: "./gracefulrestart/trpc/server.o",
		},
	}
	for _, tt := range tests {
		s.Run("GracefulRestartForEmptyIP"+tt.network, func() {
			s.testGracefulRestartForEmptyIP(tt)
		})
	}
	s.Run("OldStreamFailedButNewStreamOk", func() {
		s.testServerGracefulRestartOldStreamFailedButNewStreamOk()
	})
}

func (s *TestSuite) testServerGracefulRestartIsIdempotent(testData gracefulRestartTestData) {
	cmd, err := startServerFromBash(
		testData.sourceFile,
		testData.configFile,
		testData.binaryFile,
	)
	require.Nil(s.T(), err)
	defer func() {
		require.Nil(s.T(), exec.Command("rm", testData.binaryFile).Run())
		require.Nil(s.T(), cmd.Process.Kill())
		time.Sleep(time.Second)
	}()

	sp, err := getServerProcessByEmptyCall(testData.network, testData.target)
	require.Nil(s.T(), err)
	pid := sp.Pid
	for i := 0; i < 3; i++ {
		require.Nil(s.T(), sp.Signal(server.DefaultServerGracefulSIG))
		// Wait until server has restarted gracefully.
		time.Sleep(5 * time.Second)
		sp, err = getServerProcessByEmptyCall(testData.network, testData.target)
		require.Nil(s.T(), err)
		require.NotEqual(s.T(), pid, sp.Pid)
		pid = sp.Pid
	}
	// Kill server and wait for it to graceful exit.
	require.Nil(s.T(), sp.Kill())
	time.Sleep(time.Second)
}

func (s *TestSuite) testServerGracefulRestartContinuesHandling(testData gracefulRestartTestData) {
	cmd, err := startServerFromBash(
		testData.sourceFile,
		testData.configFile,
		testData.binaryFile,
	)
	assert.Nil(s.T(), err)
	defer func() {
		assert.Nil(s.T(), exec.Command("rm", testData.binaryFile).Run())
		assert.Nil(s.T(), cmd.Process.Kill())
		time.Sleep(time.Second)
	}()

	done := make(chan struct{})
	go func() {
		for i := 0; i < 10_0000; i++ {
			req := fmt.Sprintf("%v", i)
			rsp, err := echo(req, testData.network, testData.target)
			assert.Nil(s.T(), err)
			assert.Equal(s.T(), req, rsp)
		}
		done <- struct{}{}
	}()

	time.Sleep(time.Second)
	sp, err := getServerProcessByEmptyCall(testData.network, testData.target)
	require.Nil(s.T(), err)
	oldPid := sp.Pid
	require.Nil(s.T(), sp.Signal(server.DefaultServerGracefulSIG))
	time.Sleep(5 * time.Second)

	<-done
	sp, err = getServerProcessByEmptyCall(testData.network, testData.target)
	require.Nil(s.T(), err)
	newPid := sp.Pid
	require.NotEqual(s.T(), oldPid, newPid)
	// Kill server and wait for it to graceful exit.
	require.Nil(s.T(), sp.Kill())
	time.Sleep(time.Second)
}

func (s *TestSuite) testServerGracefulRestartOldStreamFailedButNewStreamOk() {
	const (
		binaryFile = "./gracefulrestart/streaming/server.o"
		sourceFile = "./gracefulrestart/streaming/server.go"
		configFile = "./gracefulrestart/streaming/trpc_go.yaml"
	)

	cmd, err := startServerFromBash(
		sourceFile,
		configFile,
		binaryFile,
	)
	require.Nil(s.T(), err)
	defer func() {
		require.Nil(s.T(), exec.Command("rm", binaryFile).Run())
		require.Nil(s.T(), cmd.Process.Kill())
		time.Sleep(time.Second)
	}()

	respParams := []*testpb.ResponseParameters{
		{
			Size: int32(1),
		},
	}
	payload, err := newPayload(testpb.PayloadType_COMPRESSABLE, int32(1))
	require.Nil(s.T(), err)
	req := &testpb.StreamingOutputCallRequest{
		ResponseType:       testpb.PayloadType_COMPRESSABLE,
		ResponseParameters: respParams,
		Payload:            payload,
	}

	doFullDuplexCall := func() (*os.Process, testpb.TestStreaming_FullDuplexCallClient) {
		c := testpb.NewTestStreamingClientProxy(client.WithTarget("ip://127.0.0.1:17778"))
		cs, err := c.FullDuplexCall(trpc.BackgroundContext())
		require.Nil(s.T(), err)

		require.Nil(s.T(), cs.Send(req))
		rsp, err := cs.Recv()
		require.Nil(s.T(), err)

		serverPid, err := strconv.Atoi(string(rsp.GetPayload().GetBody()))
		require.Nil(s.T(), err)
		sp, err := os.FindProcess(serverPid)
		require.Nil(s.T(), err)
		return sp, cs
	}

	sp1, cs1 := doFullDuplexCall()
	pid1 := sp1.Pid
	require.Nil(s.T(), sp1.Signal(server.DefaultServerGracefulSIG))
	// Wait until server has restarted gracefully.
	time.Sleep(5 * time.Second)

	err = cs1.Send(req)
	require.Equal(s.T(), errs.RetServerSystemErr, errs.Code(err), "full err: %+v", err)
	require.Contains(s.T(), errs.Msg(err), "Connection is Closed")

	sp2, cs2 := doFullDuplexCall()
	require.Nil(s.T(), cs2.Send(req))

	require.NotEqual(s.T(), pid1, sp2.Pid)
	// Kill server and wait for it to graceful exit.
	require.Nil(s.T(), sp2.Kill())
	time.Sleep(time.Second)
}

func (s *TestSuite) testSendNonGracefulRestartSignal(testData gracefulRestartTestData) {
	s.Run("Send Default Server Close Signal", func() {
		cmd, err := startServerFromBash(
			testData.sourceFile,
			testData.configFile,
			testData.binaryFile,
		)
		require.Nil(s.T(), err)
		defer func() {
			require.Nil(s.T(), exec.Command("rm", testData.binaryFile).Run())
			require.Nil(s.T(), cmd.Process.Kill())
			time.Sleep(time.Second)
		}()
		sp, err := getServerProcessByEmptyCall(testData.network, testData.target)
		require.Nil(s.T(), err)

		r := rand.New(rand.NewSource(time.Now().Unix()))
		closeSignal := server.DefaultServerCloseSIG[r.Intn(len(server.DefaultServerCloseSIG))]
		require.Nil(s.T(), sp.Signal(closeSignal))
		time.Sleep(time.Second)
		for {
			if _, err := getServerProcessByEmptyCall(testData.network, testData.target); err != nil {
				require.Conditionf(s.T(), func() bool {
					code := errs.Code(err)
					switch testData.network {
					case "tcp":
						// Both the following code are possible due to the implementation of connection pool.
						return code == errs.RetClientReadFrameErr || code == errs.RetClientConnectFail
					case "udp":
						return code == errs.RetClientNetErr || code == errs.RetClientFullLinkTimeout
					default:
						return false
					}
				}, "full err: %+v", err)
				return
			}
		}
	})
	s.Run("Send Non Close Signal", func() {
		cmd, err := startServerFromBash(
			testData.sourceFile,
			testData.configFile,
			testData.binaryFile,
		)
		require.Nil(s.T(), err)
		defer func() {
			require.Nil(s.T(), exec.Command("rm", testData.binaryFile).Run())
			require.Nil(s.T(), cmd.Process.Kill())
			time.Sleep(time.Second)
		}()

		sp, err := getServerProcessByEmptyCall(testData.network, testData.target)
		require.Nil(s.T(), err)
		pid := sp.Pid
		for i := 0; i < 3; i++ {
			require.Nil(s.T(), sp.Signal(syscall.SIGUSR1))
			sp, err = getServerProcessByEmptyCall(testData.network, testData.target)
			require.Equal(s.T(), pid, sp.Pid)
			require.Nil(s.T(), err)
		}
		// Kill server and wait for it to graceful exit.
		require.Nil(s.T(), sp.Kill())
		time.Sleep(time.Second)
	})
}

func (s *TestSuite) testGracefulRestartForEmptyIP(testData gracefulRestartTestData) {
	cmd, err := startServerFromBash(
		testData.sourceFile,
		testData.configFile,
		testData.binaryFile,
	)
	require.Nil(s.T(), err)
	defer func() {
		require.Nil(s.T(), exec.Command("rm", testData.binaryFile).Run())
		require.Nil(s.T(), cmd.Process.Kill())
		time.Sleep(time.Second)
	}()

	sp, err := getServerProcessByEmptyCall(testData.network, testData.target)
	require.Nil(s.T(), err)
	pid := sp.Pid
	require.Nil(s.T(), sp.Signal(server.DefaultServerGracefulSIG))
	time.Sleep(5 * time.Second)
	sp, err = getServerProcessByEmptyCall(testData.network, testData.target)
	require.Nil(s.T(), err)
	require.NotEqual(s.T(), pid, sp.Pid)
	// Kill server and wait for it to graceful exit.
	require.Nil(s.T(), sp.Kill())
	time.Sleep(time.Second)
}

func startServerFromBash(sourceFile, configFile, targetFile string) (*exec.Cmd, error) {
	cmd := exec.Command(
		"bash",
		"-c",
		fmt.Sprintf("go build -o %s %s", targetFile, sourceFile),
	)
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	cmd = exec.Command(targetFile, "-conf", configFile)
	cmd.Stdout = os.Stdout
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	// Wait until server has started.
	time.Sleep(3 * time.Second)
	return cmd, nil
}

func getServerProcessByEmptyCall(network, target string) (*os.Process, error) {
	head := &trpc.ResponseProtocol{}
	ctx, cancel := context.WithTimeout(trpc.BackgroundContext(), 5*time.Second)
	defer cancel()
	if _, err := testpb.NewTestTRPCClientProxy().EmptyCall(
		ctx,
		&testpb.Empty{},
		client.WithNetwork(network),
		client.WithTarget(target),
		client.WithRspHead(head),
	); err != nil {
		return nil, err
	}

	serverPid, err := strconv.Atoi(string(head.TransInfo["server-pid"]))
	if err != nil {
		return nil, err
	}

	sp, err := os.FindProcess(serverPid)
	if err != nil {
		return nil, err
	}

	return sp, nil
}

func echo(req, network, target string) (string, error) {
	ctx, cancel := context.WithTimeout(trpc.BackgroundContext(), 5*time.Second)
	defer cancel()
	rsp, err := testpb.NewTestTRPCClientProxy().UnaryCall(
		ctx,
		&testpb.SimpleRequest{Username: req},
		client.WithNetwork(network),
		client.WithTarget(target),
	)
	if err != nil {
		return "", err
	}
	return rsp.GetUsername(), nil
}
