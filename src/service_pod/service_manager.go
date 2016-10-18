package service_pod

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Dataman-Cloud/janitor/src/handler"
	"github.com/Dataman-Cloud/janitor/src/listener"
	"github.com/Dataman-Cloud/janitor/src/upstream"

	log "github.com/Sirupsen/logrus"
	consulApi "github.com/hashicorp/consul/api"
	"golang.org/x/net/context"
)

const (
	SESSION_RENEW_INTERVAL = time.Second * 2
)

type ServiceManager struct {
	servicePods map[upstream.UpstreamKey]*ServicePod

	handlerFactory  *handler.Factory
	listenerManager *listener.Manager
	upstreamLoader  upstream.UpstreamLoader
	consulClient    *consulApi.Client
	ctx             context.Context
	forkMutex       sync.Mutex
	rwMutex         sync.RWMutex

	sessionIDWithTTY   string
	sessionRenewTicker *time.Ticker
}

func NewServiceManager(ctx context.Context) *ServiceManager {
	serviceManager := &ServiceManager{}

	handlerFactory_ := ctx.Value(handler.HANDLER_FACTORY_KEY)
	serviceManager.handlerFactory = handlerFactory_.(*handler.Factory)

	listenerManager_ := ctx.Value(listener.LISTENER_MANAGER_KEY)
	serviceManager.listenerManager = listenerManager_.(*listener.Manager)

	serviceManager.upstreamLoader = ctx.Value(upstream.CONSUL_UPSTREAM_LOADER_KEY).(*upstream.ConsulUpstreamLoader)
	serviceManager.consulClient = serviceManager.upstreamLoader.(*upstream.ConsulUpstreamLoader).ConsulClient

	serviceManager.servicePods = make(map[upstream.UpstreamKey]*ServicePod)
	serviceManager.ctx = ctx

	serviceManager.sessionRenewTicker = time.NewTicker(SESSION_RENEW_INTERVAL)

	var err error
	serviceManager.sessionIDWithTTY, _, err = serviceManager.consulClient.Session().Create(
		&consulApi.SessionEntry{
			Behavior: "delete",
			TTL:      "10s",
		}, nil,
	)
	if err != nil {
		log.Errorf("create a session error: %s", err)
	}

	serviceManager.KeepSessionAlive()

	return serviceManager
}

func (manager *ServiceManager) ForkNewServicePod(upstream *upstream.Upstream) *ServicePod {
	manager.forkMutex.Lock()
	defer manager.forkMutex.Unlock()

	httpServer := &http.Server{Handler: manager.handlerFactory.HttpHandler(upstream)}

	listener := manager.listenerManager.FetchListener(upstream.Key())
	pod := NewServicePod(upstream, httpServer, listener)
	manager.servicePods[upstream.Key()] = pod

	manager.UpdateKVApplicationList()
	return pod
}

func (manager *ServiceManager) FetchServicePod(key upstream.UpstreamKey) *ServicePod {
	manager.rwMutex.RLock()
	defer manager.rwMutex.RUnlock()

	return manager.servicePods[key]
}

func (manager *ServiceManager) KillServicePod(u *upstream.Upstream) error {
	manager.rwMutex.Lock()
	defer manager.rwMutex.Unlock()

	pod, found := manager.servicePods[u.Key()]
	if found {
		pod.Dispose()
		delete(manager.servicePods, u.Key())
		manager.upstreamLoader.Remove(u)
		manager.listenerManager.Remove(u.Key())
	}
	manager.UpdateKVApplicationList()
	return nil
}

func (manager *ServiceManager) KeepSessionAlive() {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Error("KeepSessionAlive got error: %s", err)
				manager.KeepSessionAlive()
			}
		}()

		for {
			<-manager.sessionRenewTicker.C
			_, _, err := manager.consulClient.Session().Renew(manager.sessionIDWithTTY, nil)
			if err != nil {
				log.Errorf("renew a session error: %s", err)
			}
		}
	}()
}

func (manager *ServiceManager) UpdateKVApplicationList() {
	// use consulClient For short, UGLY
	kv := manager.consulClient.KV()

	for k, v := range manager.servicePods {
		p := &consulApi.KVPair{Key: fmt.Sprintf("%s@%s", v.upstream.ServiceName, k.Ip),
			Value:   []byte(fmt.Sprintf("%s://%s:%s", k.Proto, k.Ip, k.Port)),
			Session: manager.sessionIDWithTTY,
		}
		_, err := kv.Put(p, nil)
		if err != nil {
			log.Errorf("persist service entries error %s", err)
		}
	}
}

func (manager *ServiceManager) ClusterAddressList(prefix string) []string {
	// use consulClient For short, UGLY
	serviceEntriesWithPrefix := make([]string, 0)
	kv := manager.consulClient.KV()
	trimedPrefix := strings.TrimLeft(prefix, "/")
	kvPairs, _, err := kv.List(trimedPrefix, nil)
	if err != nil {
		log.Errorf("kv list error %s", err)
	}

	for _, v := range kvPairs {
		serviceEntriesWithPrefix = append(serviceEntriesWithPrefix, string(v.Value))
	}

	return serviceEntriesWithPrefix
}

func (manager *ServiceManager) PortsOccupied() []string {
	ports := make([]string, 0)
}
