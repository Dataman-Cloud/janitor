package config

import (
	"net/http"
	"time"
)

func DefaultConfig() Config {
	config := Config{
		Proxy: Proxy{
			Addr: "localhost:3456",
		},
		Upstream: Upstream{
			SourceType:   "consul",
			ConsulAddr:   "localhost:8500",
			PollInterval: time.Second * 1,
		},
	}

	return config
}

type Config struct {
	Proxy    Proxy
	Upstream Upstream
}

type Proxy struct {
	Addr string

	Strategy              string
	Matcher               string
	NoRouteStatus         int
	MaxConn               int
	ShutdownWait          time.Duration
	DialTimeout           time.Duration
	ResponseHeaderTimeout time.Duration
	KeepAliveTimeout      time.Duration
	ReadTimeout           time.Duration
	WriteTimeout          time.Duration
	FlushInterval         time.Duration
	LocalIP               string
	ClientIPHeader        string
	TLSHeader             string
	TLSHeaderValue        string
}

type CertSource struct {
	Name         string
	Type         string
	CertPath     string
	KeyPath      string
	ClientCAPath string
	CAUpgradeCN  string
	Refresh      time.Duration
	Header       http.Header
}

type Upstream struct {
	SourceType   string // one of consul, file or somthing else
	ConsulAddr   string
	PollInterval time.Duration
}
