package main

import (
	"net/http"

	"github.com/Dataman-Cloud/janitor/src/config"
	"github.com/Dataman-Cloud/janitor/src/proxy"
	"github.com/Dataman-Cloud/janitor/src/upstream"

	log "github.com/Sirupsen/logrus"
	"golang.org/x/net/context"
)

type JanitorServer struct {
	Ctx     context.Context
	Config  config.Config
	Running bool
}

func NewJanitorServer(Config config.Config) *JanitorServer {
	server := &JanitorServer{
		Config: Config,
	}
	return server
}

func (server *JanitorServer) Start() {
	log.Info("JanitorServer Starting ...")

	httpProxy := proxy.NewHTTPProxy(&http.Transport{}, server.Config.Proxy)
	go listenAndServeHTTP(httpProxy, server.Config.Proxy)
	log.Info("JanitorServer Listening now")

	upstreamSource, err := upstream.InitAndStart(server.Config, server.Ctx)
	if err != nil {
		panic(err)
	}
	log.Debug(upstreamSource)
}

func (server *JanitorServer) Shutdown() {
}
