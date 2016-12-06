package upstream

import (
	"fmt"
	"net"
	"strings"
	"sync"

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

func InitSwanUpstreamLoader(defaultUpstreamIp net.IP, defaultPort string) (*SwanUpstreamLoader, error) {
	swanUpstreamLoader := &SwanUpstreamLoader{}
	swanUpstreamLoader.changeNotify = make(chan bool, 64)
	swanUpstreamLoader.Upstreams = make([]*Upstream, 0)
	swanUpstreamLoader.DefaultUpstreamIp = defaultUpstreamIp
	swanUpstreamLoader.Port = defaultPort
	swanUpstreamLoader.Proto = "http"
	swanUpstreamLoader.swanEventChan = make(chan *AppEventNotify, 1)
	go swanUpstreamLoader.Poll()
	return swanUpstreamLoader, nil
}

func (swanUpstreamLoader *SwanUpstreamLoader) Poll() {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("SwanUpstreamLoader poll got error: %s", err)
			swanUpstreamLoader.Poll() // execute poll again
		}
	}()

	for {
		//var appEvent *AppEventNotify
		log.Debug("upstreamLoader is listening app event...")
		appEvent := <-swanUpstreamLoader.swanEventChan
		log.Debug("upstreamLoader receive one app event:%s", appEvent)
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
							u.SetState(STATE_CHANGED)
							break
						}
					}
					if !targetDuplicated {
						target.Upstream = u
						u.Targets = append(u.Targets, target)
						u.SetState(STATE_CHANGED)
					}

				}
			}
			if !upstreamDuplicated {
				target.Upstream = upstream
				upstream.SetState(STATE_NEW)
				upstream.Targets = append(upstream.Targets, target)
				swanUpstreamLoader.Upstreams = append(swanUpstreamLoader.Upstreams, upstream)
			}
		case "delete":
			upstream := buildSwanUpstream(appEvent, swanUpstreamLoader.DefaultUpstreamIp, swanUpstreamLoader.Port, swanUpstreamLoader.Proto)
			target := buildSwanTarget(appEvent)
			for _, u := range swanUpstreamLoader.Upstreams {
				if u.FieldsEqual(upstream) {
					u.Remove(target)
					fmt.Printf("delete target from upstream:%+v\n", target)
					if len(u.Targets) > 0 {
						u.SetState(STATE_CHANGED)
					} else {
						u.StaleMark = true
					}
				}
			}
		}
		swanUpstreamLoader.changeNotify <- true
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
	fmt.Printf("taskNamespaces:%s\n", taskNamespaces)
	taskNum := taskNamespaces[0]
	fmt.Printf("taskNamespaces[1:]:%s\n", taskNamespaces[1:])
	appName := strings.Join(taskNamespaces[1:], ".")
	upstream.ServiceName = appName
	upstream.FrontendIp = defaultUpstreamIp.String()
	upstream.FrontendPort = port
	upstream.FrontendProto = proto
	upstream.Targets = make([]*Target, 0)
	upstream.StaleMark = false
	fmt.Printf("taskNum:%s\n", taskNum)
	fmt.Printf("appName:%s\n", appName)
	return &upstream
}
