package pinger

import (
	"net/http"
	"time"
)

func newClient() http.Client {
	return http.Client{
		Transport: &http.Transport{
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
			DisableKeepAlives:     true, //as testing response speed so standardising here
			ExpectContinueTimeout: 10 * time.Second,
		},
		Timeout: 30 * time.Second,
	}
}
