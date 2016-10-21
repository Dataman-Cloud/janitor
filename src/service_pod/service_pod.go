package service_pod

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Dataman-Cloud/janitor/src/upstream"

	log "github.com/Sirupsen/logrus"
	"github.com/armon/go-proxyproto"
	consulApi "github.com/hashicorp/consul/api"
)

type ServicePod struct {
	Key      upstream.UpstreamKey
	upstream *upstream.Upstream

	Manager    *ServiceManager
	HttpServer *http.Server
	Listener   *proxyproto.Listener

	stopCh chan bool
}

func NewServicePod(upstream *upstream.Upstream) *ServicePod {
	pod := &ServicePod{
		Key: upstream.Key(),

		stopCh:   make(chan bool, 1),
		upstream: upstream,
	}

	pod.LogActivity(fmt.Sprintf("[INFO] preparing serving application %s at %s", upstream.ServiceName, upstream.Key().ToString()))

	return pod
}

func (pod *ServicePod) Invalid() {
	targets := make([]string, 0)
	for _, t := range pod.upstream.Targets {
		targets = append(targets, t.ToString())
	}

	pod.LogActivity(fmt.Sprintf("[INFO] changing application %s to targets [%s]", pod.upstream.ServiceName, strings.Join(targets, "  ")))
}

func (pod *ServicePod) LogActivity(activity string) {
	kv := pod.Manager.consulClient.KV()

	kvPair, _, err := kv.Get(fmt.Sprintf("lotus-%s", pod.upstream.ServiceName), nil)
	if err != nil {
		log.Errorf("kv get error %s", err)
	}

	existingValue := string(kvPair.Value)

	p := &consulApi.KVPair{Key: fmt.Sprintf("lotus-%s", pod.upstream.ServiceName),
		Value:   []byte(fmt.Sprintf("%s--%s", existingValue, activity)),
		Session: pod.Manager.sessionIDWithTTY,
	}
	_, err = kv.Put(p, nil)
	if err != nil {
		log.Errorf("persist service entries error %s", err)
	}
}

func (pod *ServicePod) Run() {
	go func() {
		log.Infof("start runing pod now %s", pod.Key)
		err := pod.HttpServer.Serve(pod.Listener)
		if err != nil {
			log.Errorf("pod run goroutine error  <%s>,  the error is [%s]", pod.Key, err)
		}
		log.Infof("end runing pod now %s", pod.Key)
	}()
}

func (pod *ServicePod) Dispose() {
	log.Info("disposing a service pod")
	pod.LogActivity(fmt.Sprintf("[INFO] stop application %s at %s", pod.upstream.ServiceName, pod.upstream.Key().ToString()))
}
