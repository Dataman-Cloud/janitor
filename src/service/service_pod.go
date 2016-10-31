package service

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Dataman-Cloud/janitor/src/upstream"

	log "github.com/Sirupsen/logrus"
	"github.com/armon/go-proxyproto"
	consulApi "github.com/hashicorp/consul/api"
)

const (
	SESSION_RENEW_INTERVAL = time.Second * 50
)

type ServicePod struct {
	Key upstream.UpstreamKey

	Manager    *ServiceManager
	HttpServer *http.Server
	Listener   *proxyproto.Listener

	upstream           *upstream.Upstream
	sessionIDWithTTY   string
	sessionRenewTicker *time.Ticker
	stopCh             chan bool
	lock               sync.Mutex
}

func NewServicePod(upstream *upstream.Upstream, manager *ServiceManager) (*ServicePod, error) {
	pod := &ServicePod{
		Key: upstream.Key(),

		stopCh:   make(chan bool, 1),
		upstream: upstream,
		Manager:  manager,
	}

	pod.sessionRenewTicker = time.NewTicker(SESSION_RENEW_INTERVAL)
	err := pod.setupTTLSession()
	if err != nil {
		return nil, err
	}

	pod.keepSessionAlive()
	pod.LogActivity(fmt.Sprintf("[INFO] preparing serving application %s at %s", upstream.ServiceName, upstream.Key().ToString()))

	return pod, nil
}

func (pod *ServicePod) setupTTLSession() error {
	var err error
	pod.sessionIDWithTTY, _, err = pod.Manager.consulClient.Session().Create(
		&consulApi.SessionEntry{
			Behavior: "delete",
			TTL:      "60s",
		}, nil,
	)
	if err != nil {
		log.Errorf("create a session error: %s", err)
		return err
	}
	return nil
}

func (pod *ServicePod) keepSessionAlive() {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Error("KeepSessionAlive got error: %s", err)
				pod.keepSessionAlive()
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
	pod.lock.Lock()
	defer pod.lock.Unlock()

	targets := make([]string, 0)
	for _, t := range pod.upstream.Targets {
		targets = append(targets, t.ToString())
	}

	pod.LogActivity(fmt.Sprintf("[INFO] changing application %s to targets [%s]", pod.upstream.ServiceName, strings.Join(targets, "  ")))
}

func (pod *ServicePod) LogActivity(activity string) {
	go func() { // make this run in other thread
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
			if len(existingValue) > 100000 {
				existingValues := strings.Split(string(existingValue), "--")
				existingValuesLen := len(existingValues)
				existingValue = strings.Join(existingValues[existingValuesLen-20:existingValuesLen], "--")
			}
		}

		log.Debugf(existingValue)
		p := &consulApi.KVPair{Key: fmt.Sprintf("%s/%s", SERVICE_ACTIVITIES_PREFIX, pod.upstream.ServiceName),
			Value:   []byte(fmt.Sprintf("%s--%s", existingValue, activity)),
			Session: pod.sessionIDWithTTY,
		}
		_, _, err = kv.Acquire(p, nil)
		if err != nil {
			log.Errorf("persist service entries error %s", err)
		}
	}()
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
	log.Infof("disposing a service pod")
	pod.RemovePodEntry()
	pod.LogActivity(fmt.Sprintf("[INFO] stop application %s at %s", pod.upstream.ServiceName, pod.upstream.Key().ToString()))
	pod.stopCh <- true
}

func (pod *ServicePod) RenewPodEntries() {
	// use consulClient For short, UGLY
	go func() {
		kv := pod.Manager.consulClient.KV()

		p := &consulApi.KVPair{Key: fmt.Sprintf("%s/%s/%s/%s", SERVICE_ENTRIES_PREFIX, pod.upstream.ServiceName, pod.Key.Ip, pod.Key.Port),
			Value:   []byte(fmt.Sprintf("%s://%s:%s", pod.Key.Proto, pod.Key.Ip, pod.Key.Port)),
			Session: pod.sessionIDWithTTY,
		}
		_, _, err := kv.Acquire(p, nil)
		if err != nil {
			log.Errorf("persist service entries error %s", err)
		}
	}()
}

func (pod *ServicePod) RemovePodEntry() {
	go func() {
		kv := pod.Manager.consulClient.KV()

		_, err := kv.Delete(fmt.Sprintf("%s/%s/%s/%s", SERVICE_ENTRIES_PREFIX, pod.upstream.ServiceName, pod.Key.Ip, pod.Key.Port), nil)
		if err != nil {
			log.Errorf("delete service entries error %s", err)
		}
	}()
}
