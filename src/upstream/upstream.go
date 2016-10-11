package upstream

import (
	"strings"

	"github.com/Dataman-Cloud/janitor/src/config"

	"golang.org/x/net/context"
)

type Target struct {
	Address        string
	ServiceName    string
	ServiceID      string
	ServiceAddress string
	ServicePort    int
	Node           string
}

type Upstream struct {
	ServiceName string   `json:"ServiceName"`
	Targets     []Target `json:"Target"`
}

type UpstreamLoader interface {
	Poll()
	List() []Upstream
	Get(serviceName string) *Upstream
	Remove(upstream *Upstream)
}

func InitAndStart(ctx context.Context, Config config.Config) (UpstreamLoader, error) {
	var upstreamLoader UpstreamLoader
	var err error
	switch strings.ToLower(Config.Upstream.SourceType) {
	case "consul":
		upstreamLoader, err = InitConsulUpstreamLoader(Config.Upstream.ConsulAddr, Config.Upstream.PollInterval)
		if err != nil {
			return nil, err
		}
	}

	return upstreamLoader, nil
}
