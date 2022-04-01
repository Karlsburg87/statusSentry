package statuscheck

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/karlsburg87/Status/pkg/configuration"
)

func TestRunRSSOperations(t *testing.T) {
	//test config
	c := make(chan configuration.Config)
	s := make(chan configuration.Transporter)
	go runRSSOperations(c, s)

	conf, err := configuration.FetchTestConfig()
	if err != nil {
		t.Error(err)
	}
	for _, item := range *conf {
		s, _ := item.ParseServiceInfo()
		if s == configuration.ServiceRSS {
			c <- item
		}
	}
}

func TestGetRSSFeed(t *testing.T) {
	//default the mandatory twitter token requirement
	if err := os.Setenv("TWITTER_TOKEN", "default"); err != nil {
		t.Error(err)
	}
	//fetch the test data
	conf, err := configuration.FetchTestConfig()
	if err != nil {
		t.Error(err)
	}
	//run the test data
	for _, item := range *conf {
		s, hook := item.ParseServiceInfo()
		if s == configuration.ServiceRSS {
			res, err := getRSSFeed(hook)
			if err != nil {
				t.Error(err)
			}
			//test for no data response
			if res.Channel.LastBuildDate == "" && res.Channel.PubDate == "" {
				t.Logf("%+v\n", item)
				jenc, err := json.MarshalIndent(res, " ", " ")
				if err != nil {
					t.Fatal(err)
				}
				t.Logf("%s\n", string(jenc))
				t.FailNow()
			}
			//test for no error description in response
			for _, unit := range res.Channel.Items {
				if unit.ContentEncoded == nil && unit.Description == nil {
					t.Logf("%+v\n", item)
					jenc, err := json.MarshalIndent(res, " ", " ")
					if err != nil {
						t.Fatal(err)
					}
					t.Logf("%s\n", string(jenc))
					t.FailNow()
				}
			}

		}
	}
}

func TestNormaliseXMLFormattedText(t *testing.T) {
	//fetch the test data
	conf, err := configuration.FetchTestConfig()
	if err != nil {
		t.Error(err)
	}
	//run the test data
	for _, item := range *conf {
		if serviceType, hook := item.ParseServiceInfo(); serviceType == configuration.ServiceRSS {
			res, err := getRSSFeed(hook)
			if err != nil {
				t.Fatal(err)
			}
			//test output for debug
			jenc, err := res.toJSON()
			if err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll("testOutput", 0755); err != nil {
				t.Fatal(err)
			}
			file, err := os.Create("testOutput/TestNormaliseXMLFormattedText.txt")
			if err != nil {
				t.Fatalf(err.Error())
			}
			defer file.Close()
			if _, err := file.Write(jenc); err != nil {
				t.Fatalf(err.Error())
			}
			if _, err := file.Write([]byte("\n\n")); err != nil {
				t.Fatalf(err.Error())
			}

		}
	}
}

func TestConfigParsing(t *testing.T) {
	//get test config
	//fetch the test data
	conf, err := configuration.FetchTestConfig()
	if err != nil {
		t.Error(err)
	}
	//Write results to file
	if err := os.MkdirAll("testOutput", 0755); err != nil {
		t.Fatal(err)
	}
	loc := "testOutput/TestConfigParsing.txt"
	file, err := os.Create(loc)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	enc.SetIndent(" ", " ")
	if err := enc.Encode(conf); err != nil {
		t.Fatal(err)
	}
	//Then read again
	file.Close()
	file, err = os.Open(loc)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	dec := json.NewDecoder(file)
	c := &configuration.Configuration{}
	if err := dec.Decode(c); err != nil {
		t.Fatal(err)
	}
}
