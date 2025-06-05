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
	"encoding/json"
	"flag"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/admin"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/filter"
	"trpc.group/trpc-go/trpc-go/log"
	pb "trpc.group/trpc-go/trpc-go/testdata"
	_ "git.code.oa.com/trpc-go/trpc-metrics-prometheus"
	rcodec "git.code.oa.com/trpc-go/trpc-utils/robust/codec"
	"golang.org/x/time/rate"
)

// /                  peak QPS    +-----------+
// /                             /|           |\
// /                            / |           | \
// /                           /  |           |  \
// / initial QPS +------------+   |           |   +---------------+
// /             |            |   |           |   |               |
// /            initial duration  keep duration   die down duration
// /                          |   |           |   |
// /                   change duration     change duration
var (
	initialQPS          = 1000
	peakQPS             = 20000
	changeRate          int
	initialDuration     = 10 * time.Second
	keepDuration        = 10 * time.Second
	changeDuration      = 30 * time.Second
	diedownDuration     = 10 * time.Second
	changeInterval      = time.Second
	clientNumber        = 1000
	rateLimiter         *rate.Limiter
	rateLimiterInTuning atomic.Bool
)

func init() {
	admin.HandleFunc("/cmds/restart", handleRestartTune)
}

func setupFlags() {
	flag.IntVar(&initialQPS, "i", initialQPS, "set initial QPS")
	flag.IntVar(&peakQPS, "p", peakQPS, "set peak QPS")
	flag.DurationVar(&initialDuration, "id", initialDuration, "set initial duration")
	flag.DurationVar(&keepDuration, "kd", keepDuration, "set keep duration")
	flag.DurationVar(&changeDuration, "cd", changeDuration, "set change duration")
	flag.DurationVar(&diedownDuration, "dd", diedownDuration, "set diedown duration")
	flag.DurationVar(&changeInterval, "ci", changeInterval, "set change interval")
	flag.IntVar(&clientNumber, "cn", clientNumber, "set number of clients")
	flag.Parse()
	changeRate = (peakQPS - initialQPS) / (int(changeDuration) / int(changeInterval))
}

func main() {
	setupFlags()
	s := trpc.NewServer()
	ch := make(chan error)
	go func() { ch <- s.Serve() }()
	rateLimiter = rate.NewLimiter(rate.Limit(initialQPS), 10)
	go tuneRatePeriodically(rateLimiter)
	runClients(rateLimiter)
	if err := <-ch; err != nil {
		log.Errorf("serve err: %+v", err)
	}

}

func runClients(r *rate.Limiter) {
	const maxUserPriority = 255
	sceneIDs := []string{"sceneA", "sceneB", "sceneC", "sceneD", "sceneE", "sceneF"}
	sceneCount := len(sceneIDs)
	for i := 0; i < clientNumber; i++ {
		userPriority := i % maxUserPriority
		scenePriority := 0
		sceneID := sceneIDs[i%sceneCount]
		isVIP := false
		go func() {
			// Create a new client proxy for each client.
			// Set different priority information for each client.
			proxy := pb.NewGreeterClientProxy(
				client.WithNamedFilter("set priority",
					func(ctx context.Context, req, rsp interface{}, next filter.ClientHandleFunc) error {
						msg := trpc.Message(ctx)
						rcodec.WithClientRequestPriority(msg, uint16(userPriority), uint8(scenePriority), isVIP)
						rcodec.WithClientRequestSceneID(msg, sceneID)
						return next(ctx, req, rsp)
					}))
			ctx := trpc.BackgroundContext()
			for {
				if err := r.Wait(ctx); err != nil {
					log.Errorf("rate limit wait error: %v", err)
					continue
				}
				_, err := proxy.SayHello(ctx, &pb.HelloRequest{Msg: "trpc-go"})
				if err != nil {
					log.Errorf("say hello error: %v", err)
					continue
				}
			}
		}()
	}
}

func tuneRatePeriodically(r *rate.Limiter) {
	rateLimiterInTuning.Store(true)
	defer func() { rateLimiterInTuning.Store(false) }()
	r.SetLimit(rate.Limit(initialQPS))
	log.Infof("current rate: %v", r.Limit())
	log.Infof("start initial duration")
	time.Sleep(initialDuration)
	log.Infof("initial duration passed")

	tuneIncrease(r)

	log.Infof("start keep duration")
	time.Sleep(keepDuration)
	log.Infof("keep duration passed")

	tuneDecrease(r)

	log.Infof("start die down duration")
	time.Sleep(diedownDuration)
	log.Infof("die down duration passed")
}

func tuneIncrease(r *rate.Limiter) {
	log.Infof("start increase duration")
	defer func() {
		log.Infof("increase duration passed")
	}()
	ticker := time.NewTicker(changeInterval)
	defer ticker.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), changeDuration)
	defer cancel()
	var cnt int
	for {
		select {
		case <-ticker.C:
			cnt++
			r.SetLimit(rate.Limit(initialQPS + cnt*changeRate))
			log.Infof("current rate: %v", r.Limit())
		case <-ctx.Done():
			return
		}
	}
}

func tuneDecrease(r *rate.Limiter) {
	log.Infof("start decrease duration")
	defer func() {
		log.Infof("decrease duration passed")
	}()
	ticker := time.NewTicker(changeInterval)
	defer ticker.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), changeDuration)
	defer cancel()
	var cnt int
	for {
		select {
		case <-ticker.C:
			cnt++
			r.SetLimit(rate.Limit(peakQPS - cnt*changeRate))
			log.Infof("current rate: %v", r.Limit())
		case <-ctx.Done():
			return
		}
	}
}

func handleRestartTune(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", strings.Join([]string{http.MethodGet}, ", "))
		w.WriteHeader(http.StatusMethodNotAllowed)
		admin.ErrorOutput(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	ret := map[string]interface{}{
		admin.ReturnErrCodeParam: -1,
		admin.ReturnMessageParam: "rate limiter is not ready",
	}
	if rateLimiter != nil {
		if rateLimiterInTuning.Load() {
			ret = map[string]interface{}{
				admin.ReturnErrCodeParam: -1,
				admin.ReturnMessageParam: "rate limiter is in tuning, please try later",
			}
		} else {
			go tuneRatePeriodically(rateLimiter)
			ret = map[string]interface{}{
				admin.ReturnErrCodeParam: 0,
				admin.ReturnMessageParam: "rate tuning restarted",
			}
		}
	}
	_ = json.NewEncoder(w).Encode(ret)
}
