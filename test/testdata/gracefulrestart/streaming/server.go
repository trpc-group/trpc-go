// Package main is the main package.
package main

import (
	"io"
	"os"
	"strconv"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/test"
	testpb "trpc.group/trpc-go/trpc-go/test/protocols"
)

func main() {
	svr := trpc.NewServer()
	testpb.RegisterTestStreamingService(
		svr,
		&test.StreamingService{FullDuplexCallF: func(stream testpb.TestStreaming_FullDuplexCallServer) error {
			for {
				in, err := stream.Recv()
				if err == io.EOF {
					return nil
				}
				if err != nil {
					return err
				}
				for range in.GetResponseParameters() {
					if err := stream.Send(&testpb.StreamingOutputCallResponse{
						Payload: &testpb.Payload{
							Type: testpb.PayloadType_COMPRESSIBLE,
							Body: []byte(strconv.Itoa(os.Getpid())),
						},
					}); err != nil {
						return err
					}
				}
			}
		}},
	)
	if err := svr.Serve(); err != nil {
		panic(err)
	}
}
