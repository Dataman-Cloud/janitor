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

func InitAndStart(Config config.Config, ctx context.Context) (*UpstreamSource, error) {
	var upstreamSource UpstreamSource
	var err error
	switch strings.ToLower(Config.Upstream.SourceType) {
	case "consul":
		upstreamSource, err = InitConsulUpstreamSource(Config.Upstream.ConsulAddr, Config.Upstream.PollInterval)
		if err != nil {
			return nil, err
		}
	}

	return &upstreamSource, nil
}
