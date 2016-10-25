package service_pod

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Dataman-Cloud/janitor/src/upstream"

	log "github.com/Sirupsen/logrus"
	"github.com/armon/go-proxyproto"
	consulApi "github.com/hashicorp/consul/api"
)

const (
	SESSION_RENEW_INTERVAL = time.Second * 2
)

type ServicePod struct {
	Key      upstream.UpstreamKey
	upstream *upstream.Upstream

	Manager    *ServiceManager
	HttpServer *http.Server
	Listener   *proxyproto.Listener

	sessionIDWithTTY   string
	sessionRenewTicker *time.Ticker
	stopCh             chan bool
}

func NewServicePod(upstream *upstream.Upstream, manager *ServiceManager) *ServicePod {
	pod := &ServicePod{
		Key: upstream.Key(),

		stopCh:   make(chan bool, 1),
		upstream: upstream,
		Manager:  manager,
	}

	pod.LogActivity(fmt.Sprintf("[INFO] preparing serving application %s at %s", upstream.ServiceName, upstream.Key().ToString()))

	pod.sessionRenewTicker = time.NewTicker(SESSION_RENEW_INTERVAL)
	var err error
	pod.sessionIDWithTTY, _, err = pod.Manager.consulClient.Session().Create(
		&consulApi.SessionEntry{
			Behavior: "delete",
			TTL:      "10s",
		}, nil,
	)
	if err != nil {
		log.Errorf("create a session error: %s", err)
	}

	pod.KeepSessionAlive()

	return pod
}

func (pod *ServicePod) KeepSessionAlive() {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Error("KeepSessionAlive got error: %s", err)
				pod.KeepSessionAlive()
			}
		}()

		for {
			select {
			case <-pod.sessionRenewTicker.C:
				_, _, err := pod.Manager.consulClient.Session().Renew(pod.sessionIDWithTTY, nil)
				if err != nil {
					log.Errorf("renew a session error: %s", err)
				}
			case <-pod.stopCh:
				log.Info("exiting KeepSessionAlive goroutine")
				return
			}
		}
	}()
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
	kvPair, _, err := kv.Get(fmt.Sprintf("%s/%s", SERVICE_ACTIVITIES_PREFIX, pod.upstream.ServiceName), nil)
	if err != nil {
		log.Errorf("kv get error %s", err)
	}

	var existingValue string
	if kvPair == nil {
		existingValue = ""
	} else {
		existingValue = string(kvPair.Value)
	}

	p := &consulApi.KVPair{Key: fmt.Sprintf("%s/%s", SERVICE_ACTIVITIES_PREFIX, pod.upstream.ServiceName),
		Value:   []byte(fmt.Sprintf("%s--%s", existingValue, activity)),
		Session: pod.sessionIDWithTTY,
	}
	_, err = kv.Put(p, nil)
	if err != nil {
		log.Errorf("persist service entries error %s", err)
	}
}

func (pod *ServicePod) Run() {
	pod.RenewPodEntries()
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
	pod.RenewPodEntries()
	pod.LogActivity(fmt.Sprintf("[INFO] stop application %s at %s", pod.upstream.ServiceName, pod.upstream.Key().ToString()))
	pod.stopCh <- true
}

func (pod *ServicePod) RenewPodEntries() {
	// use consulClient For short, UGLY
	kv := pod.Manager.consulClient.KV()

	p := &consulApi.KVPair{Key: fmt.Sprintf("%s/%s/%s", SERVICE_ENTRIES_PREFIX, pod.upstream.ServiceName, pod.Key.Ip),
		Value:   []byte(fmt.Sprintf("%s://%s:%s", pod.Key.Proto, pod.Key.Ip, pod.Key.Port)),
		Session: pod.sessionIDWithTTY,
	}
	_, err := kv.Put(p, nil)
	if err != nil {
		log.Errorf("persist service entries error %s", err)
	}
}