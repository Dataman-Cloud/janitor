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
	Listeners map[int]*proxyproto.Listener
	Config    config.Listener
}

func InitManager(mode string, Config config.Listener) (*Manager, error) {
	manager := &Manager{}
	manager.Mode = mode
	manager.Listeners = make(map[int]*proxyproto.Listener)
	manager.Config = Config

	switch mode {
	case SINGLE_LISTENER_MODE:
		setupSingleListener(manager)
	case MULTIPORT_LISTENER_MODE:
	}

	return manager, nil
}

func (manager *Manager) DefaultListener() *proxyproto.Listener {
	return manager.Listeners[manager.Config.DefaultPort]
}

func setupSingleListener(manager *Manager) {
	ln, err := net.Listen("tcp", net.JoinHostPort(manager.Config.IP.String(), fmt.Sprintf("%d", manager.Config.DefaultPort)))
	if err != nil {
		log.Fatal("[FATAL] ", err)
	}

	manager.Listeners[manager.Config.DefaultPort] = &proxyproto.Listener{Listener: TcpKeepAliveListener{ln.(*net.TCPListener)}}
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

func (manager *Manager) FetchListener(port int) *TcpKeepAliveListener {
	return &TcpKeepAliveListener{}
}

func (manager *Manager) ListeningPorts() []int {
	var ports []int
	for port, _ := range manager.Listeners {
		ports = append(ports, port)
	}

	return ports
}
