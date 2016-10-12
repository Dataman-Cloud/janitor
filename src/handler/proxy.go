package handler

import (
	"net/http"
	"net/url"
	"time"

	"github.com/Dataman-Cloud/janitor/src/config"
)

// httpProxy is a dynamic reverse proxy for HTTP and HTTPS protocols.
type httpProxy struct {
	tr      http.RoundTripper
	cfg     config.HttpHandler
	baseUrl url.URL
}

func NewHTTPProxy(tr http.RoundTripper, cfg config.HttpHandler, url url.URL) http.Handler {
	return &httpProxy{
		tr:      tr,
		cfg:     cfg,
		baseUrl: url,
	}
}

func (p *httpProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var h http.Handler
	switch {
	case r.Header.Get("Upgrade") == "websocket":
		h = newRawProxy(p.baseUrl)

		// To use the filtered proxy use
		// h = newWSProxy(t.URL)

	case r.Header.Get("Accept") == "text/event-stream":
		// use the flush interval for SSE (server-sent events)
		// must be > 0s to be effective
		h = newHTTPProxy(p.baseUrl, p.tr, p.cfg.FlushInterval)

	default:
		h = newHTTPProxy(p.baseUrl, p.tr, time.Duration(0))
	}

	//start := time.Now()
	h.ServeHTTP(w, r)
}
