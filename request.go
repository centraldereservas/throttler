package throttler

import (
	"context"
	"net/http"
	"time"
)

// Request is the request struct that are passed into the requests channel to be processed when a ticket is obtained.
type Request struct {
	ctx     context.Context
	name    string
	hreq    *http.Request
	resChan chan *Response
	timeout time.Duration
}
