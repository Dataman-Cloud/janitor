package listener

import (
	"github.com/Dataman-Cloud/janitor/src/config"

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
	Mode      int
	Listeners map[int]TcpKeepAliveListener
}

func InitManager(mode string, Config config.Listener) (*Manager, error) {
	return &Manager{}, nil

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
