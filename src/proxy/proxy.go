package proxy

import (
	"net/http"
	"time"

	"github.com/Dataman-Cloud/janitor/src/config"
)

// httpProxy is a dynamic reverse proxy for HTTP and HTTPS protocols.
type httpProxy struct {
	tr  http.RoundTripper
	cfg config.Proxy
}

func NewHTTPProxy(tr http.RoundTripper, cfg config.Proxy) http.Handler {
	return &httpProxy{
		tr:  tr,
		cfg: cfg,
	}
}

func (p *httpProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t := target(r)
	if t == nil {
		w.WriteHeader(p.cfg.NoRouteStatus)
		return
	}

	if err := addHeaders(r, p.cfg); err != nil {
		http.Error(w, "cannot parse "+r.RemoteAddr, http.StatusInternalServerError)
		return
	}

	var h http.Handler
	switch {
	case r.Header.Get("Upgrade") == "websocket":
		h = newRawProxy(t.URL)

		// To use the filtered proxy use
		// h = newWSProxy(t.URL)

	case r.Header.Get("Accept") == "text/event-stream":
		// use the flush interval for SSE (server-sent events)
		// must be > 0s to be effective
		h = newHTTPProxy(t.URL, p.tr, p.cfg.FlushInterval)

	default:
		h = newHTTPProxy(t.URL, p.tr, time.Duration(0))
	}

	//start := time.Now()
	h.ServeHTTP(w, r)
}
