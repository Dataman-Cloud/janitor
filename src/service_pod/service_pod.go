package service_pod

import (
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
	Key      string
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

	handerfactory_ := ctx.value(handler.HANDLER_FACTORY_KEY)
	handerfactory := handerfactory_.(*handler.factory)

	listenerManager_ := ctx.value(listener.LISTEN_MANAGER_KEY)
	listenerManager := listenerManager_.(*listener.Manager)

	pod.Server = &http.Server{
		Handler:  handerFactory.HttpHandler(upstream),
		Listener: listenerManager.FetchListener(upstream.FrontendIp, upstream.FrontendPort),
	}

	return pod
}

func (pod *ServicePod) Run() {
	go func() {
		pod.Server.Serve(pod.Listener)
	}()
}

func (pod *ServicePod) Dispose() {
	//<-pod.stopCh
	err := pod.Listener.Close()
	if err != nil {
		log.Error("error close listener for pod <%s>,  the error is [%s]", pod.Key, err)
	}
}
