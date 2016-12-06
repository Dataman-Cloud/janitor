package janitor

import (
	"github.com/Dataman-Cloud/janitor/src/config"
	"github.com/Dataman-Cloud/janitor/src/handler"
	"github.com/Dataman-Cloud/janitor/src/listener"
	"github.com/Dataman-Cloud/janitor/src/service"
	"github.com/Dataman-Cloud/janitor/src/upstream"

	log "github.com/Sirupsen/logrus"
	"golang.org/x/net/context"
)

type JanitorServer struct {
	upstreamLoader  upstream.UpstreamLoader
	listenerManager *listener.Manager
	handerFactory   *handler.Factory
	serviceManager  *service.ServiceManager

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
	log.Info("Janitor Server Initialing")
	err := server.setupUpstreamLoader()
	if err != nil {
		log.Fatalf("Setup Upstream Loader Got err: %s", err)
	}

	err = server.setupListenerManager()
	if err != nil {
		log.Fatalf("Setup Listener Manager Got err: %s", err)
	}

	server.setupHandlerFactory()
	server.setupServiceManager()

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
	log.Info("Listener Manager started")
	listenerManager, err := listener.InitManager(server.config.Listener)
	if err != nil {
		return err
	}
	server.listenerManager = listenerManager
	server.ctx = context.WithValue(server.ctx, listener.LISTENER_MANAGER_KEY, listenerManager)
	return nil
}

func (server *JanitorServer) setupHandlerFactory() error {
	log.Info("Setup handler factory")
	handerFactory := handler.NewFactory(server.config.HttpHandler, server.config.Listener)
	server.ctx = context.WithValue(server.ctx, handler.HANDLER_FACTORY_KEY, handerFactory)
	server.handerFactory = handerFactory
	return nil
}

func (server *JanitorServer) setupServiceManager() error {
	log.Info("Setup service manager")
	server.serviceManager = service.NewServiceManager(server.ctx)
	return nil
}

func (server *JanitorServer) Run() {
	for {
		<-server.upstreamLoader.ChangeNotify()
		for _, u := range server.upstreamLoader.List() {
			switch u.State.State() {
			case upstream.STATE_NEW:
				log.Infof("create new service pod: %s", u.Key())
				pod, err := server.serviceManager.ForkOrFetchNewServicePod(u)
				if err != nil {
					log.Infof("fail to create a service pod: %s", err.Error())
					continue
				}
				pod.Run()

			case upstream.STATE_CHANGED:
				log.Infof("update existing service pod: %s", u.Key())
				log.Infof("current upstream has %d targets", len(u.Targets))
				pod, err := server.serviceManager.ForkOrFetchNewServicePod(u)
				if err != nil {
					log.Errorf("failed to found pod %s", u.Key().ToString())
				} else {
					pod.Invalid()
				}
			}

			u.SetState(upstream.STATE_LISTENING)
		}

		for _, u := range server.upstreamLoader.List() {
			if u.StaleMark {
				log.Infof("remove unused service pod: %s", u.Key())
				server.serviceManager.KillServicePod(u)
			}
		}
	}
}

func (server *JanitorServer) Shutdown() {}

func (server *JanitorServer) PortsOccupied() []string {
	return server.serviceManager.PortsOccupied()
}

func (server *JanitorServer) ClusterAddressList(prefix string) ([]string, error) {
	return server.serviceManager.ClusterAddressList(prefix)
}

func (server *JanitorServer) ServiceActvities(serviceName string) ([]string, error) {
	return server.serviceManager.ServiceActvities(serviceName)
}
