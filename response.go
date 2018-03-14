package throttler

import "net/http"

// Response contains the structure returned by the client handler which is finally inserted in the request.ResponseChannel in fulfill().
type Response struct {
	HRes *http.Response
	Err  error
}
