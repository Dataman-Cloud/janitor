package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/Dataman-Cloud/janitor/src/config"
	"github.com/Dataman-Cloud/janitor/src/janitor"

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
