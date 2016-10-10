package upstream

import (
	"fmt"
	"time"

	"github.com/Dataman-Cloud/janitor/src/util"

	log "github.com/Sirupsen/logrus"
	consulApi "github.com/hashicorp/consul/api"
)

const (
	BORG_TAG = "foobar"
)

type UpstreamLoader interface {
	Poll()
	List() []Upstream
	Get(serviceName string) *Upstream
	Remove(upstream *Upstream)
}

type ConsulUpstreamLoader struct {
	UpstreamLoader

	ConsulClient *consulApi.Client
	PollTicker   *time.Ticker

	Upstreams []Upstream
}

func InitConsulUpstreamLoader(consulAddr string, pollInterval time.Duration) (*ConsulUpstreamLoader, error) {
	upstreamFromConsul := &ConsulUpstreamLoader{}

	consulConfig := consulApi.DefaultConfig()
	consulConfig.Address = consulAddr

	client, err := consulApi.NewClient(consulConfig)
	if err != nil {
		return nil, err
	}
	upstreamFromConsul.ConsulClient = client
	upstreamFromConsul.PollTicker = time.NewTicker(pollInterval)
	upstreamFromConsul.Upstreams = make([]Upstream, 0)

	upstreamFromConsul.Poll()

	return upstreamFromConsul, nil
}

func (upstreamFromConsul *ConsulUpstreamLoader) Poll() {
	for {
		<-upstreamFromConsul.PollTicker.C
		go func() {
			latestUpstreams := make([]Upstream, 0)

			services, _, err := upstreamFromConsul.ConsulClient.Catalog().Services(nil)
			//fmt.Println(services)
			if err != nil {
				log.Errorf("poll upstream from consul got err: ", err)
				return
			}

			for serviceName, tags := range services {
				if util.SliceContains(tags, BORG_TAG) {
					catalogServices, _, err := upstreamFromConsul.ConsulClient.Catalog().Service(serviceName, BORG_TAG, nil)
					if err != nil {
						log.Errorf("poll upstream from consul got err: ", err)
					}

					var upstream Upstream
					upstream.ServiceName = serviceName
					upstream.Targets = make([]Target, 0)
					for _, service := range catalogServices {
						var target Target
						target.Address = service.Address
						target.ServiceID = service.ServiceID
						target.ServiceName = service.ServiceName
						target.Node = service.Node
						target.ServiceAddress = service.ServiceAddress
						target.ServicePort = service.ServicePort
						upstream.Targets = append(upstream.Targets, target)
					}
					latestUpstreams = append(latestUpstreams, upstream)
				}
			}

			//TODO mutex to sync with reading
			upstreamFromConsul.Upstreams = make([]Upstream, 0)
			for _, u := range latestUpstreams {
				upstreamFromConsul.Upstreams = append(upstreamFromConsul.Upstreams, u)
			}

			fmt.Println(len(upstreamFromConsul.Upstreams))

			for _, u := range upstreamFromConsul.Upstreams {
				fmt.Println(u.ServiceName)

				fmt.Println(u.Targets)
			}

		}()
	}
}

func (upstreamFromConsul *ConsulUpstreamLoader) List() []Upstream {
	return upstreamFromConsul.Upstreams
}

func (upstreamFromConsul *ConsulUpstreamLoader) Get(serviceName string) *Upstream {
	return nil
}

func (upstreamFromConsul *ConsulUpstreamLoader) Remove(upstream *Upstream) {}
