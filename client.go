package throttler

import "net/http"

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type sender interface {
	send(*Request) *Response
}

type clientHandler struct {
	client httpClient
}

func newClientHandler(c httpClient) sender {
	return &clientHandler{client: c}
}

// send the http request to the client
func (hdlr *clientHandler) send(req *Request) *Response {
	results, err := hdlr.client.Do(req.HReq)
	return &Response{
		HRes: results,
		Err:  err,
	}
}
