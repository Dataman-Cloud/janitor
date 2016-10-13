package service_pod

import (
	"github.com/Dataman-Cloud/janitor/src/upstream"
)

type ServiceManager struct {
	ServicePods map[upstream.UpstreamKey]*ServicePod
}

func NewServiceManager() *ServiceManager {
	return &ServiceManager{
		ServicePods: make(map[upstream.UpstreamKey]*ServicePod),
	}
}
