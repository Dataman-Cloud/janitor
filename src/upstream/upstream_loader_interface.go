package upstream

import (
	"strings"

	"github.com/Dataman-Cloud/janitor/src/config"

	"golang.org/x/net/context"
)

type UpstreamLoader interface {
	Poll()
	List() []*Upstream
	Get(serviceName string) *Upstream
	Remove(upstream *Upstream)
	ChangeNotify() <-chan bool
}

func InitAndStartUpstreamLoader(ctx context.Context, Config config.Config) (UpstreamLoader, error) {
	var upstreamLoader UpstreamLoader
	var err error
	switch strings.ToLower(Config.Upstream.SourceType) {
	case "consul":
		upstreamLoader, err = InitConsulUpstreamLoader(Config.Upstream.ConsulAddr, Config.Listener.IP, Config.Upstream.PollInterval)
		if err != nil {
			return nil, err
		}
	}

	return upstreamLoader, nil
}
