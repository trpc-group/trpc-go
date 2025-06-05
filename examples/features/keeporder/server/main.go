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
	"flag"
	"strconv"
	"strings"
	"sync"
	"time"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/examples/features/keeporder/meta"
	"trpc.group/trpc-go/trpc-go/examples/features/keeporder/proto"
	"trpc.group/trpc-go/trpc-go/log"
	"trpc.group/trpc-go/trpc-go/server"
)

const (
	flagPreDecode    = "pre-decode"
	flagPreUnmarshal = "pre-unmarshal"
	flagNone         = "none"
)

func main() {
	var flagKeepOrder string
	flag.StringVar(&trpc.ServerConfigPath, "conf", "./trpc_go.yaml", "server config path")
	flag.StringVar(&flagKeepOrder, "keep-order", "pre-decode", "mode of keep-order feature, default `pre-decode`, "+
		"other option: `pre-unmarshal`, `none`")
	flag.Parse()
	var opts []server.Option
	switch flagKeepOrder {
	case flagPreDecode:
		log.Infof("keep-order mode is pre-decode")
		opts = append(opts, server.WithKeepOrderPreDecodeExtractor(func(ctx context.Context, reqBody []byte) (string, bool) {
			// Implement keep-order logic for pre-decoding.
			msg := codec.Message(ctx)
			m := msg.ServerMetaData()
			if m == nil {
				log.Errorf("meta data is nil for %q\n", reqBody)
				return "", false
			}
			key, ok := m[meta.KeepOrderKey]
			if !ok {
				log.Errorf("meta key %q does not exist for %q\n", meta.KeepOrderKey, reqBody)
				return "", false
			}
			return string(key), true
		}))
	case flagPreUnmarshal:
		log.Infof("keep-order mode is pre-unmarshal")
		opts = append(opts, server.WithKeepOrderPreUnmarshalExtractor(func(ctx context.Context, req interface{}) (string, bool) {
			// Implement keep-order logic for pre-unmarshaling.
			request, ok := req.(*proto.UpdateReq)
			if !ok {
				log.Errorf("invalid request type %T, want *proto.HelloReq", req)
				return "", false
			}
			return request.GetId(), true
		}))
	case flagNone:
		// Keep-order feature is disabled.
		// No-op.
		log.Infof("keep-order mode is none (disabled)")
	default:
		log.Fatalf("unsupported flag type %T", flagKeepOrder)
	}
	s := trpc.NewServer(opts...)
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
	// Sleep certain amount of time that is inverse proportional to the counter received
	// to amplify what keep-order wants to achieve.
	time.Sleep(20 * time.Millisecond * time.Duration(req.GetTotal()-req.GetCounter()))
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
