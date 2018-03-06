package throttler

import "net/http"

// Response is the response struct returned after fulfill the request.
type Response struct {
	hres *http.Response
	err  error
}
