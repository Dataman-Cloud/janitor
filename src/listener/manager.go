package listener

import (
	"fmt"
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
	MANAGER_KEY = "manager"
)

type Manager struct {
	Mode      string
	Listeners map[string]*proxyproto.Listener
	Config    config.Listener
}

func InitManager(mode string, Config config.Listener) (*Manager, error) {
	manager := &Manager{}
	manager.Mode = mode
	manager.Listeners = make(map[string]*proxyproto.Listener)
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
	manager := ctx.Value(MANAGER_KEY)
	return manager.(*Manager)
}

func (manager *Manager) Shutdown() {
	for _, listener := range manager.Listeners {
		listener.Close()
	}
}

func (manager *Manager) DefaultListener() *proxyproto.Listener {
	return manager.Listeners[manager.Config.DefaultPort]
}

func setupSingleListener(manager *Manager) {
	ln, err := net.Listen("tcp", net.JoinHostPort(manager.Config.IP.String(), manager.Config.DefaultPort))
	if err != nil {
		log.Fatal("[FATAL] ", err)
	}

	manager.Listeners[manager.Config.DefaultPort] = &proxyproto.Listener{Listener: TcpKeepAliveListener{ln.(*net.TCPListener)}}
}

func (manager *Manager) FetchListener(ip, port string) *proxyproto.Listener {
	// need sync access
	fmt.Println(ip)
	listener := manager.Listeners[port]
	if listener == nil {
		ln, err := net.Listen("tcp", net.JoinHostPort(ip, port))
		if err != nil {
			log.Fatal("[FATAL] ", err)
		}

		manager.Listeners[port] = &proxyproto.Listener{Listener: TcpKeepAliveListener{ln.(*net.TCPListener)}}
	}

	return manager.Listeners[port]
}

func (manager *Manager) ListeningPorts() []string {
	var ports []string
	for port, _ := range manager.Listeners {
		ports = append(ports, port)
	}

	return ports
}
