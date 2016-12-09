package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Dataman-Cloud/janitor/src/config"
	"github.com/Dataman-Cloud/janitor/src/janitor"
	"github.com/Dataman-Cloud/janitor/src/upstream"

	log "github.com/Sirupsen/logrus"
	//"github.com/urfave/cli"
)

var stopWait chan bool
var cleanFuncs []func()

func SetupLogger() {
	// Log as JSON instead of the default ASCII formatter.
	log.SetFormatter(&log.TextFormatter{})

	// Output to stderr instead of stdout, could also be a file.
	log.SetOutput(os.Stderr)

	// Only log the warning severity or above.
	log.SetLevel(log.DebugLevel)
}

func LoadConfig() config.Config {
	return config.DefaultConfig()
}

func TuneGolangProcess() {}

func RegisterSignalHandler() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		for _, fn := range cleanFuncs {
			fn()
		}

		stopWait <- true
	}()
}

//This is for swan
func main() {
	janitorConfig := LoadConfig()
	janitorConfig.Listener.Mode = config.SINGLE_LISTENER_MODE
	janitorConfig.Listener.DefaultPort = "9998"
	janitorConfig.HttpHandler.Domain = "dataman-inc.com"
	janitorUpstream := config.Upstream{
		SourceType: "swan",
	}
	janitorConfig.Upstream = janitorUpstream

	TuneGolangProcess()
	SetupLogger()

	server := janitor.NewJanitorServer(janitorConfig)
	go server.Init().Run()
	//cleanFuncs = append(cleanFuncs, func() {
	//	server.Shutdown()
	//})

	//<-stopWait
	//register signal handler

	ticker := time.NewTicker(time.Second * 30)
	for {
		<-ticker.C
		log.Debug("sending targetChangeEvent")
		targetChangeEvents := []*upstream.TargetChangeEvent{
			{
				Change:     "add",
				TargetName: "0.nginx0051-01.defaultGroup.dataman-mesos",
				TargetIP:   "192.168.1.162",
				TargetPort: "80",
			},
			{
				Change:     "add",
				TargetName: "1.nginx0051-01.defaultGroup.dataman-mesos",
				TargetIP:   "192.168.1.163",
				TargetPort: "80",
			},
			//{
			//	Change:     "delete",
			//	TargetName:      "0.nginx0051-01.defaultGroup.dataman-mesos",
			//	TargetIP: "192.168.1.162",
			//	TargetPort:     "80",
			//},
		}
		for _, targetChangeEvent := range targetChangeEvents {
			server.SwanEventChan() <- targetChangeEvent
		}
	}
}

/*
This is for borg
func main() {
	config := LoadConfig()

	TuneGolangProcess()
	SetupLogger()

	server := janitor.NewJanitorServer(config)
	server.Init().Run()
	cleanFuncs = append(cleanFuncs, func() {
		server.Shutdown()
	})

	//<-stopWait
	//register signal handler
}
*/
