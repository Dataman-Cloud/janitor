package service_pod

import (
	"fmt"
	"net/http"

	//"github.com/Dataman-Cloud/janitor/src/config"
	"github.com/Dataman-Cloud/janitor/src/handler"
	"github.com/Dataman-Cloud/janitor/src/listener"
	"github.com/Dataman-Cloud/janitor/src/upstream"

	log "github.com/Sirupsen/logrus"
	"github.com/armon/go-proxyproto"
	"golang.org/x/net/context"
)

type ServicePod struct {
	Key      upstream.UpstreamKey
	Server   *http.Server
	Ctx      context.Context
	Listener *proxyproto.Listener
	Upstream *upstream.Upstream

	stopCh chan bool
}

func NewServicePod(ctx context.Context, upstream *upstream.Upstream) *ServicePod {
	pod := &ServicePod{
		Key: upstream.Key(),

		Ctx:      ctx,
		stopCh:   make(chan bool, 1),
		Upstream: upstream,
	}

	handerfactory_ := ctx.Value(handler.HANDLER_FACTORY_KEY)
	handerfactory := handerfactory_.(*handler.Factory)

	listenerManager_ := ctx.Value(listener.LISTENER_MANAGER_KEY)
	listenerManager := listenerManager_.(*listener.Manager)

	pod.Server = &http.Server{
		Handler: handerfactory.HttpHandler(upstream),
	}

	pod.Listener = listenerManager.FetchListener(listener.ListenerKey{Ip: upstream.FrontendIp, Port: upstream.FrontendPort})
	return pod
}

func (pod *ServicePod) Invalid() {
	// do nothing for the present
}

func (pod *ServicePod) Run() {
	go func() {
		fmt.Println("start runing pod")
		err := pod.Server.Serve(pod.Listener)
		if err != nil {
			log.Error("pod Run goroutine error  <%s>,  the error is [%s]", pod.Key, err)
		}
	}()
}

func (pod *ServicePod) Dispose() {
	listenerManager_ := pod.Ctx.Value(listener.LISTENER_MANAGER_KEY)
	listenerManager := listenerManager_.(*listener.Manager)
	listenerManager.Remove(listener.ListenerKey{Ip: pod.Upstream.FrontendIp, Port: pod.Upstream.FrontendPort})
	err := pod.Listener.Close()
	if err != nil {
		log.Error("dispose error close listener for pod <%s>,  the error is [%s]", pod.Key, err)
	}
}
