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
	"time"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/test/protocols"
)

func (s *TestSuite) TestScopedClient() {
	s.startServer(&TRPCService{
		UnaryCallF: func(ctx context.Context, r *protocols.SimpleRequest) (*protocols.SimpleResponse, error) {
			return &protocols.SimpleResponse{Payload: r.Payload}, nil
		},
	})
	req := &protocols.SimpleRequest{
		Payload: &protocols.Payload{
			Body: []byte(`
Four score and seven years ago our fathers brought forth on this continent, a new nation, 
conceived in Liberty, and dedicated to the proposition that all men are created equal.
Now we are engaged in a great civil war, testing whether that nation, or any nation so conceived and so dedicated, 
can long endure. We are met on a great battle-field of that war. We have come to dedicate a portion of that field, 
as a final resting place for those who here gave their lives that that nation might live. 
It is altogether fitting and proper that we should do this.
But, in a larger sense, we can not dedicate -- we can not consecrate -- we can not hallow -- this ground. 
The brave men, living and dead, who struggled here, have consecrated it, far above our poor power to add or detract. 
The world will little note, nor long remember what we say here, but it can never forget what they did here. 
It is for us the living, rather, to be dedicated here to the unfinished work which they who fought here have thus far 
so nobly advanced. It is rather for us to be here dedicated to the great task remaining before us -- that from these 
honored dead we take increased devotion to that cause for which they gave the last full measure of devotion -- that 
we here highly resolve that these dead shall not have died in vain -- that this nation, under God, shall have a new 
birth of freedom -- and that government of the people, by the people, for the people, shall not perish from the earth.
`),
		},
	}
	proxy := s.newTRPCClient(client.WithServiceName(protocols.TestTRPCServer_ServiceDesc.ServiceName))
	ctx := trpc.BackgroundContext()
	start := time.Now()
	tot := 10000
	for i := 0; i < 10000; i++ {
		_, err := proxy.UnaryCall(ctx, req, client.WithScope("local"))
		if err != nil {
			s.T().Error(err)
		}
	}
	elapsedLocal := time.Since(start)
	start = time.Now()
	for i := 0; i < 10000; i++ {
		_, err := proxy.UnaryCall(ctx, req, client.WithScope("remote"))
		if err != nil {
			s.T().Error(err)
		}
	}
	elapsedRemote := time.Since(start)
	log.Infof("local scope QPS: %d, average cost: %.2fms", int(float64(tot)/elapsedLocal.Seconds()),
		1000*elapsedLocal.Seconds()/float64(tot))
	log.Infof("remote scope QPS: %d, average cost: %.2fms", int(float64(tot)/elapsedRemote.Seconds()),
		1000*elapsedRemote.Seconds()/float64(tot))
	if elapsedLocal > elapsedRemote {
		s.T().Errorf("local elapsed %v is larger than remote elapsed %v", elapsedLocal, elapsedRemote)
	}
}
