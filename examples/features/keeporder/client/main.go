package main

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	"trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/client"
	"trpc.group/trpc-go/trpc-go/examples/features/keeporder/meta"
	"trpc.group/trpc-go/trpc-go/examples/features/keeporder/proto"
	"trpc.group/trpc-go/trpc-go/log"
)

func main() {
	// Load and setup client configuration.
	trpc.LoadGlobalConfig(trpc.ServerConfigPath)
	trpc.SetupClients(&trpc.GlobalConfig().Client)
	keys := []string{"key1", "key2", "key3", "key4", "key5"}
	count := 10
	var eg errgroup.Group
	var mu sync.Mutex
	rsps := make(map[string]string)
	for _, key := range keys {
		key := key
		proxy := proto.NewPlayerClientProxy(
			client.WithMetaData(
				meta.KeepOrderKey, []byte(key), // Only needed when the server is using `pre-decode` mode.
			))
		for i := 1; i <= count; i++ {
			i := i
			eg.Go(func() error {
				// Sleep a certain amount of time that is proportional to the counter
				// to let the smaller counter reach the server first.
				// This is not very accurate, but it is the best that we can do.
				time.Sleep(time.Millisecond * time.Duration(i*20))
				ctx, cancel := context.WithTimeout(trpc.BackgroundContext(), time.Second)
				defer cancel()
				req := &proto.UpdateReq{
					Id:      key,
					Counter: int32(i),
					Total:   int32(count),
				}
				rsp, err := proxy.Update(ctx, req)
				if err != nil {
					log.Fatalf("client request failed: %+v", err)
				}
				// Only store the final result.
				mu.Lock()
				if len(rsps[key]) < len(rsp.State) {
					rsps[key] = rsp.State
				}
				mu.Unlock()
				return err
			})
		}
	}
	if err := eg.Wait(); err != nil {
		log.Fatalf("client request failed: %+v", err)
	}
	expectSlice := make([]string, 0, count)
	for i := 1; i <= count; i++ {
		expectSlice = append(expectSlice, strconv.Itoa(i))
	}
	expect := strings.Join(expectSlice, " ")
	for _, key := range keys {
		if rsps[key] != expect {
			log.Errorf("[FAIL] key %s: expect %s, but got %s", key, expect, rsps[key])
		} else {
			log.Infof("[SUCCESS] key %s: expect %s, got %s", key, expect, rsps[key])
		}
	}
}
