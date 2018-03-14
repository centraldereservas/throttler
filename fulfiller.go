package throttler

type fulfiller interface {
	fulfill(req *Request)
}

type fulfillHandler struct {
	client sender
}

func newFulfiller(client sender) fulfiller {
	return &fulfillHandler{client: client}
}

// fulfill is responsible for sending the request to the client and copy
// the response into the channel specified in the request
func (f *fulfillHandler) fulfill(req *Request) {
	var res *Response

	// check if the context is already cancelled before calling client.Do
	select {
	case <-req.Ctx.Done():
		return // context aborted, do not send response
	default:
		res = f.client.send(req)
	}

	// check if the context was cancelled during the client.send call
	select {
	case <-req.Ctx.Done():
		return // context aborted, do not send response
	default:
		req.ResChan <- res // context is alive, submit response
	}
}
