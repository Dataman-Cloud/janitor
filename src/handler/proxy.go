package handler

import (
	"net/http"
	"time"

	"github.com/Dataman-Cloud/janitor/src/config"
	"github.com/Dataman-Cloud/janitor/src/loadbalance"
	"github.com/Dataman-Cloud/janitor/src/upstream"
)

// httpProxy is a dynamic reverse proxy for HTTP and HTTPS protocols.
type httpProxy struct {
	tr           http.RoundTripper
	cfg          config.HttpHandler
	loadbalancer loadbalance.LoadBalancer
}

func NewHTTPProxy(tr http.RoundTripper, cfg config.HttpHandler, upstream *upstream.Upstream) http.Handler {
	loadbalancer := loadbalance.NewRoundRobinLoaderBalancer()
	loadbalancer.Seed(upstream.Targets)

	return &httpProxy{
		tr:           tr,
		cfg:          cfg,
		loadbalancer: loadbalancer,
	}
}

func (p *httpProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	targetEntry := p.loadbalancer.Next().Entry()
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
