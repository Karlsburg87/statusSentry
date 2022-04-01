package statuscheck

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/karlsburg87/Status/pkg/configuration"
)

var twitterBearerToken string

func init() {
	twitterBearerToken = os.Getenv("TWITTER_TOKEN")
	if twitterBearerToken == "" {
		log.Println("TWITTER_TOKEN must be set to fetch Twitter based status pages")
		//FIXME: switch off twitter functionality if login not set
	}
}

//https://developer.twitter.com/en/docs/twitter-api/tweets/filtered-stream/integrate/build-a-rule

/*********************************************************
Twitter.go is a status fetching group of functions that work
in unison to provide statusCheck functionality across
status pages using a variety of api mediums.

It implements Transports which allows it to handoff
information it fetches to other services in a standard
format using standardised protocols
*********************************************************/

//Primary goroutine -------------------------------------------------------------------

//twitterSSOperations is the main function that receives a config item and fetches a status update via an twitter feed
//  before handing off to other services
func runTwitterOperations(c <-chan configuration.Config, sender chan<- configuration.Transporter) {
	services := make(map[string]time.Time) //twitterID against last fetch time
	for config := range c {
		_, twitterID := config.ParseServiceInfo()
		//check if twitterID is actually a twitter handle/username
		if strings.HasPrefix(twitterID, "@") || strings.ContainsAny(twitterID, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_") {
			//is handle so get UserID from get user endpoint
			var err error
			twitterID, err = getTwitterUser(twitterID)
			if err != nil {
				log.Panicln(err)
			}
		}
		timeline, err := getTwitterTimeline(twitterID, services)
		if err != nil {
			log.Panicln(err)
		}

		for _, tweet := range timeline.Data {
			if err := tweet.Send(config, sender); err != nil {
				log.Panicln(err)
			}
		}

	}
}

//getTwitterUser gets a twitter user from their username or handle
//
//returns the user ID as a string
func getTwitterUser(twitterHandle string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://api.twitter.com/2/users/by/username/%s", strings.TrimSpace(strings.TrimPrefix(twitterHandle, "@"))), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", twitterBearerToken))
	res, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	userInfo := twitterGetUser{}
	if err := json.NewDecoder(res.Body).Decode(&userInfo); err != nil {
		return "", err
	}
	return userInfo.Data.ID, nil
}

func getTwitterTimeline(twitterID string, serviceLog map[string]time.Time) (twitterTimeline, error) {
	//make a request to Twitter API
	uri, err := url.Parse(fmt.Sprintf("https://api.twitter.com/2/users/%s/tweets", twitterID))
	if err != nil {
		return twitterTimeline{}, err
	}
	if _, ok := serviceLog[twitterID]; !ok {
		serviceLog[twitterID] = time.Now().Add(-24 * time.Hour) //Limit to fetching tweets from max 24 hours ago
	}
	uri.Query().Add("start_time", serviceLog[twitterID].Format(time.RFC3339))
	uri.Query().Add("tweet.fields", "created_at")

	req, err := http.NewRequest(http.MethodGet, uri.String(), nil)
	if err != nil {
		return twitterTimeline{}, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", twitterBearerToken))
	logNow := time.Now()
	res, err := httpClient.Do(req)
	if err != nil {
		return twitterTimeline{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return twitterTimeline{}, fmt.Errorf("bad HTTP.Get call: %+v", res.Status)
	}
	serviceLog[twitterID] = logNow //log the new last activity time
	timeline := twitterTimeline{}
	if err := json.NewDecoder(res.Body).Decode(&timeline); err != nil {
		return twitterTimeline{}, err
	}
	return timeline, nil
}

/**************************************
* Twitter response types
**************************************/

type twitterGetUser struct {
	Data twitterUser `json:"data"`
}

type twitterUser struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
}

type twitterTimeline struct {
	Data []twitterTweet `json:"data"`
	Meta twitterMeta    `json:"meta"`
}

type twitterTweet struct {
	ID      string `json:"id"`
	Text    string `json:"text"`
	PubDate string `json:"created_at"`
}
type twitterMeta struct {
	OldestID    string `json:"oldest_id"`
	NewestID    string `json:"newest_id"`
	ResultCount int    `json:"result_count"`
	NextToken   string `json:"next_token"`
}
type twitterError struct {
	Errors []twitterErrorItem `json:"errors"`
	Title  string             `json:"title,omitempty"`
	Detail string             `json:"detail,omitempty"`
	Type   string             `json:"type,omitempty"`
}

type twitterErrorItem struct {
	Params       twitterErrorParam `json:"parameters"`
	Message      string            `json:"message,omitempty"`
	ResourceType string            `json:"resource_type,omitempty"`
	Title        string            `json:"title,omitempty"`
	Detail       string            `json:"detail,omitempty"`
	Type         string            `json:"type,omitempty"`
	Field        string            `json:"field,omitempty"`
}
type twitterErrorParam struct {
	EndTime []string `json:"end_time,omitempty"`
}

func (twittweet twitterTweet) Send(conf configuration.Config, sender chan<- configuration.Transporter) error {
	transport, err := twittweet.ToTransport(conf)
	if err != nil {
		log.Panicln(err)
	}
	sender <- transport
	return nil
}

func (twittweet twitterTweet) ToTransport(conf configuration.Config) (configuration.Transporter, error) {
	return configuration.Transporter{
		DisplayServiceName:       conf.ServiceName,
		DisplayDomain:            conf.DisplayDomain,
		Message:                  twittweet.Text,
		RawMessage:               twittweet.Text,
		MessagePublishedDateTime: twittweet.PubDate,
		MetaStatusPage:           conf.StatusPage,
	}, nil
}
