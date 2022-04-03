package configuration

import (
	"log"
	"strconv"
	"strings"
	"time"
)

//Frequency is the duration type for the config
type Frequency time.Duration

func (freq Frequency) MarshalJSON() ([]byte, error) {
	out := strconv.Quote(time.Duration(freq).String())
	return []byte(out), nil
}
func (freq *Frequency) UnmarshalJSON(b []byte) error {
	unquoted, err := strconv.Unquote(string(b))
	if err != nil {
		return err
	}
	fq, err := time.ParseDuration(unquoted)
	if err != nil {
		return err
	}
	freq = (*Frequency)(&fq)
	return nil
}

//ServiceType is the enum type for the service type
type ServiceType string

const (
	ServiceWebhook ServiceType = "webhook"
	ServiceRSS     ServiceType = "rss"
	ServiceEmail   ServiceType = "email"
	ServiceTwitter ServiceType = "twitter"
)

//Configuration is the input to the application of various configs that can be interpreted by the application
type Configuration []Config

//Config is a single input of Configuration and the way the application receives configuration information
type Config struct {
	//ServiceName is the readable name of a group of properties E.g. Amazon Web Services, Comic Relief, eBay
	//
	//For WebHooks the incoming request must have the path "webhook/${ServiceName}" where ServiceName matches
	//a Config.ServiceName item to be processed correctly
	ServiceName string `json:"service_name"`
	//DisplayDomain is the is the top level domain that accompanies ServiceName as a descriptive label for the end user.
	//Indicates the subdomains covered in the PollPages. E.g. ebay.com
	//
	//Metadata as data will generally be taken from PollPages (expected to all be of the same domain)
	// or matched with ServiceName for human readability depending on use case
	DisplayDomain string `json:"service_domain"`
	//StatusPage is the human readable status page URI for the Service
	//
	//Metadata as application will read from TargetHook
	StatusPage string `json:"status_page,omitempty"`
	//TargetHook is the hook of the stream within the ServiceType to which to fetch if the ServiceType is a PULL type (e.g. RSS or Twitter)
	//
	//Prefixed with the service type
	// e.g.
	//
	//- twitter:@handle (twitter handle or id)
	//
	//- rss:https://websitefeed.com/rss (page to fetch feed from)
	//
	//- email:salesforce-status-alert@salesforce.com (incoming email address to look for)
	//
	//- webhook:/endpoint/path (path to look for at webhook endpoint)
	TargetHook string `json:"status_source,omitempty"`
	//PollFrequency is the frequency with which to fetch an update. In Go duration string format when JSON marshalled: e.g. "1m","2h4m13s",etc
	PollFrequency Frequency `json:"poll_frequency"`
	//PollPages are the pages within the sub domain with which to poll for uptime and record response times
	//
	//Each one will return a PingResponse when conducted every Config.Frequency time period
	PollPages []string `json:"poll_pages"`

	//latestFetch is the time of the last attempt to poll the pages in PollPages
	latestFetch time.Time `json:"-"`
	//serviceType is the service type removed from TargetHook
	serviceType ServiceType
	//plainTargetHook is the TargetHook without the service type prefix
	plainTargetHook string
}

//IsReadyToPoll returns whether it is time to poll the pages in PollPages.
//
//False means is either has no pages to poll or latestFetch has not passed by at least PollFrequency
func (config Config) IsReadyToPoll() bool {
	if len(config.PollPages) == 0 {
		return false
	}
	return config.latestFetch.Add(time.Duration(config.PollFrequency)).Before(time.Now())
}
func (config *Config) SetFetchTime(t time.Time) {
	if t.IsZero() {
		t = time.Now()
	}
	config.latestFetch = t
}

//ParseServiceInfo parses and splits the TargetHook into the service type and raw targetHook string components
func (config *Config) ParseServiceInfo() (serviceType ServiceType, plainTargetHook string) {
	if config.plainTargetHook != "" && config.serviceType != "" {
		return config.serviceType, config.plainTargetHook
	}
	prefixEnd := strings.Index(config.TargetHook, ":")
	serviceType = ServiceType(strings.TrimSpace(strings.ToLower(config.TargetHook[:prefixEnd])))
	plainTargetHook = config.TargetHook[prefixEnd+1:]
	//log.Printf("service type: %s\nraw target hook: %s", serviceType, plainTargetHook)
	return
}

//PingResponse is the response info from a ping on a PollPage inclusive of response time and status code
type PingResponse struct {
	//StatusPage is the human readable status page URI for the Service
	//
	//Metadata as application will read from TargetHook
	StatusPage    string     `json:"-"`
	ServiceName   string     `json:"-"`                   //ServiceName is taken from the Config instruction and is the readable name of a group of properties
	Domain        string     `json:"-"`                   //Domain is taken from the Config instruction and is the readable domain under which PollPages are grouped
	URL           string     `json:"pinged_url"`          //URL is the URL that was pinged
	ResponseTimes PingTimes  `json:"ping_response_times"` //ResponseTime are the collection of http response times in milliseconds
	StatusCode    int        `json:"ping_response_code"`  //StatusCode is the http status code 200 or 201 for OK and other RFC codes for various errors
	ErrorText     string     `json:"ping_error"`          //ErrorText may be either the status code text description or the body of the response if exists and status code is not 200/201
	Time          string     `json:"ping_time"`           //Time is the timestamp the ping was initiated at in RFC3339 format
	Certificates  []PingCert `json:"ping_certs"`          //Certificates are the TLS certificate information of the response server as sent in the response of the ping
	TimeGo        time.Time  `json:"-"`                   //TimeGo is Time but in usable format
}

//PingTimes is the collection of http response times in milliseconds
//
//See https://stackoverflow.com/questions/48077098/getting-ttfb-time-to-first-byte-value-in-golang/48077762#48077762
type PingTimes struct {
	DNS           int64 `json:"dns"`
	TLSHandshake  int64 `json:"tls_handshake"`
	Connect       int64 `json:"connect"`
	FirstResponse int64 `json:"first_response"`
}

type PingCert struct {
	ConnVerified bool   `json:"cert_primary"`     //ConnVerified signifies that this was the certificate the connection was verified against
	ValidFrom    string `json:"cert_valid_from"`  //ValidFrom is the date the server SSL certificate is considered valid from - RFC3339
	ValidUntil   string `json:"cert_valid_until"` //ValidUntil is the date the server SSL certificate expires after - RFC3339
	Issuer       string `json:"cert_issuer"`      //Issuer is the common name of the entity that issued the server TLS cert
	Subject      string `json:"cert_subject"`     //Subject is common name of the entity to which this cert has been issued
	IsExpired    bool   `json:"cert_expired"`     //IsExpired is if the server SSL certificate has expired

}

//ToTransport for pingResponse to implement Transports
func (ping PingResponse) ToTransport(conf Config) (Transporter, error) {
	return Transporter{
		PingResponse: &ping,
	}, nil
}

//Send for pingResponse to implement Transports
func (ping PingResponse) Send(conf Config, sender chan<- Transporter) error {
	transport, err := ping.ToTransport(conf)
	if err != nil {
		log.Panicln(err)
	}
	sender <- transport
	return nil
}
