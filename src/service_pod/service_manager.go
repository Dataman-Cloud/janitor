package service_pod

import (
	"net/http"
	"sync"

	"github.com/Dataman-Cloud/janitor/src/handler"
	"github.com/Dataman-Cloud/janitor/src/listener"
	"github.com/Dataman-Cloud/janitor/src/upstream"

	"golang.org/x/net/context"
)

type ServiceManager struct {
	servicePods map[upstream.UpstreamKey]*ServicePod

	handlerFactory  *handler.Factory
	listenerManager *listener.Manager
	upstreamLoader  upstream.UpstreamLoader
	ctx             context.Context
	forkMutex       sync.Mutex
	rwMutex         sync.RWMutex
}

func NewServiceManager(ctx context.Context) *ServiceManager {
	serviceManager := &ServiceManager{}

	handlerFactory_ := ctx.Value(handler.HANDLER_FACTORY_KEY)
	serviceManager.handlerFactory = handlerFactory_.(*handler.Factory)

	listenerManager_ := ctx.Value(listener.LISTENER_MANAGER_KEY)
	serviceManager.listenerManager = listenerManager_.(*listener.Manager)

	serviceManager.upstreamLoader = ctx.Value(upstream.CONSUL_UPSTREAM_LOADER_KEY).(*upstream.ConsulUpstreamLoader)

	serviceManager.servicePods = make(map[upstream.UpstreamKey]*ServicePod)
	serviceManager.ctx = ctx

	return serviceManager
}

func (manager *ServiceManager) ForkNewServicePod(upstream *upstream.Upstream) *ServicePod {
	manager.forkMutex.Lock()
	defer manager.forkMutex.Unlock()

	httpServer := &http.Server{Handler: manager.handlerFactory.HttpHandler(upstream)}

	listener := manager.listenerManager.FetchListener(upstream.Key())
	pod := NewServicePod(upstream, httpServer, listener)
	manager.servicePods[upstream.Key()] = pod

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
	return nil
}
