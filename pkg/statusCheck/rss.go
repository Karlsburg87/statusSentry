package statuscheck

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/karlsburg87/Status/pkg/configuration"
)

/*********************************************************
RSS.go is a status fetching group of functions that work
in unison to provide statusCheck functionality across
status pages using a variety of api mediums.

It implements Transports which allows it to handoff
information it fetches to other services in a standard
format using standardised protocols
*********************************************************/

//Primary goroutine -------------------------------------------------------------------

//runRSSOperations is the main function that receives a config item and fetches a status update via an RSS feed
//  before handing off to other services
func runRSSOperations(c <-chan configuration.Config, sender chan<- configuration.Transporter) {
	services := make(map[string]time.Time) //serviceName against last publish date on the Channel.pubdate(of lastBuildDate if pubdate is blank)
	for config := range c {
		_, l := config.ParseServiceInfo()
		feed, err := getRSSFeed(l)
		if err != nil {
			log.Panicln(err)
		}
		// TODO: parse the rss response and send each relevent *rssItem object somewhere for processing and storage
		lastPubDate, ok := services[config.ServiceName]
		if !ok {
			lastPubDate = time.Now().Add(-24 * time.Hour)
		}
		feedItems, err := feed.GetLatest(lastPubDate, &config)
		if err != nil {
			log.Panicln(err)
		}
		for _, feedItem := range feedItems {
			feedItem.Send(config, sender)
		}

		d := feed.Channel.PubDate
		if d == "" {
			d = feed.Channel.LastBuildDate
		}

		t, err := parseRSSDate(d, 0)
		if err != nil {
			log.Panicln(err)
		}

		services[config.ServiceName] = t
	}
}

//parseRSSDate parses the dates in an RSS string with allowances for the wide formatting range found in RSS in the wild
func parseRSSDate(date string, zero int) (time.Time, error) {
	formatsToTry := []string{time.RFC822, time.RFC1123, time.RFC822Z, time.RFC1123Z, time.RFC3339}
	t, err := time.Parse(formatsToTry[zero], date)

	if err != nil {
		zero += 1
		if zero >= len(formatsToTry) {
			return t, err
		}
		t, err = parseRSSDate(date, zero)
	}

	return t, err
}

//top level functions ------------------------------------------------------------------

func getRSSFeed(loc string) (*rss, error) {
	res, err := httpClient.Get(loc)
	if err != nil {
		return nil, fmt.Errorf("http.Client.Get error in getRSSFeed: %v", err)
	}
	defer res.Body.Close()

	var out *rss
	if err := xml.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("xml decode fail in getRSSFeed: %v", err)
	}
	if err := out.normaliseXMLFormattedText(); err != nil {
		return nil, fmt.Errorf("normaliseXMLFormattedText fail in getRSSFeed: %v", err)
	}

	return out, nil
}

//GetLatest aggregates the rssItems since the last fetch that needs to be sent on to the next service
func (rss rss) GetLatest(lastPubDate time.Time, config *configuration.Config) ([]rssItem, error) {
	out := make([]rssItem, 0)
	for _, item := range rss.Channel.Items {
		t, err := parseRSSDate(item.PubDate, 0)
		if err != nil {
			return nil, err
		}

		if t.After(lastPubDate) {
			out = append(out, item)
		}

	}
	return out, nil
}

//Response formatting -----------------------------------------------------------------------

//normaliseXMLFormattedText removes the HTML encodings and returns human readable plain text
func (rss *rss) normaliseXMLFormattedText() error {
	for _, unit := range rss.Channel.Items {
		if unit.Description != nil {
			unit.Description.Clean = normaliseXMLFormattedText(unit.Description.Data)
		}
		if unit.ContentEncoded != nil {
			unit.ContentEncoded.Clean = normaliseXMLFormattedText(unit.ContentEncoded.Data)
		}
	}
	return nil
}

//normaliseXMLFormattedText removes the HTML encodings and returns human readable plain text
func normaliseXMLFormattedText(text string) string {
	text = html.UnescapeString(text)
	//newline replacers
	replacer := strings.NewReplacer("<br />", "\n", "<br/>", "\n", "</p>", "\n", "<li>", "\n- ", "</ul>", "\n", "<small>", "[", "</small>", "]", "<strong>", "**", "</strong>", "**", "<b>", "**", "</b>", "**", "<i>", "_", "</i>", "_")
	text = replacer.Replace(text)
	//remove the remaining html charecters
	htmlChars := regexp.MustCompile(`<[a-z\/ ='\-]+?>`)
	text = htmlChars.ReplaceAllLiteralString(text, "")
	//spaces
	excessiveSpaces := regexp.MustCompile(`( |\\n|\n){3,}`)
	text = excessiveSpaces.ReplaceAllString(text, "$1$1")
	text = strings.TrimSpace(text)
	return text
}

/*****************************************************************************************
* RSS types
*
* XML unmarshal by example at: https://tutorialedge.net/golang/parsing-xml-with-golang/
*****************************************************************************************/

//rss is a standard rss structure
type rss struct {
	XMLName xml.Name   `xml:"rss" json:"-"`
	Channel rssChannel `xml:"channel" json:"channel"`
}

//rssChannel is the contents of a rss channel
type rssChannel struct {
	XMLName       xml.Name  `xml:"channel" json:"-"`
	Title         string    `xml:"title" json:"title"`
	Link          string    `xml:"link" json:"link"`
	Description   string    `xml:"description" json:"description"`
	LastBuildDate string    `xml:"lastBuildDate" json:"last_build_date"`
	Docs          string    `xml:"docs" json:"docs"`
	Generator     string    `xml:"generator" json:"generator"`
	Items         []rssItem `xml:"item" json:"items"`
	PubDate       string    `xml:"pubDate" json:"pub_date"`
}

//rssItem is the content of an individual rss entry which could be one of many
type rssItem struct {
	XMLName     xml.Name    `xml:"item" json:"-"`
	Title       string      `xml:"title" json:"title"`
	Link        string      `xml:"link" json:"link"`
	GUID        string      `xml:"guid" json:"guid"`
	PubDate     string      `xml:"pubDate" json:"pub_date"`
	Description *rssContent `xml:"description" json:"description,omitempty"`

	//Namespace representation:
	// https://validator.w3.org/feed/docs/howto/declare_namespaces.html
	// (point 3 in: ) https://pkg.go.dev/encoding/xml@go1.18#Unmarshal
	ContentEncoded *rssContent `xml:"http://purl.org/rss/1.0/modules/content/ encoded" json:"content_encoded,omitempty"`
}

//rssContent contains the raw xml content of content:encoded XML tag in the RSS standard
type rssContent struct {
	Data  string `xml:",chardata" json:"raw"`
	Clean string `xml:"-" json:"readable"`
}

//Helper Methods and functions on types --------------------------------------------

//toJSON converts an RSS page to a json object
func (rss rss) toJSON() ([]byte, error) {
	return toJSON(rss)
}

//toJSON converts an RSS item to a json object
func (rssItem rssItem) toJSON() ([]byte, error) {
	return toJSON(rssItem)
}

//toJSON is the generic helper function for converting rss feeds to json without escaping HTML charecters
func toJSON(rss interface{}) ([]byte, error) {
	jenc := bytes.NewBuffer([]byte{})
	enc := json.NewEncoder(jenc)
	enc.SetEscapeHTML(false)
	enc.SetIndent(" ", " ")
	if err := enc.Encode(rss); err != nil {
		return nil, fmt.Errorf("error encoding json in rss.toJson: %v", err)
	}
	return jenc.Bytes(), nil
}

/**************************************************************************************************
// Implement Transporter on RSSItem
**************************************************************************************************/

//ToTransport creates a Transport object from the RSSItem. Needed to implement Transports
func (rssItem rssItem) ToTransport(conf configuration.Config) (configuration.Transporter, error) {
	t := configuration.Transporter{
		DisplayServiceName:       conf.ServiceName,
		DisplayDomain:            conf.DisplayDomain,
		Message:                  rssItem.ContentEncoded.Clean,
		RawMessage:               rssItem.ContentEncoded.Data,
		MessagePublishedDateTime: rssItem.PubDate,
		MetaStatusPage:           conf.StatusPage,
	}
	//Give Description is ContentEncoded is empty (standard says if both are present description is summary and content encoded is full text)
	if t.Message == "" {
		t.Message = rssItem.Description.Clean
		t.RawMessage = rssItem.Description.Data
	}
	//Format MessagePublishedTime to RFC3339. RSS2.0 uses RFC822 as standard. See http://validator.w3.org/feed/docs/rss2.html
	if t.MessagePublishedDateTime != "" {
		w, err := parseRSSDate(t.MessagePublishedDateTime, 0)
		if err != nil {
			return t, err
		}

		t.MessagePublishedDateTime = w.Format(time.RFC3339)
	}
	return t, nil
}

//Send sends the RSSItem to the next internal service using the standard Transporter format. Needed to implement Transports
func (rssItem rssItem) Send(conf configuration.Config, sender chan<- configuration.Transporter) error {
	transport, err := rssItem.ToTransport(conf)
	if err != nil {
		return err
	}
	sender <- transport
	return nil
}
