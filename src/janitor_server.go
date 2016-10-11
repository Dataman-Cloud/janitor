package main

import (
	"fmt"
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
		Ctx:    context.Background(),
	}
	return server
}

func (server *JanitorServer) Start() {
	log.Info("JanitorServer Starting ...")

	httpProxy := proxy.NewHTTPProxy(&http.Transport{}, server.Config.Proxy)
	go proxy.ListenAndServeHTTP(httpProxy, server.Config.Proxy)
	log.Info("JanitorServer Listening now")

	upstreamLoader, err := upstream.InitAndStart(server.Config, server.Ctx)
	if err != nil {
		panic(err)
	}
	server.Ctx = context.WithValue(server.Ctx, upstream.CONSUL_UPSTREAM_LOADER_KEY, upstreamLoader)

	fmt.Println(upstream.ConsulUpstreamLoaderFromContext(server.Ctx))
}

func (server *JanitorServer) Shutdown() {
}
