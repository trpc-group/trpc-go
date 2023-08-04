// Package weightroundrobin provides weight round robin utilities.
package weightroundrobin

import (
	"sync"
	"time"

	"trpc.group/trpc-go/trpc-go/naming/loadbalance"
	"trpc.group/trpc-go/trpc-go/naming/registry"
)

var defaultUpdateRate time.Duration = time.Second * 10

func init() {
	loadbalance.Register("weight_round_robin", NewWeightRoundRobin(defaultUpdateRate))
}

// NewWeightRoundRobin creates a new WeightRoundRobin.
func NewWeightRoundRobin(interval time.Duration) *WeightRoundRobin {
	if interval == 0 {
		interval = defaultUpdateRate
	}
	return &WeightRoundRobin{
		pickers:  new(sync.Map),
		interval: interval,
	}
}

// WeightRoundRobin is a smooth weighted roundrobin algorithm.
type WeightRoundRobin struct {
	pickers  *sync.Map
	interval time.Duration
}

// Select implements loadbalance.LoadBalancer.
func (wrr *WeightRoundRobin) Select(serviceName string, list []*registry.Node,
	opt ...loadbalance.Option) (*registry.Node, error) {
	opts := &loadbalance.Options{}
	for _, o := range opt {
		o(opts)
	}
	p, ok := wrr.pickers.Load(serviceName)
	if ok {
		return p.(*wrrPicker).Pick(list, opts)
	}

	newPicker := &wrrPicker{
		interval: wrr.interval,
	}
	v, ok := wrr.pickers.LoadOrStore(serviceName, newPicker)
	if !ok {
		return newPicker.Pick(list, opts)
	}
	return v.(*wrrPicker).Pick(list, opts)
}

// wrrPicker is a picker based on weighted roundrobin algorithm.
type wrrPicker struct {
	list     []*Server
	updated  time.Time
	mu       sync.Mutex
	interval time.Duration
}

// Server records the node status.
type Server struct {
	node          *registry.Node
	weight        int
	currentWeight int
	effectWeight  int
}

// Pick picks a node.
func (p *wrrPicker) Pick(list []*registry.Node, opts *loadbalance.Options) (*registry.Node, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.updateState(list)
	if len(p.list) == 0 {
		return nil, loadbalance.ErrNoServerAvailable
	}
	selected := p.selectServer()
	return selected.node, nil
}

func (p *wrrPicker) selectServer() *Server {
	var selected *Server
	var total int
	for _, s := range p.list {
		s.currentWeight += s.effectWeight
		total += s.effectWeight
		if s.effectWeight < s.weight {
			s.effectWeight++
		}
		if selected == nil || s.currentWeight > selected.currentWeight {
			selected = s
		}
	}
	selected.currentWeight -= total
	return selected
}

func (p *wrrPicker) updateState(list []*registry.Node) {
	if len(p.list) == 0 ||
		len(p.list) != len(list) ||
		time.Since(p.updated) > p.interval {
		p.list = p.getServers(list)
		p.updated = time.Now()
	}
}

func (p *wrrPicker) getServers(list []*registry.Node) []*Server {
	servers := make([]*Server, 0, len(list))
	for _, n := range list {
		weight := n.Weight
		if weight == 0 {
			weight = 1000
		}
		servers = append(servers, &Server{
			node:          n,
			weight:        weight,
			effectWeight:  weight,
			currentWeight: 0,
		})
	}
	return servers
}
