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

package test

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"github.com/stretchr/testify/require"
	trpcpb "trpc.group/trpc/trpc-protocol/pb/go/trpc"

	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/errs"
	"trpc.group/trpc-go/trpc-go/server"
	testpb "trpc.group/trpc-go/trpc-go/test/protocols"
)

func (s *TestSuite) TestServerGracefulRestart() {
	s.Run("ServerGracefulRestartIsIdempotent", func() {
		s.testServerGracefulRestartIsIdempotent()
	})
	s.Run("OldStreamFailedButNewStreamOk", func() {
		s.testServerGracefulRestartOldStreamFailedButNewStreamOk()
	})
	s.Run("SendNonGracefulRestartSignal", func() {
		s.testSendNonGracefulRestartSignal()
	})
	s.Run("GracefulRestartForEmptyIP", func() {
		s.testGracefulRestartForEmptyIP()
	})
}

func (s *TestSuite) testServerGracefulRestartIsIdempotent() {
	const (
		binaryFile = "./gracefulrestart/trpc/server.o"
		sourceFile = "./gracefulrestart/trpc/server.go"
		configFile = "./gracefulrestart/trpc/trpc_go.yaml"
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
	}()

	const target = "ip://127.0.0.1:17777"
	sp, err := getServerProcessByEmptyCall(target)
	require.Nil(s.T(), err)
	pid := sp.Pid
	for i := 0; i < 3; i++ {
		require.Nil(s.T(), sp.Signal(server.DefaultServerGracefulSIG))
		// wait until server has restarted gracefully.
		time.Sleep(1 * time.Second)
		sp, err = getServerProcessByEmptyCall(target)
		require.Nil(s.T(), err)
		require.NotEqual(s.T(), pid, sp.Pid)
		pid = sp.Pid
	}
	require.Nil(s.T(), sp.Kill())
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
	}()

	respParams := []*testpb.ResponseParameters{
		{
			Size: int32(1),
		},
	}
	payload, err := newPayload(testpb.PayloadType_COMPRESSIBLE, int32(1))
	require.Nil(s.T(), err)
	req := &testpb.StreamingOutputCallRequest{
		ResponseType:       testpb.PayloadType_COMPRESSIBLE,
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
	// wait until server has restarted gracefully.
	time.Sleep(1 * time.Second)

	err = cs1.Send(req)
	require.Equal(s.T(), errs.RetServerSystemErr, errs.Code(err))
	require.Contains(s.T(), errs.Msg(err), "Connection is Closed")

	sp2, cs2 := doFullDuplexCall()
	require.Nil(s.T(), cs2.Send(req))

	require.NotEqual(s.T(), pid1, sp2.Pid)
	require.Nil(s.T(), sp2.Kill())
}

func (s *TestSuite) TestServerGracefulRestartOldListenerIsClosed() {
	const binaryFile = "./gracefulrestart/trpc/server.o"
	cmd := exec.Command(
		"bash",
		"-c",
		fmt.Sprintf("go build -o %s ./gracefulrestart/trpc/server.go", binaryFile),
	)
	require.Nil(s.T(), cmd.Run())
	defer func() {
		cmd := exec.Command("rm", binaryFile)
		require.Nil(s.T(), cmd.Run())
	}()

	cmd = exec.Command(binaryFile, "-conf", "./gracefulrestart/trpc/trpc_go.yaml")
	cmd.Stdout = os.Stdout
	require.Nil(s.T(), cmd.Start())
	// wait until server has started.
	time.Sleep(3 * time.Second)
	defer func() {
		require.Nil(s.T(), cmd.Process.Kill())
	}()

	c := testpb.NewTestTRPCClientProxy()
	doEmptyCall := func() *os.Process {
		head := &trpcpb.ResponseProtocol{}
		_, err := c.EmptyCall(
			trpc.BackgroundContext(),
			&testpb.Empty{},
			client.WithTarget("ip://127.0.0.1:17777"),
			client.WithRspHead(head),
		)
		require.Nil(s.T(), err)

		serverPid, err := strconv.Atoi(string(head.TransInfo["server-pid"]))
		require.Nil(s.T(), err)
		sp, err := os.FindProcess(serverPid)
		require.Nil(s.T(), err)
		return sp
	}

	sp := doEmptyCall()
	pid := sp.Pid
	require.Nil(s.T(), sp.Signal(server.DefaultServerGracefulSIG))
	time.Sleep(600 * time.Millisecond)
	for i := 0; i < 30; i++ {
		sp = doEmptyCall()
		require.NotEqual(s.T(), pid, sp.Pid) // The old listener is closed, all request is sent to the new one.
	}
	require.Nil(s.T(), sp.Kill())
}

func (s *TestSuite) testSendNonGracefulRestartSignal() {
	const (
		sourceFile = "./gracefulrestart/trpc/server.go"
		configFile = "./gracefulrestart/trpc/trpc_go.yaml"
		binaryFile = "./gracefulrestart/trpc/server.o"

		target = "ip://127.0.0.1:17777"
	)

	s.Run("Send Default Server Close Signal", func() {
		cmd, err := startServerFromBash(
			sourceFile,
			configFile,
			binaryFile,
		)
		require.Nil(s.T(), err)
		defer func() {
			require.Nil(s.T(), exec.Command("rm", binaryFile).Run())
			require.Nil(s.T(), cmd.Process.Kill())
		}()
		sp, err := getServerProcessByEmptyCall(target)
		require.Nil(s.T(), err)

		r := rand.New(rand.NewSource(time.Now().Unix()))
		closeSignal := server.DefaultServerCloseSIG[r.Intn(len(server.DefaultServerCloseSIG))]
		require.Nil(s.T(), sp.Signal(closeSignal))
		for {
			if _, err := getServerProcessByEmptyCall(target); err != nil {
				require.EqualValues(s.T(), errs.RetClientReadFrameErr, errs.Code(err))
				return
			}
		}
	})
	s.Run("Send Non Close Signal", func() {
		cmd, err := startServerFromBash(
			sourceFile,
			configFile,
			binaryFile,
		)
		require.Nil(s.T(), err)
		defer func() {
			require.Nil(s.T(), exec.Command("rm", binaryFile).Run())
			require.Nil(s.T(), cmd.Process.Kill())
		}()

		sp, err := getServerProcessByEmptyCall(target)
		require.Nil(s.T(), err)
		pid := sp.Pid
		for i := 0; i < 3; i++ {
			require.Nil(s.T(), sp.Signal(syscall.SIGUSR1))
			sp, err = getServerProcessByEmptyCall(target)
			require.Equal(s.T(), pid, sp.Pid)
		}
		require.Nil(s.T(), sp.Kill())
	})
}

func (s *TestSuite) testGracefulRestartForEmptyIP() {
	const (
		binaryFile = "./gracefulrestart/trpc/server.o"
		sourceFile = "./gracefulrestart/trpc/server.go"
		configFile = "./gracefulrestart/trpc/trpc_go_emptyip.yaml"
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
	}()

	const target = "ip://127.0.0.1:17777"
	sp, err := getServerProcessByEmptyCall(target)
	require.Nil(s.T(), err)
	pid := sp.Pid
	require.Nil(s.T(), sp.Signal(server.DefaultServerGracefulSIG))
	time.Sleep(1 * time.Second)
	sp, err = getServerProcessByEmptyCall(target)
	require.Nil(s.T(), err)
	require.NotEqual(s.T(), pid, sp.Pid)
	pid = sp.Pid
	require.Nil(s.T(), sp.Kill())
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
	// wait until server has started.
	time.Sleep(3 * time.Second)
	return cmd, nil
}

func getServerProcessByEmptyCall(target string) (*os.Process, error) {
	head := &trpcpb.ResponseProtocol{}
	if _, err := testpb.NewTestTRPCClientProxy().EmptyCall(
		trpc.BackgroundContext(),
		&testpb.Empty{},
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
