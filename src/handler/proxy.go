package handler

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Dataman-Cloud/janitor/src/config"
	"github.com/Dataman-Cloud/janitor/src/upstream"

	log "github.com/Sirupsen/logrus"
)

// httpProxy is a dynamic reverse proxy for HTTP and HTTPS protocols.
type httpProxy struct {
	tr             http.RoundTripper
	cfg            config.HttpHandler
	listenerConfig config.Listener
	upstreamLoader upstream.UpstreamLoader
	targetEntry    *url.URL
}

func NewHTTPProxy(tr http.RoundTripper, cfg config.HttpHandler, configListener config.Listener, upstream *upstream.Upstream, upstreamLoader upstream.UpstreamLoader) http.Handler {
	var targetEntry *url.URL
	// in MULTI_LISTENER_MODE
	if upstream != nil {
		targetEntry = upstream.NextTargetEntry()
	}

	return &httpProxy{
		tr:             tr,
		listenerConfig: configListener,
		cfg:            cfg,
		upstreamLoader: upstreamLoader,
		targetEntry:    targetEntry,
	}
}

func (p *httpProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if p.listenerConfig.Mode == config.SINGLE_LISTENER_MODE {
		hostname := r.Header.Get("Host")
		hostname = "http://nginx0051-01.defaultGroup.dataman-mesos.dataman-inc.com:80"
		if hostname == "" {
			log.Debug("hostname is null")
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		u, err := url.Parse(hostname)
		if err != nil {
			log.Debugf("fail to parse hostname[%s]", hostname)
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		host, _, _ := net.SplitHostPort(u.Host)
		log.Debugf("Request from host:%s", host)

		// get targetEntry based on hostname
		domainIndex := strings.Index(host, p.cfg.Domain)
		namespace := host[0 : domainIndex-1]
		hostNamespaces := strings.Split(namespace, ".")
		if len(hostNamespaces) == 4 {
			// host is targeted at task level
			serviceID := hostNamespaces[0]
			serviceName := strings.Join(hostNamespaces[1:len(hostNamespaces)], ".")
			upstream := p.upstreamLoader.Get(serviceName)
			if upstream != nil {
				target := upstream.GetTarget(serviceID)
				if target != nil {
					p.targetEntry = target.Entry()
				}
			}
		} else if len(hostNamespaces) == 3 {
			// host is targeted at app level
			serviceName := strings.Join(hostNamespaces, ".")
			upstream := p.upstreamLoader.Get(serviceName)
			if upstream != nil {
				p.targetEntry = upstream.NextTargetEntry()
			}
		}

	}
	log.Debugf("targetEntry [%s] is found", p.targetEntry)
	if p.targetEntry == nil {
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
		h = newRawProxy(p.targetEntry)

		// To use the filtered proxy use
		// h = newWSProxy(t.URL)

	case r.Header.Get("Accept") == "text/event-stream":
		// use the flush interval for SSE (server-sent events)
		// must be > 0s to be effective
		h = newHTTPProxy(p.targetEntry, p.tr, p.cfg.FlushInterval)

	default:
		h = newHTTPProxy(p.targetEntry, p.tr, time.Duration(0))
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
