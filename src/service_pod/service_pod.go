package service_pod

import (
	"net/http"

	"github.com/Dataman-Cloud/janitor/src/upstream"

	log "github.com/Sirupsen/logrus"
	"github.com/armon/go-proxyproto"
)

type ServicePod struct {
	Key        upstream.UpstreamKey
	httpServer *http.Server
	upstream   *upstream.Upstream
	listener   *proxyproto.Listener

	stopCh chan bool
}

func NewServicePod(upstream *upstream.Upstream, HttpServer *http.Server, Listener *proxyproto.Listener) *ServicePod {
	pod := &ServicePod{
		Key: upstream.Key(),

		stopCh:     make(chan bool, 1),
		upstream:   upstream,
		httpServer: HttpServer,
		listener:   Listener,
	}

	return pod
}

func (pod *ServicePod) Invalid() {
	// do nothing for the present
}

func (pod *ServicePod) Run() {
	go func() {
		log.Infof("start runing pod now %s", pod.Key)
		err := pod.httpServer.Serve(pod.listener)
		if err != nil {
			log.Errorf("pod Run goroutine error  <%s>,  the error is [%s]", pod.Key, err)
		}
		log.Infof("end runing pod now %s", pod.Key)
	}()
}

func (pod *ServicePod) Dispose() {
	log.Info("disposing a service pod")
	//pod.httpServer.Handler.Stopping = true
}
