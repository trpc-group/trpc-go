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
// //
// //
// // Tencent is pleased to support the open source community by making tRPC available.
// //
// // Copyright (C) 2023 THL A29 Limited, a Tencent company.
// // All rights reserved.
// //
// // If you have downloaded a copy of the tRPC source code from Tencent,
// // please note that tRPC source code is licensed under the  Apache 2.0 License,
// // A copy of the Apache 2.0 License is included in this file.
// //
// //

package main

// import (
// 	"context"
// 	"crypto/sha256"
// 	"encoding/hex"
// 	"encoding/json"
// 	"fmt"
// 	"math/rand"
// 	"net/http"
// 	"strings"

// 	_ "git.code.oa.com/trpc-go/trpc-metrics-prometheus"
// 	_ "git.woa.com/trpc-go/trpc-robust"
// 	"git.woa.com/trpc-go/trpc-robust/flags"

// 	"trpc.group/trpc-go/trpc-go"
// 	"trpc.group/trpc-go/trpc-go/admin"
// 	"trpc.group/trpc-go/trpc-go/log"
// 	pb "trpc.group/trpc-go/trpc-go/testdata"
// )

// func init() {
// 	admin.HandleFunc("/cmds/robust", handleTRPCRobust)
// }

// func main() {
// 	s := trpc.NewServer()
// 	pb.RegisterGreeterService(s.Service("trpc.test.helloworld.Greeter"), &serviceImpl{})
// 	if err := s.Serve(); err != nil {
// 		log.Error(err)
// 	}
// }

// type serviceImpl struct{}

// func (si *serviceImpl) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
// 	rsp := &pb.HelloReply{
// 		Msg: "hi, " + req.Msg + " " + simulateHighCPUForString(req.Msg),
// 	}
// 	return rsp, nil
// }

// func (si *serviceImpl) SayHi(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
// 	rsp := &pb.HelloReply{
// 		Msg: "hi, " + req.Msg + " " + simulateHighCPUForString(req.Msg),
// 	}
// 	return rsp, nil
// }

// // simulateHighCPUForString takes a string and applies a SHA-256 hash operation
// // multiple times to simulate high CPU cost.
// func simulateHighCPUForString(input string) string {
// 	const (
// 		base        = 80
// 		fluctuation = 10
// 	)
// 	iterations := base - fluctuation/2 + rand.Int()%fluctuation
// 	for i := 0; i < iterations; i++ {
// 		hasher := sha256.New()
// 		hasher.Write([]byte(input))
// 		input = hex.EncodeToString(hasher.Sum(nil))
// 	}
// 	return input
// }

// func handleTRPCRobust(w http.ResponseWriter, r *http.Request) {
// 	if r.Method != http.MethodGet && r.Method != http.MethodPut {
// 		w.Header().Set("Allow", strings.Join([]string{http.MethodGet, http.MethodPut}, ", "))
// 		w.WriteHeader(http.StatusMethodNotAllowed)
// 		admin.ErrorOutput(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
// 		return
// 	}

// 	w.Header().Set("X-Content-Type-Options", "nosniff")
// 	w.Header().Set("Content-Type", "application/json; charset=utf-8")

// 	if err := r.ParseForm(); err != nil {
// 		admin.ErrorOutput(w, err.Error(), admin.ErrCodeServer)
// 		return
// 	}
// 	cfg := trpc.GlobalConfig().Server.Admin
// 	addr := fmt.Sprintf("%s:%d", cfg.IP, cfg.Port)

// 	message := fmt.Sprintf(`Usage:
// - Disable trpc-robust: curl -X PUT http://%s/cmds/robust?disable=1
// - Enable trpc-robust: curl -X PUT http://%s/cmds/robust?disable=0
// - Get flags (Read Only): curl http://%s/cmds/robust
// Note: If you want to modify disable flags, please use HTTP PUT method.
// `, addr, addr, addr)
// 	ret := map[string]interface{}{
// 		admin.ReturnErrCodeParam: -1,
// 		admin.ReturnMessageParam: message,
// 	}

// 	if disable := r.Form.Get("disable"); disable != "" {
// 		flag, ok := parseBoolParam(disable)
// 		if !ok {
// 			warningAboutParamValuesForResponse(ret)
// 			loadFlagsForResponse(ret)
// 			_ = json.NewEncoder(w).Encode(ret)
// 			return
// 		}

// 		if r.Method == http.MethodPut {
// 			flags.DisableTRPCRobust.Store(flag)
// 			ret[admin.ReturnMessageParam] = fmt.Sprintf("DisableTRPCRobust is turned to %v", flag)
// 		} else {
// 			loadFlagsForResponse(ret)
// 			_ = json.NewEncoder(w).Encode(ret)
// 			return
// 		}
// 	}

// 	ret[admin.ReturnErrCodeParam] = 0
// 	loadFlagsForResponse(ret)
// 	_ = json.NewEncoder(w).Encode(ret)
// }

// func warningAboutParamValuesForResponse(ret map[string]interface{}) {
// 	ret["warning"] = "Invalid parameter values, expected 0, 1, true and false only."
// }

// func loadFlagsForResponse(ret map[string]interface{}) {
// 	const currentDisableTRPCRobust = "currentDisableTRPCRobust"
// 	ret[currentDisableTRPCRobust] = flags.DisableTRPCRobust.Load()
// }

// func parseBoolParam(param string) (value, ok bool) {
// 	switch param {
// 	case "true", "1":
// 		return true, true
// 	case "false", "0":
// 		return false, true
// 	default:
// 		return false, false
// 	}
// }
