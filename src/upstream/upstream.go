package upstream

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/Dataman-Cloud/janitor/src/config"
	log "github.com/Sirupsen/logrus"

	"golang.org/x/net/context"
)

type Target struct {
	Node           string
	Address        string
	ServiceName    string
	ServiceID      string
	ServiceAddress string
	ServicePort    string
	Upstream       *Upstream
}

func (t Target) Entry() *url.URL {
	fmt.Println(net.JoinHostPort(t.ServiceAddress, t.ServicePort))
	url, err := url.Parse(fmt.Sprintf("%s://%s", t.Upstream.FrontendProto, net.JoinHostPort(t.ServiceAddress, t.ServicePort)))
	if err != nil {
		log.Error("parse target.Address %s to url got err %s", t.Address, err)
	}

	return url
}

type Upstream struct {
	ServiceName   string `json:"ServiceName"`
	FrontendPort  string
	FrontendIp    string
	FrontendProto string

	Targets []*Target `json:"Target"`
}

type UpstreamLoader interface {
	Poll()
	List() []*Upstream
	Get(serviceName string) *Upstream
	Remove(upstream *Upstream)
}

func InitAndStart(ctx context.Context, Config config.Config) (UpstreamLoader, error) {
	var upstreamLoader UpstreamLoader
	var err error
	switch strings.ToLower(Config.Upstream.SourceType) {
	case "consul":
		upstreamLoader, err = InitConsulUpstreamLoader(Config.Upstream.ConsulAddr, Config.Upstream.PollInterval)
		if err != nil {
			return nil, err
		}
	}

	return upstreamLoader, nil
}
