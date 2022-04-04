# StatusSentry
**statusSentry** is the service that pings pages to check for 
- uptime 
- response speed 
- valid SSL certs *[roadmap]*
- SSL expiry times *[roadmap]*

and receives/pulls status updates from status pages from 
- RSS feeds
- twitter
- webhook
- email *via cloudmailin.com*

## Settings
Settings are established via environment variables

| Envar | Use |
|-|-|
|`PORT`|The port of the *webhook server*. The *config refresh* server is hardcoded as `8099`. Defaults to 8080|
|`TWITTER_TOKEN`|Accessing the Twitter API|
|`CONFIG_LOCATION`|Location the configuration JSON file is kept and updated. Expected to be a public URL endpoint|
|`STATUS_CHECK_ONLY`|Only runs the status checker service. No ping polling in the config will be checked and returned. Defaults false|
|`PINGER_ONLY`|Only runs the pinger service. No status pages in the config will be checked and returned. Defaults false|
|`OUTBOUND_URL`|The URI endpoint to sent status updates and ping polling stats to|

|GCP Pubsub Specific |to disable GCP PubSub set PROJECT_ID to ""|
|-|-|
|`GOOGLE_APPLICATION_CREDENTIALS`|The path of the JSON file that contains your service account key. Only required if not running application on Cloud Run or Cloud Functions|
|`PROJECT_ID`|The GCP project ID if using GCP PubSub|
|`STATUS_UPDATE_TOPIC`|GCP PubSub topic to publish status page updates to. Defaults to `statusUpdates`|
|`PING_RESPONSE_TOPIC`|GCP PubSub topic to publish ping responses to. Defaults to `pagePings`|

## Usage

To run from the source files enter the following into the terminal from the project root
```
TWITTER_TOKEN=XXXXXXXXXXXXX \
GOOGLE_APPLICATION_CREDENTIALS="credentials.json" \
PROJECT_ID=yourProject \
PORT=8080 \
go run main.go
```
Remember to replace the envars with your own values

## Configuration file

The configuration file allows the application to know which updates to expect to receive so it can verify and also which status endpoints to poll to receive status updates. 

In addition it will have a list of endpoints to poll and record uptime on.

### Configuration file format
The format is an array of configs 
```go 
type Configuration []Config
```
Where a config is a set of information on where to/from to get information and how frequently. ServiceName must match webhook endpoints given to the webhook service but DisplayDomain and StatusPage are entirely for labelling and meta data purposes. 

```go
type Config struct {
	ServiceName string      `json:"service_name"` 				//Mandatory
	DisplayDomain string    `json:"service_domain"`
	StatusPage string       `json:"status_page,omitempty"`
	TargetHook string       `json:"status_source,omitempty"` 	//Mandatory for status page updates
	PollFrequency Frequency `json:"poll_frequency"` 			//Mandatory for polling tasks
	PollPages []string      `json:"poll_pages"`					//Mandatory for polling tasks
}
```
ServiceName
: ServiceName is the readable name of a group of properties E.g. Amazon Web Services, Comic Relief, eBay 
: For WebHooks the incoming request must have the path "webhook/${ServiceName}" where ServiceName matches a Config.ServiceName item to be processed correctly


DisplayDomain
: The the top level domain that accompanies ServiceName as a descriptive label for the end user.
: Indicates the subdomains covered in the PollPages. E.g. ebay.com
: Metadata as data will generally be taken from PollPages (expected to all be of the same domain)or matched with ServiceName for human readability depending on use case

StatusPage 
: The human readable status page URI for the Service
: Metadata as application will read from TargetHook
	

TargetHook 
: The hook of the stream within the ServiceType to which to fetch if the ServiceType is a PULL type (e.g. RSS or Twitter)
:Prefixed with the service type

> e.g.
>	- twitter:@handle (twitter handle or id)
>	
>	- rss:https://websitefeed.com/rss (page to fetch feed from)
>	
>	- email:salesforce-status-alert@salesforce.com (incoming email address to look for)
>	
>	- webhook:/endpoint/path (path to look for at webhook endpoint)    
>
> *Notice there are no spaces in the string*

PollFrequency 
: The frequency with which to fetch an update. In Go duration string format when JSON marshalled: e.g. "1m","2h4m13s",etc

PollPages 
: The pages within the sub domain with which to poll for uptime and record response times
: Each one will return a PingResponse when conducted every Config.Frequency time period

### Raw JSON example

```json
[
  {
   "service_name": "Stripe",
   "service_domain": "stripe.com",
   "status_page": "https://status.stripe.com/",
   "status_source": "twitter:@stripestatus",
   "poll_frequency": "1m0s",
   "poll_pages": [
    "https://www.stripe.com"
   ]
  }
]
```
## Setting up email with mailcloud
Ensure you set the endpoing for webhooks to `/email` endpoint of the server at port `PORT` as set by the envar