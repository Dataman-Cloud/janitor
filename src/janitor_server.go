package main

import (
	"github.com/Dataman-Cloud/janitor/src/config"
	"github.com/Dataman-Cloud/janitor/src/handler"
	"github.com/Dataman-Cloud/janitor/src/listener"
	"github.com/Dataman-Cloud/janitor/src/upstream"

	log "github.com/Sirupsen/logrus"
	"golang.org/x/net/context"
)

type JanitorServer struct {
	upstreamLoader  upstream.UpstreamLoader
	listenerManager *listener.Manager
	handerFactory   *handler.Factory

	ctx     context.Context
	config  config.Config
	running bool
}

func NewJanitorServer(Config config.Config) *JanitorServer {
	server := &JanitorServer{
		config: Config,
		ctx:    context.Background(),
	}
	return server
}

func (server *JanitorServer) Init() *JanitorServer {
	log.Info("JanitorServer Initialing ...")
	err := server.setupUpstreamLoader()
	if err != nil {
		log.Fatalf("Setup Upstream Loader Got err: %s", err)
	}

	err = server.setupListenerManager()
	if err != nil {
		log.Fatalf("Setup Listener Manager Got err: %s", err)
	}

	server.setupHandlerFactory()

	return server
}

func (server *JanitorServer) setupUpstreamLoader() error {
	log.Info("Upstream Loader started")
	upstreamLoader, err := upstream.InitAndStartUpstreamLoader(server.ctx, server.config)
	if err != nil {
		return err
	}
	server.ctx = context.WithValue(server.ctx, upstream.CONSUL_UPSTREAM_LOADER_KEY, upstreamLoader)
	server.upstreamLoader = upstreamLoader
	return nil
}

func (server *JanitorServer) setupListenerManager() error {
	log.Info("ListenerManager started")
	listenerManager, err := listener.InitManager(listener.MULTIPORT_LISTENER_MODE, server.config.Listener)
	if err != nil {
		return err
	}
	server.listenerManager = listenerManager
	server.ctx = context.WithValue(server.ctx, listener.LISTENER_MANAGER_KEY, listenerManager)
	return nil
}

func (server *JanitorServer) setupHandlerFactory() error {
	log.Info("Setup handler factory")
	handerFactory := handler.NewFactory(server.config.HttpHandler)
	server.ctx = context.WithValue(server.ctx, handler.HANDLER_FACTORY_KEY, handerFactory)
	server.handerFactory = handerFactory
	return nil
}

func (server *JanitorServer) Run() {
	for {
		<-server.upstreamLoader.ChangeNotify()
		log.Info("reloading listeners")
		//for _, upstream := range server.upstreamLoader.List() {
		//}
	}
}

func (server *JanitorServer) Shutdown() {}
