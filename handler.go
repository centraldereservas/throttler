package throttler

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Handler controls that the queued requests do not overtake the provider rate limits.
type Handler interface {
	// Rate returns the minimal time.Duration between sending two requests
	Rate() time.Duration

	// SetClient allows to use an alternative to http.DefaultClient
	SetClient(client *http.Client)

	// StartRequestsHandler initiates the process responsible to attend the requests from the requestsChannel
	StartRequestsHandler()

	// Queue builds a new throttle.Request and put it into the requestsChannel
	Queue(ctx context.Context, name string, hreq *http.Request, timeout time.Duration) (*http.Response, error)
}

type handler struct {
	reqChan               chan *Request
	rate                  Rate
	requestHandlerStarted bool
	verbose               bool
	client                *http.Client
}

// NewHandler configure the requests channel and starts the handler in a new goroutine the requests handler
func NewHandler(rate Rate, reqChanCapacity int, verbose bool) (Handler, error) {
	if rate == nil {
		return nil, fmt.Errorf("rate can not be nil")
	}
	if reqChanCapacity < 0 {
		return nil, fmt.Errorf("reqChanCapacity must be greater than zero")
	}

	// creates the channel for enqueuing requests
	requestsCh := make(chan *Request, reqChanCapacity)

	// execute in a new goroutine the handler responsible for fulfilling
	// the queued requests read from the channel at the configured frequency
	handler := &handler{
		reqChan:               requestsCh,
		rate:                  rate,
		verbose:               verbose,
		requestHandlerStarted: false,
		client:                http.DefaultClient,
	}
	return handler, nil
}

// SetClient allows the user to provide it's desired http.Client specifying timeouts, redirect policy, proxies
// or use the package with Google App Engine. This call is optional, by default the client is set to http.DefaultClient.
func (h *handler) SetClient(client *http.Client) {
	h.client = client
}

// StartRequestsHandler executes in a new goroutine the handler responsible for fulfilling
// the queued requests read from the channel at the configured frequency
func (h *handler) StartRequestsHandler() {
	go h.requestsHandler(h.Rate())
	h.requestHandlerStarted = true
}

// Queue is called to queue a new request into the requests channel.
// It assures that we will not overtake the rate limit constraint.
func (h *handler) Queue(ctx context.Context, name string, hreq *http.Request, timeout time.Duration) (*http.Response, error) {
	if !h.requestHandlerStarted {
		return nil, fmt.Errorf("requestHandler has not been started")
	}

	var res *Response
	c := make(chan *Response)
	defer close(c)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	request := &Request{ctx: ctx, name: name, hreq: hreq, resChan: c, timeout: timeout}
	h.reqChan <- request
	select {
	case <-ctx.Done():
		return nil, ctx.Err() // context cancelled
	case res = <-c:
		return res.hres, res.err
	}
}

// Rate returns the rate calculated as period + guardTime
func (h *handler) Rate() time.Duration {
	return h.rate.CalculateRate()
}

// requestsHandler is responsible for not exceeding the calculated maximal rate limit
// when processing requests using the leaky bucket algorithm
func (h *handler) requestsHandler(rate time.Duration) {
	throttle := time.Tick(rate)
	for req := range h.reqChan {
		<-throttle
		if h.verbose {
			fmt.Printf("[%v] got ticket; Fulfilling Request [%v]\n", time.Now(), req.name)
		}
		go h.fulfillRequest(req)
		if h.verbose {
			fmt.Printf("[%v] Request fulfilled [%v]\n", time.Now(), req.name)
		}
	}
}

// fulfillRequest is responsible for sending the request to the client and copy
// the response into the channel specified in the request
func (h *handler) fulfillRequest(req *Request) {
	var res *Response

	// check if the context is already cancelled before calling client.Do
	select {
	case <-req.ctx.Done():
		return // context aborted, do not send response
	default:
		// send the http request to the client
		results, err := h.client.Do(req.hreq)
		res = &Response{hres: results, err: err}
	}

	// check if the context is already cancelled after calling client.Do
	select {
	case <-req.ctx.Done():
		return // context aborted, do not send response
	default:
		req.resChan <- res // context is alive, submit response
	}
}
