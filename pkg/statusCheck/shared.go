package statuscheck

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/karlsburg87/statusSentry/pkg/configuration"
)

func newServer(mux *http.ServeMux, port int) http.Server {
	return http.Server{
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       20 * time.Second,
		Handler:           mux,
		Addr:              fmt.Sprintf(":%d", port),
	}
}

func newClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			IdleConnTimeout:       30 * time.Second,
			TLSHandshakeTimeout:   15 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
			ExpectContinueTimeout: 10 * time.Second, //Not sending large payloads in the body so 10 secs should be plenty
			DisableKeepAlives:     false,            //keep alives active for efficiency
			MaxIdleConns:          30,
			MaxConnsPerHost:       0,
			MaxIdleConnsPerHost:   0,
		},
		Timeout: 30 * time.Second,
	}
}

//batchConfig batches the config into service type groups to be handled by the respective functions
func batchConfig(config *configuration.Configuration) (map[configuration.ServiceType]configuration.Configuration, error) {
	//get config in map form to do channel specific checks
	streamMap := make(map[configuration.ServiceType]configuration.Configuration)
	for _, con := range *config {
		serviceType, _ := con.ParseServiceInfo()
		if _, ok := streamMap[serviceType]; !ok {
			streamMap[serviceType] = make(configuration.Configuration, 0)
		}
		streamMap[serviceType] = append(streamMap[serviceType], con)
	}
	return streamMap, nil
}

//validator is the communication message template sent by PUSH service types to ensure what they
// are receiving is a valid update in config
type validator struct {
	valid        chan configuration.Config //valid is the response channel with a bool signalling true if valid service update
	serviceName  string
	emailAddress string
}

//operator orchestrates configMap updating and status Pulls to avoid data races in configmap updates
//
//Effectively the central function of the package
func operator(dir directory) {
	var configMap map[configuration.ServiceType]configuration.Configuration
	var err error

	//How often to check pull updates
	tckr := time.NewTicker(60 * time.Second)

	for {
		select {
		case <-dir.cancel:
			//close all channels in dir for PUSH to allow goroutines to complete
			dir.closePush()
			return

		case conf := <-dir.configChan: //incoming call to update the configmap
			configMap, err = batchConfig(conf)
			if err != nil {
				log.Panicln(err)
			}

		case toValidate := <-dir.validators.webhook: //validation of incoming webhook messages - returns the relevant config
			for _, item := range configMap[configuration.ServiceWebhook] {
				if item.ServiceName == toValidate.serviceName {
					toValidate.valid <- item
					continue
				}
			}
			toValidate.valid <- configuration.Config{} //send default empty struct if not valid

		case toValidate := <-dir.validators.email:
			for _, item := range configMap[configuration.ServiceEmail] {
				if _, senderAddress := item.ParseServiceInfo(); senderAddress == toValidate.emailAddress {
					toValidate.valid <- item
				}
			}
			toValidate.valid <- configuration.Config{}

		case <-tckr.C: //cron to run through the PULL service type update operations
			for channelType, entries := range configMap {
				for _, entry := range entries {
					switch channelType {
					case configuration.ServiceRSS:
						dir.rssChan <- entry
					case configuration.ServiceTwitter:
						if twitterBearerToken == "" {
							continue
						}
						dir.twitterChan <- entry
					}
				}
			}
		}
	}
}
