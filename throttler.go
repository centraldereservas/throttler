package throttler

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Limiter is the interface that contains the basic methods for using the throttler
// which controls that the queued requests do not overtake the provider rate limits.
type Limiter interface {
	// Rate returns the minimal allowed time.Duration between sending two requests
	Rate() time.Duration

	// Run initiates the process responsible to attend the requests from the requestsChannel
	Run()

	// Queue builds a new throttle.Request and queue it into the requestsChannel to be processed
	Queue(ctx context.Context, name string, hreq *http.Request, timeout time.Duration) (*http.Response, error)
}

type throttler struct {
	reqChan         chan *Request
	rate            Rate
	verbose         bool
	listener        listener
	listenerStarted bool
}

// New initializes the throttler handler.
func New(rate Rate, reqChanCapacity int, verbose bool, client *http.Client) (Limiter, error) {
	if rate == nil {
		return nil, fmt.Errorf("rate can not be nil")
	}
	if reqChanCapacity < 0 {
		return nil, fmt.Errorf("reqChanCapacity must be greater than zero")
	}
	if client == nil {
		client = http.DefaultClient
	}

	// creates the channel for enqueuing requests
	requestsCh := make(chan *Request, reqChanCapacity)

	// build services to be injected
	clientHandler := newClientHandler(client)
	fulfiller := newFulfiller(clientHandler)
	listener, _ := newListener(rate.CalculateRate(), requestsCh, verbose, fulfiller)

	throttler := &throttler{
		reqChan:         requestsCh,
		rate:            rate,
		verbose:         verbose,
		listener:        listener,
		listenerStarted: false,
	}
	return throttler, nil
}

// Run executes in a new goroutine the handler responsible for fulfilling
// the queued requests read from the channel at the configured frequency
func (t *throttler) Run() {
	go t.listener.listen()
	t.listenerStarted = true
}

// Queue is called to queue a new request into the requests channel.
// It assures that the system will not overtake the rate limit constraint.
func (t *throttler) Queue(ctx context.Context, name string, hreq *http.Request, timeout time.Duration) (*http.Response, error) {
	if !t.listenerStarted {
		return nil, fmt.Errorf("requestHandler has not been started")
	}

	var res *Response
	c := make(chan *Response)
	defer close(c)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	request := &Request{ctx, name, hreq, c, timeout}
	t.reqChan <- request
	select {
	case <-ctx.Done():
		return nil, ctx.Err() // context cancelled
	case res = <-c:
		return res.HRes, res.Err
	}
}

// Rate returns the rate calculated as period + guardTime
func (t *throttler) Rate() time.Duration {
	return t.rate.CalculateRate()
}
