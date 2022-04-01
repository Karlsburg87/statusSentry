package pinger

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/http/httptrace"
	"time"

	"github.com/karlsburg87/Status/pkg/configuration"
	"github.com/karlsburg87/Status/pkg/dispatch"
)

func Launch(ctx context.Context, conf <-chan *configuration.Configuration) {
	go Ping(ctx, conf)
}

//Ping is the goroutine responsible for webpage uptime polling
func Ping(ctx context.Context, conf <-chan *configuration.Configuration) {
	//logbook records the URL string that has been called against the call time so frequency can be respected
	logbook := make(map[string]time.Time)
	//initial configs
	configs := <-conf
	//spin up sender that sends to pubsub and other services
	sender := make(chan configuration.Transporter)
	go dispatch.Sender("", sender)
	//spin up poller which does the ping and collects the data
	pinger := make(chan *configuration.Configuration)
	go ping(pinger, logbook, sender)

	//control pace
	tckr := time.NewTicker(15 * time.Second)

	for {
		select {
		case config := <-conf: //update config list
			configs = config
		case <-ctx.Done():
			return
		case <-tckr.C:
			pinger <- configs
		}
	}
}

//ping sends a get request to each poll url in the configuration.Configuration
// and records to logbook and sends a Transport for each
func ping(config <-chan *configuration.Configuration, logbook map[string]time.Time, sender chan<- configuration.Transporter) {
	httpClient := newClient()
	//start polling worker pool
	pageChan := make(chan pageParcel)
	for i := 0; i < 20; i += 1 {
		go poll(pageChan, &httpClient)
	}

	log.Printf("ping worker pool ready to receive\n")

	for conf := range config {
		for _, item := range *conf {
			//test if it is time to poll this config item
			if !item.IsReadyToPoll() {
				continue
			}
			for _, page := range item.PollPages {
				fmt.Printf("pinging %s\n", page)
				//test the response of the page
				goPoll := pageParcel{
					url:          page,
					responseData: make(chan configuration.PingResponse),
					config:       item,
				}
				pageChan <- goPoll
				//send off data for that page
				pingDetails := <-goPoll.responseData
				fmt.Printf("ping response: %+v\n", pingDetails)
				if err := pingDetails.Send(item, sender); err != nil {
					log.Printf("error on PingResponse.Send for URL %s and error : %v", page, err)
				}
				item.SetFetchTime(pingDetails.TimeGo)
			}
		}
	}
}

//pageParcel is for communication between ping writer func and poll receiver goroutines
type pageParcel struct {
	url          string
	responseData chan configuration.PingResponse
	config       configuration.Config
}

func poll(pageChan <-chan pageParcel, client *http.Client) error {
	//tracing variables
	var start, dns, tlsHandshake, connect time.Time
	var toFirstResponseDuration, dnsDuration, tlsHandshakeDuration, connectDuration time.Duration

	trace := &httptrace.ClientTrace{
		WroteRequest: func(wri httptrace.WroteRequestInfo) { start = time.Now() },
		DNSStart:     func(dsi httptrace.DNSStartInfo) { dns = time.Now() },
		DNSDone: func(ddi httptrace.DNSDoneInfo) {
			dnsDuration = time.Since(dns)
		},

		TLSHandshakeStart: func() { tlsHandshake = time.Now() },
		TLSHandshakeDone: func(cs tls.ConnectionState, err error) {
			tlsHandshakeDuration = time.Since(tlsHandshake)
		},

		ConnectStart: func(network, addr string) { connect = time.Now() },
		ConnectDone: func(network, addr string, err error) {
			connectDuration = time.Since(connect)
		},

		GotFirstResponseByte: func() {
			toFirstResponseDuration = time.Since(start)
		},
	}
	for page := range pageChan {
		req, err := http.NewRequest(http.MethodGet, page.url, nil)
		if err != nil {
			log.Printf("error making new request for polling URL %s", page.url)
			log.Panicln(err)
		}
		//add in the trace
		req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

		//if just measuring if there was a response at all without caring for status
		//code or dealing with redirects,etc (like ping command line program)
		/*if _, err := http.DefaultTransport.RoundTrip(req); err != nil {
			log.Fatal(err)
		}*/

		res, err := client.Do(req)
		if err != nil {
			log.Printf("error on client.Do for polling URL %s", page.url)
			log.Panicln(err)
		}
		//get ready to record result
		tme := time.Now()
		page.responseData <- configuration.PingResponse{
			StatusPage:  page.config.StatusPage,
			ServiceName: page.config.ServiceName,
			Domain:      page.config.DisplayDomain,
			URL:         page.url,
			StatusCode:  res.StatusCode,
			Time:        tme.Format(time.RFC3339),
			TimeGo:      tme,
			ResponseTimes: configuration.PingTimes{
				DNS:           dnsDuration.Milliseconds(),
				TLSHandshake:  tlsHandshakeDuration.Milliseconds(),
				Connect:       connectDuration.Milliseconds(),
				FirstResponse: toFirstResponseDuration.Milliseconds(),
			},
		}
	}
	return nil
}
