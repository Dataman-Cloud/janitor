package listener

import (
	"net"

	"github.com/Dataman-Cloud/janitor/src/config"

	log "github.com/Sirupsen/logrus"
	"github.com/armon/go-proxyproto"
	"golang.org/x/net/context"
)

const (
	SINGLE_LISTENER_MODE    = "single_port"
	MULTIPORT_LISTENER_MODE = "multi_port"
)

const (
	LISTENER_MANAGER_KEY = "listener_manager"
)

type ListenerKey struct {
	Ip   string
	Port string
}

type Manager struct {
	Mode      string
	Listeners map[ListenerKey]*proxyproto.Listener
	Config    config.Listener
}

func InitManager(mode string, Config config.Listener) (*Manager, error) {
	manager := &Manager{}
	manager.Mode = mode
	manager.Listeners = make(map[ListenerKey]*proxyproto.Listener)
	manager.Config = Config

	switch mode {
	case SINGLE_LISTENER_MODE:
		setupSingleListener(manager)
	case MULTIPORT_LISTENER_MODE:
		// Do nothing
	}

	return manager, nil
}

func ManagerFromContext(ctx context.Context) *Manager {
	manager := ctx.Value(LISTENER_MANAGER_KEY)
	return manager.(*Manager)
}

func (manager *Manager) Shutdown() {
	for _, listener := range manager.Listeners {
		listener.Close()
	}
}

func (manager *Manager) DefaultListenerKey() ListenerKey {
	return ListenerKey{Ip: manager.Config.IP.String(), Port: manager.Config.DefaultPort}
}

func (manager *Manager) DefaultListener() *proxyproto.Listener {
	return manager.Listeners[manager.DefaultListenerKey()]
}

func setupSingleListener(manager *Manager) {
	ln, err := net.Listen("tcp", net.JoinHostPort(manager.Config.IP.String(), manager.Config.DefaultPort))
	if err != nil {
		log.Fatal("[FATAL] ", err)
	}

	manager.Listeners[manager.DefaultListenerKey()] = &proxyproto.Listener{Listener: TcpKeepAliveListener{ln.(*net.TCPListener)}}
}

func (manager *Manager) FetchListener(key ListenerKey) *proxyproto.Listener {
	listener := manager.Listeners[key]
	if listener == nil {
		ln, err := net.Listen("tcp", net.JoinHostPort(key.Ip, key.Port))
		if err != nil {
			log.Fatal("[FATAL] ", err)
		}

		manager.Listeners[key] = &proxyproto.Listener{Listener: TcpKeepAliveListener{ln.(*net.TCPListener)}}
	}

	return manager.Listeners[key]
}

func (manager *Manager) Remove(key ListenerKey) {
	delete(manager.Listeners, key)
}

func (manager *Manager) ListeningPorts() []string {
	var ports []string
	for key, _ := range manager.Listeners {
		ports = append(ports, key.Port)
	}

	return ports
}
