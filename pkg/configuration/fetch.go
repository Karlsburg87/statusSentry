package configuration

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

func FetchConfig(client *http.Client, configLocation string) (*Configuration, error) {
	res, err := client.Get(configLocation)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var out *Configuration
	if err := json.NewDecoder(res.Body).Decode(out); err != nil {
		return nil, err
	}
	log.Println("new config fetched from server")
	return out, nil
}

func FetchTestConfig() (*Configuration, error) {
	log.Println("Test config fetched")
	return &Configuration{
		Config{
			ServiceName:   "Stripe",
			DisplayDomain: "stripe.com",
			StatusPage:    "https://status.stripe.com/",
			TargetHook:    "twitter:@stripestatus",
			PollFrequency: Frequency(5 * time.Minute),
			PollPages:     []string{"https://www.stripe.com"},
		},
		Config{
			ServiceName:   "Paypal Services (incl. Braintree)",
			DisplayDomain: "paypal.com",
			StatusPage:    "https://www.paypal-status.com/product/production",
			TargetHook:    "rss:https://www.paypal-status.com/feed/rss",
			PollFrequency: Frequency(5 * time.Minute),
			PollPages:     []string{"https://www.paypal.com/uk/home", "https://www.braintreepayments.com/"},
		},
		Config{
			ServiceName:   "Salesforce UK",
			DisplayDomain: "salesforce.com",
			StatusPage:    "https://status.salesforce.com/",
			TargetHook:    "email:status_alerts@salesforce.com",
			PollFrequency: Frequency(5 * time.Minute), //indicates instant
			PollPages:     []string{"https://salesforce.com/uk"},
		},
		Config{
			ServiceName:   "GoCardless",
			DisplayDomain: "gocardless.com",
			StatusPage:    "https://www.gocardless-status.com",
			TargetHook:    "rss:https://www.gocardless-status.com/history.rss",
			PollFrequency: Frequency(5 * time.Minute),
			PollPages:     []string{"https://www.gocardless.com"},
		},
		Config{
			ServiceName:   "Atlassian - Jira",
			DisplayDomain: "https://www.atlassian.com/software/jira",
			StatusPage:    "https://jira-software.status.atlassian.com",
			TargetHook:    "rss:https://jira-software.status.atlassian.com/history.rss",
			PollFrequency: Frequency(5 * time.Minute),
			PollPages:     []string{"https://www.atlassian.com/software/jira"},
		},
	}, nil
}
