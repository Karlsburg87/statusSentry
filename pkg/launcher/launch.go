package launcher

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/karlsburg87/statusSentry/pkg/configuration"
	"github.com/karlsburg87/statusSentry/pkg/pinger"
	statuscheck "github.com/karlsburg87/statusSentry/pkg/statusCheck"
)

//---------------------------------------------
// Helper functions
//---------------------------------------------
var configLocation string

//ServerConfig is the config passed to confiRefreshServer to run a server according to envar configs
type ServerConfig struct {
	StatusChecker serverService
	Pinger        serverService
}

//ServerService specifies the channel and whether the service should be used in this runtime
type serverService struct {
	Channel chan *configuration.Configuration
	Active  bool
}

func init() {
	configLocation = os.Getenv("CONFIG_LOCATION")
	if configLocation == "" {
		configLocation = "" //TODO: default location to go here
	}
}

//Setup runs the Launcher for pinger and statusChecker unless a X_ONLY config envar has been set. Returns channels to running services and a cancelfunc. Non running services will have chan as nil
func Setup(conf *configuration.Configuration) (serverConfig ServerConfig, cancel context.CancelFunc) {
	//Setup cancellation context
	ctx, cancel := context.WithCancel(context.Background())
	//process any config specifying which parts of the application should run
	pingerOnly, err := strconv.ParseBool(os.Getenv("PINGER_ONLY"))
	if err != nil {
		pingerOnly = false
	}
	statusCheckOnly, err := strconv.ParseBool(os.Getenv("STATUS_CHECK_ONLY"))
	if err != nil {
		statusCheckOnly = false
	}
	if statusCheckOnly && pingerOnly {
		statusCheckOnly = false
		pingerOnly = false
	}
	//start status page checkers
	if !pingerOnly {
		serverConfig.StatusChecker.Active = true
		serverConfig.StatusChecker.Channel = make(chan *configuration.Configuration)
		statuscheck.Launch(ctx, serverConfig.StatusChecker.Channel)
		serverConfig.StatusChecker.Channel <- conf //send initial config data pointer to status check processors
	}
	//start pingers
	if !statusCheckOnly {
		serverConfig.Pinger.Active = true
		serverConfig.Pinger.Channel = make(chan *configuration.Configuration)
		pinger.Launch(ctx, serverConfig.Pinger.Channel)
		serverConfig.Pinger.Channel <- conf //send initial config data pointer to ping processors
	}
	return
}

//GetConfigurationFile fetches the configuration file and
func GetConfigurationFile(locationOfConfigFile string) (*configuration.Configuration, error) {
	//get config
	//return configuration.FetchConfig(http.DefaultClient, configLocation)

	return configuration.FetchTestConfig() //FIXME: change to FetchConfig when sending to production
}

//RefreshConfigServer is a server setup to listen exclusively for signal to
// reload the config data from CONFIG_LOCATION set as an environment variable
func RefreshConfigServer(config ServerConfig) *http.Server {
	refreshPort := 8099
	refreshMux := http.NewServeMux()
	refreshMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "This endpoint is not for public consumption")
	})
	refreshMux.HandleFunc("/config/refresh", func(w http.ResponseWriter, r *http.Request) {
		newConfig, err := GetConfigurationFile("")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"status": "error", "note": "Oops something went wrong.\nPlease contact the administrator", "error_msg": err.Error()})
			log.Fatalln(err)
		}
		if config.StatusChecker.Active {
			config.StatusChecker.Channel <- newConfig //statusPage checking functions
		}
		if config.Pinger.Active {
			config.Pinger.Channel <- newConfig //pinger functions
		}
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
