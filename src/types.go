package main

import (
	"net"
)

type Upstreams []Upstream

type Upstream struct {
	Port       uint16
	Ip         net.IP
	Name       string
	PathPrefix string
	Subdomain  string

	Metrics Metric
}

type Metric struct {
	NumOfRequests uint16
	Failed        uint16
	Success       uint16
	Timeout       uint16
}
