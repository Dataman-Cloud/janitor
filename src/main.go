package main

import (
	"net/http"

	"github.com/Dataman-Cloud/janitor/src/config"
	"github.com/Dataman-Cloud/janitor/src/proxy"

	log "github.com/Sirupsen/logrus"
	//"github.com/urfave/cli"
)

func main() {

	var stop chan struct{}
	cfg := config.DefaultConfig()
	httpProxy := proxy.NewHTTPProxy(&http.Transport{}, cfg.Proxy)

	log.Info("reverse proxy start to listen now")
	go listenAndServeHTTP(httpProxy)
	<-stop

}
