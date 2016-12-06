package handler

import (
	"net/http"

	"github.com/Dataman-Cloud/janitor/src/config"
	"github.com/Dataman-Cloud/janitor/src/upstream"
)

const HANDLER_FACTORY_KEY = "handler_factory"

type Factory struct {
	HttpHandlerCfg config.HttpHandler
	ListenerCfg    config.Listener
	UpstreamLoader upstream.UpstreamLoader
}

func NewFactory(cfg config.HttpHandler, listenerCfg config.Listener) *Factory {
	return &Factory{
		HttpHandlerCfg: cfg,
		ListenerCfg:    listenerCfg,
	}
}

func (factory *Factory) HttpHandler(upstream *upstream.Upstream) http.Handler {
	var proxy http.Handler
	switch factory.ListenerCfg.Mode {
	case config.MULTIPORT_LISTENER_MODE:
		proxy = NewHTTPProxy(&http.Transport{}, factory.HttpHandlerCfg, factory.ListenerCfg, upstream)
	case config.SINGLE_LISTENER_MODE:
		proxy = NewSwanHTTPProxy(&http.Transport{}, factory.HttpHandlerCfg, factory.ListenerCfg, factory.UpstreamLoader)
	}
	return proxy
}
