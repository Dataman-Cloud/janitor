package main

import (
	"fmt"
	"net/http"

	"github.com/Dataman-Cloud/janitor/src/config"
	"github.com/Dataman-Cloud/janitor/src/handler"
	"github.com/Dataman-Cloud/janitor/src/listener"
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

	handler := handler.NewHTTPProxy(&http.Transport{}, server.Config.HttpHandler)
	log.Info("JanitorServer Listening now")
	log.Info(handler)

	log.Info("Upstream Loader started")
	upstreamLoader, err := upstream.InitAndStart(server.Ctx, server.Config)
	if err != nil {
		panic(err)
	}
	server.Ctx = context.WithValue(server.Ctx, upstream.CONSUL_UPSTREAM_LOADER_KEY, upstreamLoader)

	log.Info("ListenerManager started")
	listenerManager, err := listener.InitManager(listener.SINGLE_LISTENER_MODE, server.Config.Listener)
	if err != nil {
		panic(err)
	}
	server.Ctx = context.WithValue(server.Ctx, listener.MANAGER_KEY, listenerManager)
	srv := &http.Server{
		Handler: handler,
	}
	srv.Serve(listenerManager.DefaultListener())

	fmt.Println(upstream.ConsulUpstreamLoaderFromContext(server.Ctx))
	fmt.Println(listener.ManagerFromContext(server.Ctx))
}

func (server *JanitorServer) Shutdown() {
}
