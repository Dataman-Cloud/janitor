package handler

import (
	"net/http"

	"github.com/Dataman-Cloud/janitor/src/config"
)

type Factory struct {
	HttpHandlerCfg config.HttpHandler
}

func NewFactory(cfg config.HttpHandler) *Factory {
	return &Factory{HttpHandlerCfg: cfg}
}

func (factory *Factory) HttpHandler() http.Handler {
	return NewHTTPProxy(&http.Transport{}, factory.HttpHandlerCfg)
}
