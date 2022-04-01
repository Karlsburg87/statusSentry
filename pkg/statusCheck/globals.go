package statuscheck

import "net/http"

var (
	httpClient *http.Client
)

func init() {
	httpClient = newClient()
}
