// Package main is the main package.
package main

import (
	"context"
	"os"
	"strconv"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/test"
	testpb "trpc.group/trpc-go/trpc-go/test/protocols"
)

func main() {
	svr := trpc.NewServer()
	testpb.RegisterTestTRPCService(
		svr,
		&test.TRPCService{EmptyCallF: func(ctx context.Context, in *testpb.Empty) (*testpb.Empty, error) {
			trpc.SetMetaData(ctx, "server-pid", []byte(strconv.Itoa(os.Getpid())))
			return &testpb.Empty{}, nil
		}},
	)
	if err := svr.Serve(); err != nil {
		panic(err)
	}
}
