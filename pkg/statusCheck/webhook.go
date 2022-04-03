package statuscheck

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/karlsburg87/statusSentry/pkg/configuration"
)

//newMux is a multiplexer that serves status webhook routes to the correct handling function
func newMux(ctx context.Context, validCheckWebhook chan<- validator, validCheckEmail chan<- validator, sender chan<- configuration.Transporter) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintf(w, "Welcome to Status Sentry") })
	mux.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		//check incoming for service name identifiers
		identifier := r.URL.Path //the path of format "/webhook/$ServiceName" which must correspond to the ServiceName in the config
		//match with config to ensure it is a valid update
		validation := validator{
			serviceName: strings.Split(identifier, "/")[len(identifier)-1], //the path has format "/webhook/$ServiceName"
			valid:       make(chan configuration.Config),
		}
		validCheckWebhook <- validation

		select { //allow for ctx cancel
		case <-ctx.Done():
			close(validCheckWebhook)
			validCheckWebhook = nil
			return
		case item := <-validation.valid:
			if item.ServiceName != "" {
				//parse and do something with the match and the webhook info - decode json and use
				body, err := io.ReadAll(r.Body)
				if err != nil {
					log.Panicln(err)
				}
				if err := (webhookReceive{message: string(body)}).Send(item, sender); err != nil {
					log.Panicln(err)
				}
				return
			}
			//else log error
			toDebug := make(map[string]interface{})
			json.NewDecoder(r.Body).Decode(&toDebug)
			log.Printf("Update received deemed to be non valid: %+v\n", toDebug)
		}

	})

	//Also be alive to email JSON alerts from cloudmailin - email.go for logic
	mux.HandleFunc("/email", func(w http.ResponseWriter, r *http.Request) {
		//TODO: Basic Auth check
		emailHandler(w, r, ctx, validCheckEmail, sender)
	})

	return mux
}

//runStatusWebhookServer is a goroutine for handling webhook status updates
func runStatusWebhookServer(ctx context.Context, validCheck validators, sender chan<- configuration.Transporter) {
	var err error
	port := 8080 //default
	if raw := os.Getenv("PORT"); raw != "" {
		port, err = strconv.Atoi(raw)
		if err != nil {
			log.Panicln(err)
		}
	}

	server := newServer(newMux(ctx, validCheck.webhook, validCheck.email, sender), port)
	log.Printf("Webhook server now running on port %d\n", port)
	log.Fatalln(server.ListenAndServe())
}

//webhookReceive implements Transports
type webhookReceive struct {
	message string
}

func (wh webhookReceive) ToTransport(conf configuration.Config) (configuration.Transporter, error) {
	return configuration.Transporter{
		DisplayServiceName:       conf.ServiceName,
		DisplayDomain:            conf.DisplayDomain,
		Message:                  wh.message,
		RawMessage:               wh.message,
		MessagePublishedDateTime: time.Now().Format(time.RFC3339),
		MetaStatusPage:           conf.StatusPage,
	}, nil
}
func (wh webhookReceive) Send(conf configuration.Config, sender chan<- configuration.Transporter) error {
	transport, err := wh.ToTransport(conf)
	if err != nil {
		log.Panicln(err)
	}
	sender <- transport
	return nil
}
