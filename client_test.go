package throttler

import (
	"context"
	"log"
	"net/http"
	"testing"
	"time"
)

type MockHTTPClient struct {
	DoMock func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoMock(req)
}

func createHTTPRequest() *http.Request {
	url := "http://date.jsontest.com/"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalf("unable to create request: %v", err)
	}
	return req
}

func createRequest() *Request {
	hreq := createHTTPRequest()
	ctx := context.Background()
	resChan := make(chan *Response)
	req := &Request{
		HReq:    hreq,
		Ctx:     ctx,
		Name:    "test request",
		Timeout: 5 * time.Second,
		ResChan: resChan,
	}
	return req
}

func TestSend(t *testing.T) {
	tt := []struct {
		name string
	}{
		{"Positive TC"},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			mockHTTPClient := &MockHTTPClient{
				DoMock: func(req *http.Request) (*http.Response, error) {
					called = true
					return &http.Response{}, nil
				},
			}

			sender := NewClientHandler(mockHTTPClient)
			hreq := createHTTPRequest()
			ctx := context.Background()
			resChan := make(chan *Response)
			req := &Request{
				HReq:    hreq,
				Ctx:     ctx,
				Name:    "test request",
				Timeout: 5 * time.Second,
				ResChan: resChan,
			}
			res := sender.send(req)
			if res == nil {
				t.Errorf("empty response")
			}
			if res.Err != nil {
				t.Errorf("response with errors: %v", res.Err)
			}
			if !called {
				t.Fatalf("did not call client.Do")
			}
		})
	}
}
