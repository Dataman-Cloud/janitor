package routing

import (
	"net/url"
)

type Target struct {
	Host string
	Port uint32
	URL  *url.URL
}
