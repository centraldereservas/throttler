package throttler

import (
	"fmt"
	"time"
)

type listener interface {
	listen()
}

type requestHandler struct {
	rate      time.Duration
	reqChan   chan *Request
	verbose   bool
	fulfiller fulfiller
}

func newListener(r time.Duration, ch chan *Request, v bool, f fulfiller) (listener, error) {
	if ch == nil {
		return nil, fmt.Errorf("request channel can not be nil")
	}
	if f == nil {
		return nil, fmt.Errorf("fulfiller can not be nil")
	}
	return &requestHandler{
		rate:      r,
		reqChan:   ch,
		verbose:   v,
		fulfiller: f,
	}, nil
}

// listen waits for receiving new requests from the requests channel and processes them
// without exceeding the calculated maximal rate limit using the leaky bucket algorithm
func (l *requestHandler) listen() {
	throttle := time.Tick(l.rate)
	for req := range l.reqChan {
		<-throttle
		if l.verbose {
			fmt.Printf("[%v] got ticket; Fulfilling Request [%v]\n", time.Now(), req.Name)
		}
		go l.fulfiller.fulfill(req)
		if l.verbose {
			fmt.Printf("[%v] Request fulfilled [%v]\n", time.Now(), req.Name)
		}
	}
}
