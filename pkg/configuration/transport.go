package configuration

import (
	"encoding/json"
)

//Transporter is the standard transport struct used by all system senders and receivers
type Transporter struct {
	//StausPage updates--------------------

	//DisplayServiceName is the given name of the service as to display to end users. From Config
	DisplayServiceName string `json:"display_name"`
	//DisplayDomain is the top level URL that gives context to the DisplayServiceName. From Config
	DisplayDomain string `json:"display_domain,omitempty"`
	//Message is the update readable text from the status update
	Message string `json:"message,omitempty"`
	//RawMessage is the update in its raw format from the source
	RawMessage string `json:"raw_message,omitempty"`
	//MessagePublishedTime is the time the status update was published as RFC3339
	MessagePublishedDateTime string `json:"pub_date,omitempty"`

	//Polling data------------------------

	//PingResponse is embedded object used to report on responses from ping checks
	*PingResponse `json:",omitempty"`

	//Meta---------------------------------

	//MetaStatusPage is the URL of the status page - different from the source of the updates used by the application
	MetaStatusPage string `json:"status_page,omitempty"`
}

//ToJSON returns a JSON representation of the transporter object
func (transporter Transporter) ToJSON() ([]byte, error) {
	return json.Marshal(transporter)
}

//Transports is an interface for objects that implement the ToTransport and Send methods
// that readies and commits it for inter service communication
type Transports interface {
	ToTransport(Config) (Transporter, error)
	Send(Config, chan<- Transporter) error
}
