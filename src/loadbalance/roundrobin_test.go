package loadbalance

import (
	"github.com/Dataman-Cloud/janitor/src/upstream"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewRoundRobinLoadBalancer(t *testing.T) {
	rr := NewRoundRobinLoadBalancer()
	assert.NotNil(t, rr)
}

func TestSeed(t *testing.T) {
	rr := NewRoundRobinLoadBalancer()
	u := &upstream.Upstream{}
	rr.Seed(u)

	assert.Equal(t, rr.Upstream, u)
	assert.Equal(t, rr.NextIndex, 0)
}

func TestNext(t *testing.T) {
	rr := NewRoundRobinLoadBalancer()
	u := &upstream.Upstream{Targets: make([]*upstream.Target, 0)}
	u.Targets = append(u.Targets, &upstream.Target{})
	rr.Seed(u)

	assert.Equal(t, rr.Next(), u.Targets[0])
}
