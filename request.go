package throttler

import (
	"context"
	"net/http"
	"time"
)

// Request contains the basic structure to be send into the requests channel by Queue function
type Request struct {
	Ctx     context.Context
	Name    string
	HReq    *http.Request
	ResChan chan *Response
	Timeout time.Duration
}
