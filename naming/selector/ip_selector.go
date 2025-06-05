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

package selector

import (
	"errors"
	"math/rand"
	"strings"
	"time"

	"trpc.group/trpc-go/trpc-go/internal/random"
	"trpc.group/trpc-go/trpc-go/naming/bannednodes"
	"trpc.group/trpc-go/trpc-go/naming/discovery"
	"trpc.group/trpc-go/trpc-go/naming/loadbalance"
	"trpc.group/trpc-go/trpc-go/naming/registry"
	"trpc.group/trpc-go/trpc-go/naming/servicerouter"
)

func init() {
	Register("ip", NewIPSelector())  // ip://ip:port
	Register("dns", NewIPSelector()) // dns://domain:port
}

// ipSelector is a selector based on ip list.
type ipSelector struct {
	vanilla  bool
	safeRand *rand.Rand
	cb       circuitBreaker
}

// NewIPSelector creates a new ipSelector.
func NewIPSelector() *ipSelector {
	return &ipSelector{
		vanilla:  true,
		safeRand: random.New(),
		cb:       noopCircuitBreak{},
	}
}

// NewIPSelectorWithCircuitBreaker creates a new ipSelector with a circuitBreaker.
func NewIPSelectorWithCircuitBreaker(cb circuitBreaker) *ipSelector {
	return &ipSelector{
		safeRand: random.New(),
		cb:       cb,
	}
}

// Select implements Selector.Select. ServiceName may have multiple IP, such as ip1:port1,ip2:port2.
// If ctx has bannedNodes, Select will try its best to select a node not in bannedNodes.
// If no node is available due to circuit breaker, a random circuit broken node is returned.
func (s *ipSelector) Select(
	serviceName string, opts ...Option,
) (*registry.Node, error) {
	if serviceName == "" {
		return nil, errors.New("ip selector err: serviceName is empty")
	}
	if s.vanilla && strings.IndexByte(serviceName, ',') == -1 {
		return &registry.Node{ServiceName: serviceName, Address: serviceName}, nil
	}
	addr, err := s.selectOne(serviceName, opts...)
	var cirErr *circuitBrokenErr
	if errors.As(err, &cirErr) {
		ss := *s
		ss.cb = noopCircuitBreak{}
		addr, err = ss.chooseOneSlow(cirErr.circuitBrokenAddrs)
	}

	if err != nil {
		return nil, err
	}
	return &registry.Node{ServiceName: serviceName, Address: addr}, nil
}

func (s *ipSelector) selectOne(
	serviceName string, opt ...Option,
) (addr string, err error) {
	o := Options{
		DiscoveryOptions:     make([]discovery.Option, 0, defaultDiscoveryOptionsSize),
		ServiceRouterOptions: make([]servicerouter.Option, 0, defaultServiceRouterOptionsSize),
		LoadBalanceOptions:   make([]loadbalance.Option, 0, defaultLoadBalanceOptionsSize),
	}
	for _, opt := range opt {
		opt(&o)
	}
	if o.Ctx == nil {
		return s.chooseOne(serviceName)
	}

	bans, mandatory, ok := bannednodes.FromCtx(o.Ctx)
	if !ok {
		return s.chooseOne(serviceName)
	}

	defer func() {
		if err == nil {
			bannednodes.Add(o.Ctx, &registry.Node{ServiceName: serviceName, Address: addr})
		}
	}()

	addr, err = s.chooseUnbanned(strings.Split(serviceName, ","), bans)
	if !mandatory && err != nil {
		addr, err = s.chooseOne(serviceName)
	}
	return addr, err
}

func (s *ipSelector) chooseOne(serviceName string) (string, error) {
	num := strings.Count(serviceName, ",") + 1
	if num == 1 {
		if !s.cb.Available(serviceName) {
			return s.chooseOneSlow([]string{serviceName})
		}
		return serviceName, nil
	}

	var addr string
	remaining := serviceName
	r := s.safeRand.Intn(num)
	for i := 0; i <= r; i++ {
		j := strings.IndexByte(remaining, ',')
		if j < 0 {
			addr = remaining
			break
		}
		addr, remaining = remaining[:j], remaining[j+1:]
	}

	if !s.cb.Available(addr) {
		return s.chooseOneSlow(strings.Split(serviceName, ","))
	}
	return addr, nil
}

func (s *ipSelector) chooseOneSlow(addrs []string) (string, error) {
	s.safeRand.Shuffle(len(addrs), func(i, j int) {
		addrs[i], addrs[j] = addrs[j], addrs[i]
	})

	for _, addr := range addrs {
		if s.cb.Available(addr) {
			return addr, nil
		}
	}

	err := errors.New("no available targets")
	for _, addr := range addrs {
		err = wrapCircuitBrokenIn(err, addr)
	}
	return "", err
}

// chooseUnbanned function may have an issue:
// once it finds the first non-banned node, it no longer tracks the bans list for filtering.
func (s *ipSelector) chooseUnbanned(addrs []string, bans *bannednodes.Nodes) (string, error) {
	if len(addrs) == 0 {
		return "", errors.New("no available targets")
	}

	r := s.safeRand.Intn(len(addrs))
	if !bans.Range(func(n *registry.Node) bool {
		return n.Address != addrs[r]
	}) {
		return s.chooseUnbanned(append(addrs[:r], addrs[r+1:]...), bans)
	}

	if !s.cb.Available(addrs[r]) {
		return s.chooseOneSlow(append(addrs[:r], addrs[r+1:]...))
	}
	return addrs[r], nil
}

// Report reports n.Address and whether the err is nil.
func (s *ipSelector) Report(n *registry.Node, _ time.Duration, err error) error {
	s.cb.Report(n.Address, err == nil)
	return nil
}

type circuitBreaker interface {
	Available(addr string) bool
	Report(addr string, ok bool)
}

type noopCircuitBreak struct{}

// Available always return true.
func (noopCircuitBreak) Available(addr string) bool { return true }

// Report reports nothing.
func (noopCircuitBreak) Report(addr string, ok bool) {}

type circuitBrokenErr struct {
	err                error
	circuitBrokenAddrs []string
}

// Error returns the errMsg for circuitBrokenErr.
func (e *circuitBrokenErr) Error() string {
	return e.err.Error() +
		", the following addresses are circuit broken: " +
		strings.Join(e.circuitBrokenAddrs, " ")
}

// Unwrap unwraps the err for circuitBrokenErr.
func (e *circuitBrokenErr) Unwrap() error {
	return e.err
}

func wrapCircuitBrokenIn(e error, addr string) error {
	if e == nil {
		return nil
	}

	var err *circuitBrokenErr
	if errors.As(e, &err) {
		err.circuitBrokenAddrs = append(err.circuitBrokenAddrs, addr)
		return e
	}

	return &circuitBrokenErr{
		err:                e,
		circuitBrokenAddrs: []string{addr},
	}
}
