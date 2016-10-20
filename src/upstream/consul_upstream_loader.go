package upstream

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/Dataman-Cloud/janitor/src/util"

	log "github.com/Sirupsen/logrus"
	consulApi "github.com/hashicorp/consul/api"
	"golang.org/x/net/context"
)

const (
	BORG_TAG            = "BORG"
	BORG_FRONTEND_IP    = "BORG_FRONTEND_IP"
	BORG_FRONTEND_PORT  = "BORG_FRONTEND_POR"
	BORG_FRONTEND_PROTO = "BORG_FRONTEND_PROTO"

	CONSUL_UPSTREAM_LOADER_KEY = "ConsulUpstreamLoader"
)

type ConsulUpstreamLoader struct {
	UpstreamLoader

	ConsulClient *consulApi.Client
	PollTicker   *time.Ticker

	Upstreams    []*Upstream
	changeNotify chan bool
	sync.Mutex
	DefaultUpstreamIp net.IP
}

func ConsulUpstreamLoaderFromContext(ctx context.Context) *ConsulUpstreamLoader {
	upstreamLoader := ctx.Value(CONSUL_UPSTREAM_LOADER_KEY)
	return upstreamLoader.(*ConsulUpstreamLoader)
}

func InitConsulUpstreamLoader(consulAddr string, defaultUpstreamIp net.IP, pollInterval time.Duration) (*ConsulUpstreamLoader, error) {
	consulUpstreamLoader := &ConsulUpstreamLoader{}

	consulUpstreamLoader.changeNotify = make(chan bool, 64)
	consulConfig := consulApi.DefaultConfig()
	consulConfig.Address = consulAddr

	client, err := consulApi.NewClient(consulConfig)
	if err != nil {
		return nil, err
	}
	consulUpstreamLoader.ConsulClient = client
	consulUpstreamLoader.PollTicker = time.NewTicker(pollInterval)
	consulUpstreamLoader.Upstreams = make([]*Upstream, 0)
	consulUpstreamLoader.DefaultUpstreamIp = defaultUpstreamIp

	go consulUpstreamLoader.Poll()

	return consulUpstreamLoader, nil
}

func (consulUpstreamLoader *ConsulUpstreamLoader) Poll() {
	defer func() {
		if err := recover(); err != nil {
			log.Error("ConsulUpstreamLoader poll got error: %s", err)
			consulUpstreamLoader.Poll() // execute poll again
		}
	}()

	for {
		<-consulUpstreamLoader.PollTicker.C
		log.Debug("consul upstream loader loading services form consul")

		services, _, err := consulUpstreamLoader.ConsulClient.Catalog().Services(nil)
		if err != nil {
			log.Errorf("poll upstream from consul got err: ", err)
			return
		}

		newUpstreams := make([]*Upstream, 0)
		for serviceName, tags := range services {
			// skip services not intent for local server
			if !util.SliceContains(tags, BORG_TAG) {
				log.Debugf("application does't contain tag BORG")
				continue
			}
			serviceEntries, _, err := consulUpstreamLoader.ConsulClient.Health().Service(serviceName, BORG_TAG, true, nil)
			if err != nil {
				log.Errorf("poll upstream from consul got err: ", err)
			}

			upstream := buildUpstream(serviceName, tags, serviceEntries, consulUpstreamLoader.DefaultUpstreamIp.String())
			newUpstreams = append(newUpstreams, &upstream)
		}

		// find and mark oldUpstream that are stale
		for _, oldUpstream := range consulUpstreamLoader.Upstreams {
			shouldSweep := true
			for _, newUpstream := range newUpstreams {
				if oldUpstream.FieldsEqual(newUpstream) && len(newUpstream.Targets) != 0 {
					shouldSweep = false
				}
			}

			if shouldSweep {
				oldUpstream.StaleMark = true
			}
		}

		// find and mark oldUpstream that are changed with targets
		for _, oldUpstream := range consulUpstreamLoader.Upstreams {
			for _, newUpstream := range newUpstreams {
				if oldUpstream.FieldsEqual(newUpstream) && oldUpstream.FieldsEqualButTargetsDiffer(newUpstream) {
					oldUpstream.SetState(STATE_CHANGED)
					oldUpstream.Targets = newUpstream.Targets
				}
			}
		}

		upstreamsShouldAppend := make([]*Upstream, 0)
		for _, newUpstream := range newUpstreams {
			notInTheSlice := true
			for _, oldUpstream := range consulUpstreamLoader.Upstreams {
				if oldUpstream.FieldsEqual(newUpstream) {
					notInTheSlice = false
				}
			}

			if notInTheSlice {
				upstreamsShouldAppend = append(upstreamsShouldAppend, newUpstream)
			}
		}

		for _, upstream := range upstreamsShouldAppend {
			if len(upstream.Targets) > 0 {
				log.Infof("new upstream found %s", upstream.Key())
				consulUpstreamLoader.Upstreams = append(consulUpstreamLoader.Upstreams, upstream)
			}
		}

		consulUpstreamLoader.changeNotify <- true
	}
}

func (consulUpstreamLoader *ConsulUpstreamLoader) List() []*Upstream {
	consulUpstreamLoader.Lock()
	defer consulUpstreamLoader.Unlock()
	return consulUpstreamLoader.Upstreams
}

func (consulUpstreamLoader *ConsulUpstreamLoader) ServiceEntries() []string {
	entryList := make([]string, 0)
	for _, u := range consulUpstreamLoader.Upstreams {
		entry := fmt.Sprintf("%s://%s:%s", u.Key().Proto, u.Key().Ip, u.Key().Port)
		entryList = append(entryList, entry)
	}

	return entryList
}

func (consulUpstreamLoader *ConsulUpstreamLoader) Get(serviceName string) *Upstream {
	return nil
}

func (consulUpstreamLoader *ConsulUpstreamLoader) Remove(upstream *Upstream) {
	index := -1
	for k, v := range consulUpstreamLoader.Upstreams {
		if v == upstream {
			index = k
			break
		}
	}

	if index >= 0 {
		consulUpstreamLoader.Upstreams = append(consulUpstreamLoader.Upstreams[:index], consulUpstreamLoader.Upstreams[index+1:]...)
	}
}

func (consulUpstreamLoader *ConsulUpstreamLoader) ChangeNotify() <-chan bool {
	return consulUpstreamLoader.changeNotify
}

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

func buildUpstream(serviceName string, tags []string, serviceEntries []*consulApi.ServiceEntry, defaultUpstreamIp string) Upstream {
	var upstream Upstream
	upstream.ServiceName = serviceName
	upstream.FrontendPort = ParseValueFromTags(BORG_FRONTEND_PORT, tags)
	upstream.FrontendIp = defaultUpstreamIp
	upstream.FrontendProto = ParseValueFromTags(BORG_FRONTEND_PROTO, tags)
	upstream.Targets = make([]*Target, 0)
	upstream.StaleMark = false
	upstream.SetState(STATE_NEW)

	for _, service := range serviceEntries {
		var target Target
		target.Address = service.Node.Address
		target.Node = service.Node.Node
		target.ServiceID = service.Service.ID
		target.ServiceName = serviceName
		target.ServiceAddress = service.Service.Address
		target.ServicePort = fmt.Sprintf("%d", service.Service.Port)
		target.Upstream = &upstream
		upstream.Targets = append(upstream.Targets, &target)
	}
	return upstream
}
