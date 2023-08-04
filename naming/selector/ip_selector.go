package selector

import (
	"errors"
	"strings"
	"time"

	"trpc.group/trpc-go/trpc-go/internal/rand"
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
	safeRand *rand.SafeRand
}

// NewIPSelector creates a new ipSelector.
func NewIPSelector() *ipSelector {
	return &ipSelector{
		safeRand: rand.NewSafeRand(time.Now().UnixNano()),
	}
}

// Select implements Selector.Select. ServiceName may have multiple IP, such as ip1:port1,ip2:port2.
// If ctx has bannedNodes, Select will try its best to select a node not in bannedNodes.
func (s *ipSelector) Select(
	serviceName string, opt ...Option,
) (node *registry.Node, err error) {
	if serviceName == "" {
		return nil, errors.New("serviceName empty")
	}

	var o Options = Options{
		DiscoveryOptions:     make([]discovery.Option, 0, defaultDiscoveryOptionsSize),
		ServiceRouterOptions: make([]servicerouter.Option, 0, defaultServiceRouterOptionsSize),
		LoadBalanceOptions:   make([]loadbalance.Option, 0, defaultLoadBalanceOptionsSize),
	}
	for _, opt := range opt {
		opt(&o)
	}
	if o.Ctx == nil {
		addr, err := s.chooseOne(serviceName)
		if err != nil {
			return nil, err
		}
		return &registry.Node{ServiceName: serviceName, Address: addr}, nil
	}

	bans, mandatory, ok := bannednodes.FromCtx(o.Ctx)
	if !ok {
		addr, err := s.chooseOne(serviceName)
		if err != nil {
			return nil, err
		}
		return &registry.Node{ServiceName: serviceName, Address: addr}, nil
	}

	defer func() {
		if err == nil {
			bannednodes.Add(o.Ctx, node)
		}
	}()

	addr, err := s.chooseUnbanned(strings.Split(serviceName, ","), bans)
	if !mandatory && err != nil {
		addr, err = s.chooseOne(serviceName)
	}
	if err != nil {
		return nil, err
	}
	return &registry.Node{ServiceName: serviceName, Address: addr}, nil
}

func (s *ipSelector) chooseOne(serviceName string) (string, error) {
	num := strings.Count(serviceName, ",") + 1
	if num == 1 {
		return serviceName, nil
	}

	var addr string
	r := s.safeRand.Intn(num)
	for i := 0; i <= r; i++ {
		j := strings.IndexByte(serviceName, ',')
		if j < 0 {
			addr = serviceName
			break
		}
		addr, serviceName = serviceName[:j], serviceName[j+1:]
	}
	return addr, nil
}

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
	return addrs[r], nil
}

// Report reports nothing.
func (s *ipSelector) Report(*registry.Node, time.Duration, error) error {
	return nil
}
