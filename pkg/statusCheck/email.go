package statuscheck

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/karlsburg87/statusSentry/pkg/configuration"
)

//emailHandler receives alerts via email then recovers the corresponding config item before forwarding on to the sender function
//
//Implemented via the web server in webhook.go
func emailHandler(rw http.ResponseWriter, r *http.Request, ctx context.Context, validCheck chan<- validator, sender chan<- configuration.Transporter) {
	//get json body
	payload := &emailJSON{}
	if err := json.NewDecoder(r.Body).Decode(payload); err != nil {
		//respond back with error message to display on the cloudmailin dashboard
		rw.WriteHeader(http.StatusInternalServerError)
		out := map[string]string{
			"error": err.Error(),
		}
		if err := json.NewEncoder(rw).Encode(out); err != nil {
			log.Panicf("could not json encode error message for email mux: %v", err)
		}
	}
	//Get corresponding config
	validation := validator{
		emailAddress: payload.Envelope.From, //should match config.
		valid:        make(chan configuration.Config),
	}
	validCheck <- validation

	select { //allow for ctx cancel
	case <-ctx.Done():
		close(validCheck)
		validCheck = nil
		return
	case conf := <-validation.valid:
		if conf.ServiceName != "" {
			if err := payload.Send(conf, sender); err != nil {
				//respond back with error message to display on the cloudmailin dashboard
				rw.WriteHeader(http.StatusInternalServerError)
				out := map[string]string{
					"error": err.Error(),
				}
				if err := json.NewEncoder(rw).Encode(out); err != nil {
					log.Panicf("could not json encode error message for email mux: %v", err)
				}
			}
		}
	}
}

//emailJSON is provided by cloudmailin which uses this email representation in JSON
type emailJSON struct {
	Headers struct {
		ReturnPath            string   `json:"return_path"`
		Received              []string `json:"received"`
		Date                  string   `json:"date"`
		From                  string   `json:"from"`
		To                    string   `json:"to"`
		MessageID             string   `json:"message_id"`
		Subject               string   `json:"subject"`
		MimeVersion           string   `json:"mime_version"`
		ContentType           string   `json:"content_type"`
		DeliveredTo           string   `json:"delivered_to"`
		ReceivedSpf           string   `json:"received_spf"`
		AuthenticationResults string   `json:"authentication_results"`
		UserAgent             string   `json:"user_agent"`
	} `json:"headers"`
	Envelope struct {
		To         string   `json:"to"`
		From       string   `json:"from"`
		HeloDomain string   `json:"helo_domain"`
		RemoteIP   string   `json:"remote_ip"`
		Recipients []string `json:"recipients"`
		Spf        struct {
			Result string `json:"result"`
			Domain string `json:"domain"`
		} `json:"spf"`
		TLS bool `json:"tls"`
	} `json:"envelope"`
	Plain       string `json:"plain"`
	HTML        string `json:"html"`
	ReplyPlain  string `json:"reply_plain"`
	Attachments []struct {
		Content     string `json:"content"`
		FileName    string `json:"file_name"`
		ContentType string `json:"content_type"`
		Size        int    `json:"size"`
		Disposition string `json:"disposition"`
	} `json:"attachments"`
}

func (email emailJSON) ToTransport(conf configuration.Config) (configuration.Transporter, error) {
	return configuration.Transporter{
		DisplayServiceName:       conf.ServiceName,
		DisplayDomain:            conf.DisplayDomain,
		Message:                  email.HTML,
		RawMessage:               email.Plain,
		MessagePublishedDateTime: email.Headers.Date,
		MetaStatusPage:           conf.StatusPage,
	}, nil
}

func (email emailJSON) Send(conf configuration.Config, sender chan<- configuration.Transporter) error {
	transport, err := email.ToTransport(conf)
	if err != nil {
		log.Panicln(err)
	}
	sender <- transport
	return nil
}
