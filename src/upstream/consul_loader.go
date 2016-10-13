package upstream

import (
	"fmt"
	"strings"
	"time"

	"github.com/Dataman-Cloud/janitor/src/util"

	log "github.com/Sirupsen/logrus"
	consulApi "github.com/hashicorp/consul/api"
	"golang.org/x/net/context"
)

const (
	BORG_TAG            = "borg"
	BORG_FRONTEND_IP    = "borg-frontend-ip"
	BORG_FRONTEND_PORT  = "borg-frontend-port"
	BORG_FRONTEND_PROTO = "borg-frontend-proto"

	CONSUL_UPSTREAM_LOADER_KEY = "ConsulUpstreamLoader"
)

type ConsulUpstreamLoader struct {
	UpstreamLoader

	ConsulClient *consulApi.Client
	PollTicker   *time.Ticker

	Upstreams     []*Upstream
	ChangeTrigger chan bool
}

func ConsulUpstreamLoaderFromContext(ctx context.Context) *ConsulUpstreamLoader {
	upstreamLoader := ctx.Value(CONSUL_UPSTREAM_LOADER_KEY)
	return upstreamLoader.(*ConsulUpstreamLoader)
}

func InitConsulUpstreamLoader(consulAddr string, pollInterval time.Duration) (*ConsulUpstreamLoader, error) {
	consulUpstreamLoader := &ConsulUpstreamLoader{}

	consulConfig := consulApi.DefaultConfig()
	consulConfig.Address = consulAddr

	client, err := consulApi.NewClient(consulConfig)
	if err != nil {
		return nil, err
	}
	consulUpstreamLoader.ConsulClient = client
	consulUpstreamLoader.PollTicker = time.NewTicker(pollInterval)
	consulUpstreamLoader.Upstreams = make([]*Upstream, 0)

	go consulUpstreamLoader.Poll()

	return consulUpstreamLoader, nil
}

func (consulUpstreamLoader *ConsulUpstreamLoader) Poll() {
	for {
		<-consulUpstreamLoader.PollTicker.C
		go func() {
			log.Debug("consul upstream loader loading services form consul")
			latestUpstreams := make([]*Upstream, 0)

			services, _, err := consulUpstreamLoader.ConsulClient.Catalog().Services(nil)
			if err != nil {
				log.Errorf("poll upstream from consul got err: ", err)
				return
			}

			for serviceName, tags := range services {
				if util.SliceContains(tags, BORG_TAG) {
					catalogServices, _, err := consulUpstreamLoader.ConsulClient.Catalog().Service(serviceName, BORG_TAG, nil)
					if err != nil {
						log.Errorf("poll upstream from consul got err: ", err)
					}

					var upstream Upstream
					upstream.ServiceName = serviceName
					upstream.FrontendIp = ParseValueFromTags(BORG_FRONTEND_IP, tags)
					upstream.FrontendPort = ParseValueFromTags(BORG_FRONTEND_PORT, tags)
					upstream.FrontendProto = ParseValueFromTags(BORG_FRONTEND_PROTO, tags)
					upstream.Targets = make([]*Target, 0)
					for _, service := range catalogServices {
						var target Target
						target.Address = service.Address
						target.ServiceID = service.ServiceID
						target.ServiceName = service.ServiceName
						target.Node = service.Node
						target.ServiceAddress = service.ServiceAddress
						target.ServicePort = fmt.Sprintf("%d", service.ServicePort)
						target.Upstream = &upstream
						upstream.Targets = append(upstream.Targets, &target)
					}
					latestUpstreams = append(latestUpstreams, &upstream)
				}
			}

			//TODO mutex to sync with reading
			consulUpstreamLoader.Upstreams = make([]*Upstream, 0)
			for _, u := range latestUpstreams {
				consulUpstreamLoader.Upstreams = append(consulUpstreamLoader.Upstreams, u)
			}
		}()
	}
}

func (consulUpstreamLoader *ConsulUpstreamLoader) List() []*Upstream {
	return consulUpstreamLoader.Upstreams
}

func (consulUpstreamLoader *ConsulUpstreamLoader) Get(serviceName string) *Upstream {
	return nil
}

func (consulUpstreamLoader *ConsulUpstreamLoader) Remove(upstream *Upstream) {}

func ParseValueFromTags(what string, tags []string) string {
	for _, tag := range tags {
		if strings.HasPrefix(tag, what) {
			if len(strings.Split(tag, ":")) == 2 {
				return strings.Split(tag, ":")[1]
			}
		}
	}
	return ""
}
