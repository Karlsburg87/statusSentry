package main

import (
	"log"
	"os"

	"github.com/karlsburg87/statusSentry/launcher"
)

func main() {
	//get the initial configuration files with which status pages to check and URLs to ping
	conf, err := launcher.GetConfigurationFile(os.Getenv("CONFIG_LOCATION"))
	if err != nil {
		log.Fatalln(err)
	}

	serverConfig, cancel := launcher.Setup(conf)
	defer cancel() //TODO: work this properly for autoheal and graceful shutdown

	//check for updates of config file with call to endpoint
	if err := launcher.RefreshConfigServer(serverConfig).ListenAndServe(); err != nil {
		log.Fatalln(err)
	}
}
