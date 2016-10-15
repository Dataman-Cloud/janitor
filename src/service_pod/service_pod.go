package service_pod

import (
	"fmt"
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
		fmt.Println("start runing pod")
		err := pod.httpServer.Serve(pod.listener)
		if err != nil {
			log.Error("pod Run goroutine error  <%s>,  the error is [%s]", pod.Key, err)
		}
	}()
}

func (pod *ServicePod) Dispose() {
	err := pod.listener.Close()
	if err != nil {
		log.Error("dispose error close listener for pod <%s>,  the error is [%s]", pod.Key, err)
	}
}
