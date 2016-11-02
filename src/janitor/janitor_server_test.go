package janitor

import (
	"github.com/Dataman-Cloud/janitor/src/config"
	"github.com/Dataman-Cloud/janitor/src/handler"
	"github.com/Dataman-Cloud/janitor/src/listener"
	//"github.com/Dataman-Cloud/janitor/src/service"
	"github.com/Dataman-Cloud/janitor/src/upstream"

	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewJanitor(t *testing.T) {
	janitorServer := NewJanitorServer(config.DefaultConfig())
	assert.NotNil(t, janitorServer)
}

func TestInit(t *testing.T) {
	janitorServer := NewJanitorServer(config.DefaultConfig())
	janitorServer.Init()
	assert.NotNil(t, janitorServer.listenerManager)
	assert.NotNil(t, janitorServer.upstreamLoader)
	assert.NotNil(t, janitorServer.handerFactory)
	assert.NotNil(t, janitorServer.serviceManager)
}

func TestSetupUpstreamLoader(t *testing.T) {
	janitorServer := NewJanitorServer(config.DefaultConfig())
	assert.Nil(t, janitorServer.ctx.Value(upstream.CONSUL_UPSTREAM_LOADER_KEY))
	janitorServer.setupUpstreamLoader()
	assert.NotNil(t, janitorServer.upstreamLoader)
	assert.NotNil(t, janitorServer.ctx.Value(upstream.CONSUL_UPSTREAM_LOADER_KEY))
}

func TestSetupListenerManager(t *testing.T) {
	janitorServer := NewJanitorServer(config.DefaultConfig())
	assert.Nil(t, janitorServer.ctx.Value(listener.LISTENER_MANAGER_KEY))
	janitorServer.setupListenerManager()
	assert.NotNil(t, janitorServer.listenerManager)
	assert.NotNil(t, janitorServer.ctx.Value(listener.LISTENER_MANAGER_KEY))
}

func TestSetupHandlerFactory(t *testing.T) {
	janitorServer := NewJanitorServer(config.DefaultConfig())
	assert.Nil(t, janitorServer.ctx.Value(handler.HANDLER_FACTORY_KEY))
	janitorServer.setupHandlerFactory()
	assert.NotNil(t, janitorServer.handerFactory)
	assert.NotNil(t, janitorServer.ctx.Value(handler.HANDLER_FACTORY_KEY))
}

func TestSetupServiceManager(t *testing.T) {
	janitorServer := NewJanitorServer(config.DefaultConfig())
	janitorServer.setupListenerManager()
	assert.Not(t, janitorServer.serviceManager)
}
