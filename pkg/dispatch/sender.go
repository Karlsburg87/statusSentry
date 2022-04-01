package dispatch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/karlsburg87/Status/pkg/configuration"
)

//retry is the key for the Send func which allows failed messages to be attempted again in future
type retry struct {
	lastAttempt    time.Time     //lastAttempt is the time of the most recent failed attempt to send
	nextTime       time.Time     //nextTime is the  time after which another send can be attempted
	currentBackoff time.Duration //currentBackoff is the time used in the calculation for the set attemptTime
}

//again is a directory of transports that have failed to send and need to be retried
type again map[retry]configuration.Transporter

//newDispatcher outputs a function which attempts a send of the Transport t to the destination URL and updates retryBacklog if errors occur
//
//the output function does not automatically error on a http post failure as the failed transport is queued for reattempt
func (again again) newDispatcher(client *http.Client, destination string) func(configuration.Transporter, retry) (map[string]interface{}, error) {
	return func(t configuration.Transporter, rt retry) (map[string]interface{}, error) {
		now := time.Now()
		payload, err := t.ToJSON()
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequest(http.MethodPost, destination, bytes.NewReader(payload))
		if err != nil {
			return nil, err
		}
		req.Header.Set("content-type", "application/json")
		res, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()

		o := make(map[string]interface{})
		if err := json.NewDecoder(req.Body).Decode(&o); err != nil {
			return nil, err
		}

		if res.StatusCode >= 200 && res.StatusCode < 300 {
			delete(again, rt)
			return o, nil
		}
		//update retry and extend the backoff due to failure
		newRetry := retry{
			lastAttempt:    now,
			currentBackoff: rt.currentBackoff * 2,
		}
		newRetry.nextTime = newRetry.lastAttempt.Add(newRetry.currentBackoff)
		again[newRetry] = t
		delete(again, rt)

		return o, nil
	}
}

//Sender is a goroutine that receives configuration.Transporters and uses a single http client to send http post request utilising keep alives
func Sender(destinationURL string, senderFunnel <-chan configuration.Transporter) error {
	//create sender map to ensure failed sends by URL are stored and retried
	attempt := make(again)
	//new dispatcher to commit the send and rerun failed
	dispatch := again.newDispatcher(attempt, newClient(), destinationURL)
	//setup PubSub goroutine configs
	pubsub := make(chan configuration.Transporter)
	gcpProjectID := os.Getenv("PROJECT_ID")
	gcp := true
	if gcpProjectID == "" {
		gcp = false
	}

	//run GCP Publisher if GCP details are available
	if gcp {
		go configuration.Publisher(pubsub)
	}

	log.Printf("Sender ready to receive using GCP project '%s' and target URL '%s'", gcpProjectID, destinationURL)
	//loop the chan
	for {
		select {
		case t := <-senderFunnel:
			fmt.Printf("Outbound: %+v\n\n", t)
			if gcp {
				pubsub <- t
			}

			if destinationURL != "" {
				rt := retry{
					lastAttempt:    time.Now(),
					currentBackoff: 1 * time.Minute,
				}

				parsedResponse, err := dispatch(t, rt)
				if err != nil {
					return err
				}
				log.Printf("%+v", parsedResponse)
			}

		default: //by default it searches for a single qualifying retry entry and does the http post request again
			if destinationURL != "" {
				now := time.Now()
				for info, transport := range attempt {
					if now.After(info.nextTime) {
						parsedResponse, err := dispatch(transport, info)
						if err != nil {
							return err
						}
						log.Printf("%+v", parsedResponse)
						break
					}
				}
			}
		}
	}

}

//newClient returns a custom http client tuned for the dispatch pkg
func newClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:          40,
			MaxIdleConnsPerHost:   2,
			IdleConnTimeout:       5 * time.Minute,
			ResponseHeaderTimeout: 20 * time.Second,
			TLSHandshakeTimeout:   15 * time.Second,
			DisableKeepAlives:     false,
			MaxConnsPerHost:       0,
		},
		Timeout: 30 * time.Second,
	}
}
