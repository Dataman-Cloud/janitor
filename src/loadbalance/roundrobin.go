package loadbalance

import (
	"github.com/Dataman-Cloud/janitor/src/upstream"
	"sync"
)

type LoadBalancer interface {
	Next() *upstream.Target
	Seed(upstream *upstream.Upstream)
}

type RoundRobinLoadBalancer struct {
	Upstream  *upstream.Upstream
	NextIndex int
	SeedLock  sync.Mutex
}

func NewRoundRobinLoadBalancer() *RoundRobinLoadBalancer {
	return &RoundRobinLoadBalancer{}
}

func (rr *RoundRobinLoadBalancer) Seed(upstream *upstream.Upstream) {
	rr.SeedLock.Lock()
	defer rr.SeedLock.Unlock()
	rr.Upstream = upstream
	rr.NextIndex = 0
}

func (rr *RoundRobinLoadBalancer) Next() *upstream.Target {
	current := rr.Upstream.Targets[rr.NextIndex]
	rr.NextIndex = (rr.NextIndex + 1) % len(rr.Upstream.Targets)
	return current
}
