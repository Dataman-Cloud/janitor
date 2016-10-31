package handler

import (
	"testing"

	"github.com/Dataman-Cloud/janitor/src/config"
	"github.com/Dataman-Cloud/janitor/src/upstream"

	"github.com/stretchr/testify/assert"
)

func TestNewFactory(t *testing.T) {
	c := config.DefaultConfig()
	f := NewFactory(c.HttpHandler, c.Listener)
	assert.NotNil(t, f)
}

func TestHttpHandler(t *testing.T) {
	c := config.DefaultConfig()
	f := NewFactory(c.HttpHandler, c.Listener)
	httpHandler := f.HttpHandler(&upstream.Upstream{})
	assert.NotNil(t, httpHandler)
}
