package handler

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Dataman-Cloud/janitor/src/config"
	"github.com/Dataman-Cloud/janitor/src/loadbalance"
	"github.com/Dataman-Cloud/janitor/src/upstream"
)

// httpProxy is a dynamic reverse proxy for HTTP and HTTPS protocols.
type httpProxy struct {
	tr             http.RoundTripper
	cfg            config.HttpHandler
	listenerConfig config.Listener
	loadbalancers  []*loadbalance.RoundRobinLoadBalancer
	upstreamLoader upstream.UpstreamLoader
}

func NewSwanHTTPProxy(tr http.RoundTripper, cfg config.HttpHandler, configListener config.Listener, upstreamLoader upstream.UpstreamLoader) http.Handler {
	return &httpProxy{
		tr:             tr,
		listenerConfig: configListener,
		cfg:            cfg,
		upstreamLoader: upstreamLoader,
	}
}

func NewHTTPProxy(tr http.RoundTripper, cfg config.HttpHandler, configListener config.Listener, upstream *upstream.Upstream) http.Handler {
	var loadbalancers []*loadbalance.RoundRobinLoadBalancer
	loadbalancer := loadbalance.NewRoundRobinLoadBalancer()
	loadbalancer.Seed(upstream)
	loadbalancers = append(loadbalancers, loadbalancer)

	return &httpProxy{
		tr:             tr,
		listenerConfig: configListener,
		cfg:            cfg,
		loadbalancers:  loadbalancers,
	}
}

func (p *httpProxy) UpdateLoadBalancers() {
	for {
		<-p.upstreamLoader.ChangeNotify()
		p.loadbalancers = p.loadbalancers[0:0]
		for _, upstream := range p.upstreamLoader.List() {
			loadbalancer := loadbalance.NewRoundRobinLoadBalancer()
			loadbalancer.Seed(upstream)
			p.loadbalancers = append(p.loadbalancers, loadbalancer)
		}
	}
}

func (p *httpProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("start handling request\n")
	var targetEntry *url.URL
	switch p.listenerConfig.Mode {
	case config.MULTIPORT_LISTENER_MODE:
		targetEntry = p.loadbalancers[0].Next().Entry()
	case config.SINGLE_LISTENER_MODE:
		go p.UpdateLoadBalancers()
		hostname := r.Header.Get("Host")
		domain := "dataman-inc.com"
		hostname = "http://nginx0051-01.defaultGroup.dataman-mesos.dataman-inc.com:80"
		if hostname == "" {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		u, err := url.Parse(hostname)
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		host, port, _ := net.SplitHostPort(u.Host)
		domainIndex := strings.Index(host, domain)
		fmt.Printf("scheme:%s\n", u.Scheme)
		fmt.Printf("host:%s\n", host)
		fmt.Printf("port:%s\n", port)
		namespace := host[0 : domainIndex-1]
		hostNamespaces := strings.Split(namespace, ".")
		if len(hostNamespaces) == 4 {
			serviceID := hostNamespaces[0]
			serviceName := strings.Join(hostNamespaces[1:len(hostNamespaces)], ".")
			fmt.Printf("serviceID:%s\n", serviceID)
			fmt.Printf("serviceName:%s\n", serviceName)
			upstream := p.upstreamLoader.Get(serviceName)
			if upstream != nil {
				target := upstream.GetTarget(serviceID)
				if target != nil {
					targetEntry = target.Entry()
				}
			}
		} else if len(hostNamespaces) == 3 {
			serviceName := strings.Join(hostNamespaces, ".")
			upstream := p.upstreamLoader.Get(serviceName)
			if upstream != nil {
				for _, lb := range p.loadbalancers {
					if upstream.Equal(lb.Upstream) {
						targetEntry = lb.Next().Entry()
						break
					}
				}
			}
		}

	}
	fmt.Printf("targetEntry:%s\n", targetEntry)
	if targetEntry == nil {
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	if err := p.AddHeaders(r); err != nil {
		http.Error(w, "cannot parse "+r.RemoteAddr, http.StatusInternalServerError)
		return
	}

	var h http.Handler
	switch {
	case r.Header.Get("Upgrade") == "websocket":
		h = newRawProxy(targetEntry)

		// To use the filtered proxy use
		// h = newWSProxy(t.URL)

	case r.Header.Get("Accept") == "text/event-stream":
		// use the flush interval for SSE (server-sent events)
		// must be > 0s to be effective
		h = newHTTPProxy(targetEntry, p.tr, p.cfg.FlushInterval)

	default:
		h = newHTTPProxy(targetEntry, p.tr, time.Duration(0))
	}

	//start := time.Now()
	h.ServeHTTP(w, r)
}

func (proxy *httpProxy) AddHeaders(r *http.Request) error {
	remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return errors.New("cannot parse " + r.RemoteAddr)
	}

	// set configurable ClientIPHeader
	// X-Real-Ip is set later and X-Forwarded-For is set
	// by the Go HTTP reverse proxy.
	if proxy.cfg.ClientIPHeader != "" &&
		proxy.cfg.ClientIPHeader != "X-Forwarded-For" &&
		proxy.cfg.ClientIPHeader != "X-Real-Ip" {
		r.Header.Set(proxy.cfg.ClientIPHeader, remoteIP)
	}

	if r.Header.Get("X-Real-Ip") == "" {
		r.Header.Set("X-Real-Ip", remoteIP)
	}

	// set the X-Forwarded-For header for websocket
	// connections since they aren't handled by the
	// http proxy which sets it.
	ws := r.Header.Get("Upgrade") == "websocket"
	if ws {
		r.Header.Set("X-Forwarded-For", remoteIP)
	}

	if r.Header.Get("X-Forwarded-Proto") == "" {
		switch {
		case ws && r.TLS != nil:
			r.Header.Set("X-Forwarded-Proto", "wss")
		case ws && r.TLS == nil:
			r.Header.Set("X-Forwarded-Proto", "ws")
		case r.TLS != nil:
			r.Header.Set("X-Forwarded-Proto", "https")
		default:
			r.Header.Set("X-Forwarded-Proto", "http")
		}
	}

	if r.Header.Get("X-Forwarded-Port") == "" {
		r.Header.Set("X-Forwarded-Port", localPort(r))
	}

	fwd := r.Header.Get("Forwarded")
	if fwd == "" {
		fwd = "for=" + remoteIP
		switch {
		case ws && r.TLS != nil:
			fwd += "; proto=wss"
		case ws && r.TLS == nil:
			fwd += "; proto=ws"
		case r.TLS != nil:
			fwd += "; proto=https"
		default:
			fwd += "; proto=http"
		}
	}
	if proxy.listenerConfig.IP.String() != "" {
		fwd += "; by=" + proxy.listenerConfig.IP.String()
	}
	r.Header.Set("Forwarded", fwd)

	//if cfg.TLSHeader != "" && r.TLS != nil {
	//r.Header.Set(cfg.TLSHeader, cfg.TLSHeaderValue)
	//}

	return nil
}
