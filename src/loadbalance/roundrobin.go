package loadbalance

import (
	"github.com/Dataman-Cloud/janitor/src/upstream"
)

type LoadBalancer interface {
	Next() *upstream.Target
	Seed(upstream *upstream.Upstream)
}

type RoundRobinLoaderBalancer struct {
	Upstream  *upstream.Upstream
	NextIndex int
}

func NewRoundRobinLoaderBalancer() *RoundRobinLoaderBalancer {
	return &RoundRobinLoaderBalancer{}
}

func (rr *RoundRobinLoaderBalancer) Seed(upstream *upstream.Upstream) {
	rr.Upstream = upstream
	rr.NextIndex = 0
}

func (rr *RoundRobinLoaderBalancer) Next() *upstream.Target {
	current := rr.Upstream.Targets[rr.NextIndex]
	rr.NextIndex = (rr.NextIndex + 1) % len(rr.Upstream.Targets)
	return current
}
