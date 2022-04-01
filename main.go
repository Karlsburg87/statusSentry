package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/karlsburg87/Status/pkg/configuration"
	"github.com/karlsburg87/Status/pkg/pinger"
	statuscheck "github.com/karlsburg87/Status/pkg/statusCheck"
)

func main() {
	//get the initial configuration files with which status pages to check and URLs to ping
	conf, err := getConfigurationFile(configLocation)
	if err != nil {
		log.Fatalln(err)
	}

	//Setup cancellation context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() //TODO: work this properly for autoheal and graceful shutdown

	//start status page checkers
	scConf := make(chan *configuration.Configuration)
	statuscheck.Launch(ctx, scConf)
	scConf <- conf //send initial config data pointer to status check processors

	//start pingers
	pConf := make(chan *configuration.Configuration)
	pinger.Launch(ctx, pConf)
	pConf <- conf //send initial config data pointer to ping processors

	//check for updates of config file with call to endpoint
	if err := configRefreshServer(scConf, pConf).ListenAndServe(); err != nil {
		log.Fatalln(err)
	}

}

//---------------------------------------------
// Helper functions
//---------------------------------------------
var configLocation string

func init() {
	configLocation = os.Getenv("CONFIG_LOCATION")
	if configLocation == "" {
		configLocation = "" //TODO: default location to go here
	}
}

//getConfigurationFile fetches the configuration file and
func getConfigurationFile(locationOfConfigFile string) (*configuration.Configuration, error) {
	//get config
	//return configuration.FetchConfig(http.DefaultClient, configLocation)

	return configuration.FetchTestConfig() //change to FetchConfig when sending to production
}

//configRefreshServer is a server setup to listen exclusively for signal to
// reload the config data from CONFIG_LOCATION set as an envar
func configRefreshServer(scConf, pConf chan *configuration.Configuration) *http.Server {
	refreshPort := 8099
	refreshMux := http.NewServeMux()
	refreshMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "This endpoint is not for public consumption")
	})
	refreshMux.HandleFunc("/config/refresh", func(w http.ResponseWriter, r *http.Request) {
		newConfig, err := getConfigurationFile("")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"status": "error", "note": "Oops something went wrong.\nPlease contact the administrator", "error_msg": err.Error()})
			log.Fatalln(err)
		}
		scConf <- newConfig //statusPage checking functions
		pConf <- newConfig  //pinger functions
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	})
	refreshServer := &http.Server{
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       20 * time.Second,
		Handler:           http.NewServeMux(),
		Addr:              fmt.Sprintf(":%d", refreshPort),
	}
	log.Printf("Config refresh server created on port %d\n", refreshPort)
	return refreshServer
}
