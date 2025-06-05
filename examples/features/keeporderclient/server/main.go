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
	"strconv"
	"strings"
	"sync"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/examples/features/keeporderclient/proto"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/server"
)

const (
	flagPreDecode    = "pre-decode"
	flagPreUnmarshal = "pre-unmarshal"
	flagNone         = "none"
)

func main() {
	s := trpc.NewServer(server.WithServerAsync(false))
	proto.RegisterPlayerService(s, &serviceImpl{ids: make(map[string][]string)})
	if err := s.Serve(); err != nil {
		log.Fatal(err)
	}
}

type serviceImpl struct {
	mu  sync.Mutex
	ids map[string][]string
}

func (si *serviceImpl) Update(ctx context.Context, req *proto.UpdateReq) (*proto.UpdateRsp, error) {
	// Sleep certain amount of time that is inverse proportional to the couter received
	// to amplify what keep-order wants to achieve.
	// time.Sleep(20 * time.Millisecond * time.Duration(req.GetTotal()-req.GetCounter()))
	log.Infof("start process update request %+v", req)
	si.mu.Lock()
	defer si.mu.Unlock()
	ids := si.ids[req.GetId()]
	ids = append(ids, strconv.Itoa(int(req.GetCounter())))
	si.ids[req.GetId()] = ids
	rsp := &proto.UpdateRsp{
		State: strings.Join(ids, " "),
	}
	if len(ids) == int(req.GetTotal()) {
		// Clear the key when full.
		delete(si.ids, req.GetId())
	}
	return rsp, nil
}
