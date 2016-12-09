package upstream

import (
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/Dataman-Cloud/janitor/src/loadbalance"

	log "github.com/Sirupsen/logrus"
)

const (
	SWAN_UPSTREAM_LOADER_KEY = "SwanUpstreamLoader"
)

type AppEventNotify struct {
	Operation     string
	TaskName      string
	AgentHostName string
	AgentPort     string
}

type SwanUpstreamLoader struct {
	UpstreamLoader
	Upstreams    []*Upstream
	changeNotify chan bool
	sync.Mutex
	swanEventChan     chan *AppEventNotify
	DefaultUpstreamIp net.IP
	Port              string
	Proto             string
}

func InitSwanUpstreamLoader(defaultUpstreamIp net.IP, defaultPort string, defaultProto string) (*SwanUpstreamLoader, error) {
	swanUpstreamLoader := &SwanUpstreamLoader{}
	swanUpstreamLoader.changeNotify = make(chan bool, 64)
	swanUpstreamLoader.Upstreams = make([]*Upstream, 0)
	swanUpstreamLoader.DefaultUpstreamIp = defaultUpstreamIp
	swanUpstreamLoader.Port = defaultPort
	swanUpstreamLoader.Proto = defaultProto
	swanUpstreamLoader.swanEventChan = make(chan *AppEventNotify, 1)
	go swanUpstreamLoader.Poll()
	return swanUpstreamLoader, nil
}

func (swanUpstreamLoader *SwanUpstreamLoader) Poll() {
	log.Debug("SwanUpstreamLoader starts listening app event...")
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("SwanUpstreamLoader poll got error: %s", err)
			swanUpstreamLoader.Poll() // execute poll again
		}
	}()

	for {
		//var appEvent *AppEventNotify
		appEvent := <-swanUpstreamLoader.swanEventChan
		log.Debugf("SwanUpstreamLoader receive one app event:%s", appEvent)
		switch strings.ToLower(appEvent.Operation) {
		case "add":
			upstream := buildSwanUpstream(appEvent, swanUpstreamLoader.DefaultUpstreamIp, swanUpstreamLoader.Port, swanUpstreamLoader.Proto)
			target := buildSwanTarget(appEvent)
			upstreamDuplicated := false
			for _, u := range swanUpstreamLoader.Upstreams {
				if u.FieldsEqual(upstream) {
					upstreamDuplicated = true
					targetDuplicated := false
					for _, t := range u.Targets {
						if t.Equal(target) {
							targetDuplicated = true
							break
						} else if t.ServiceID == target.ServiceID {
							target.Upstream = u
							u.Remove(t)
							u.Targets = append(u.Targets, target)
							log.Debugf("Target [%s] updated in upstream [%s]", target.ServiceID, u.ServiceName)
							u.SetState(STATE_CHANGED)
							break
						}
					}
					if !targetDuplicated {
						target.Upstream = u
						u.Targets = append(u.Targets, target)
						log.Debugf("Target [%s] added in upstream [%s]", target.ServiceID, u.ServiceName)
						u.SetState(STATE_CHANGED)
					}

				}
			}
			if !upstreamDuplicated {
				//set state
				upstream.SetState(STATE_NEW)
				//add target
				target.Upstream = upstream
				upstream.Targets = append(upstream.Targets, target)
				log.Debugf("Target [%s] added in upstream [%s]", target.ServiceID, upstream.ServiceName)
				//add loadbalance when upstream created
				loadBalance := loadbalance.NewRoundRobinLoadBalancer()
				loadBalance.Seed()
				upstream.LoadBalance = loadBalance
				swanUpstreamLoader.Upstreams = append(swanUpstreamLoader.Upstreams, upstream)
				log.Debugf("Upstream [%s] created", upstream.ServiceName)
			}
		case "delete":
			upstream := buildSwanUpstream(appEvent, swanUpstreamLoader.DefaultUpstreamIp, swanUpstreamLoader.Port, swanUpstreamLoader.Proto)
			target := buildSwanTarget(appEvent)
			for _, u := range swanUpstreamLoader.Upstreams {
				if u.FieldsEqual(upstream) {
					u.Remove(target)
					log.Debugf("Target %s removed from upstream [%s]", target.ServiceID, upstream.ServiceName)
					if len(u.Targets) > 0 {
						u.SetState(STATE_CHANGED)
					} else {
						u.StaleMark = true
					}
				}
			}
		}
	}
}

func (swanUpstreamLoader *SwanUpstreamLoader) List() []*Upstream {
	swanUpstreamLoader.Lock()
	defer swanUpstreamLoader.Unlock()
	return swanUpstreamLoader.Upstreams
}

func (swanUpstreamLoader *SwanUpstreamLoader) SwanEventChan() chan<- *AppEventNotify {
	return swanUpstreamLoader.swanEventChan
}

func (swanUpstreamLoader *SwanUpstreamLoader) ServiceEntries() []string {
	entryList := make([]string, 0)
	for _, u := range swanUpstreamLoader.Upstreams {
		entry := fmt.Sprintf("%s://%s:%s", u.Key().Proto, u.Key().Ip, u.Key().Port)
		entryList = append(entryList, entry)
	}

	return entryList
}

func (swanUpstreamLoader *SwanUpstreamLoader) Get(serviceName string) *Upstream {
	for _, u := range swanUpstreamLoader.Upstreams {
		if u.ServiceName == serviceName {
			return u
			break
		}
	}
	return nil
}

func (swanUpstreamLoader *SwanUpstreamLoader) Remove(upstream *Upstream) {
	index := -1
	for k, v := range swanUpstreamLoader.Upstreams {
		if v == upstream {
			index = k
			break
		}
	}

	if index >= 0 {
		swanUpstreamLoader.Upstreams = append(swanUpstreamLoader.Upstreams[:index], swanUpstreamLoader.Upstreams[index+1:]...)
	}
}

func (swanUpstreamLoader *SwanUpstreamLoader) ChangeNotify() <-chan bool {
	return swanUpstreamLoader.changeNotify
}

func buildSwanTarget(appEvent *AppEventNotify) *Target {
	// create a new target
	var target Target
	taskNamespaces := strings.Split(appEvent.TaskName, ".")
	taskNum := taskNamespaces[0]
	appName := strings.Join(taskNamespaces[1:], ".")
	target.Address = appEvent.AgentHostName
	target.ServiceName = appName
	target.ServiceID = taskNum
	target.ServiceAddress = appEvent.AgentHostName
	target.ServicePort = appEvent.AgentPort
	return &target
}

func buildSwanUpstream(appEvent *AppEventNotify, defaultUpstreamIp net.IP, port string, proto string) *Upstream {
	// create a new upstream
	var upstream Upstream
	taskNamespaces := strings.Split(appEvent.TaskName, ".")
	appName := strings.Join(taskNamespaces[1:], ".")
	upstream.ServiceName = appName
	upstream.FrontendIp = defaultUpstreamIp.String()
	upstream.FrontendPort = port
	upstream.FrontendProto = proto
	upstream.Targets = make([]*Target, 0)
	upstream.StaleMark = false
	return &upstream
}
