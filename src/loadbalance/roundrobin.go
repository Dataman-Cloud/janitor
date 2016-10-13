package loadbalance

import (
	"github.com/Dataman-Cloud/janitor/src/upstream"
)

type LoadBalancer interface {
	Next() *upstream.Target
	Seed(targets []*upstream.Target)
}

type RoundRobinLoaderBalancer struct {
	Targets   []*upstream.Target
	NextIndex int
}

func NewRoundRobinLoaderBalancer() *RoundRobinLoaderBalancer {
	return &RoundRobinLoaderBalancer{
		Targets: make([]*upstream.Target, 0),
	}
}

func (rr *RoundRobinLoaderBalancer) Seed(targets []*upstream.Target) {
	rr.Targets = targets
	rr.NextIndex = 0
}

func (rr *RoundRobinLoaderBalancer) Next() *upstream.Target {
	current := rr.Targets[rr.NextIndex]
	rr.NextIndex = (rr.NextIndex + 1) % len(rr.Targets)
	return current
}
