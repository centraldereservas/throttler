package throttler

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"testing"
	"time"
)

type MockSender struct {
	sendMock func(*Request) *Response
}

func (m *MockSender) send(req *Request) *Response {
	return m.sendMock(req)
}

type contextMode int

const (
	contextDoneNotCalled contextMode = iota
	contextDoneCalledBeforeSend
	contextDoneCalledAfterSend
)

func createHTTPResponse(req *http.Request, body string) *http.Response {
	resp := &http.Response{
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
		Body:       ioutil.NopCloser(bytes.NewBufferString(body)),
		Request:    req,
		StatusCode: http.StatusOK,
	}
	return resp
}

func createResponse(req *http.Request, body string) *Response {
	hresp := createHTTPResponse(req, body)
	return &Response{
		HRes: hresp,
		Err:  nil,
	}
}

func TestFulfill(t *testing.T) {
	tt := []struct {
		name     string
		ctxMode  contextMode
		testData string
	}{
		{"Positive TC", contextDoneNotCalled, "body content"},
		{"Negative TC: context done before send", contextDoneCalledBeforeSend, "body content"},
		{"Negative TC: context done after send", contextDoneCalledAfterSend, "body content"},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			var cancel context.CancelFunc
			called := false
			mockSender := &MockSender{
				sendMock: func(req *Request) *Response {
					called = true
					req.ResChan <- createResponse(req.HReq, tc.testData)
					if tc.ctxMode == contextDoneCalledAfterSend {
						req.Ctx, cancel = context.WithDeadline(req.Ctx, time.Now().Add(-7*time.Hour))
						cancel()
					}
					return &Response{}
				},
			}
			fulfiller := NewFulfiller(mockSender)
			ctx := context.Background()

			if tc.ctxMode == contextDoneCalledBeforeSend {
				ctx, cancel = context.WithDeadline(ctx, time.Now().Add(-7*time.Hour))
				cancel()
			}

			resChan := make(chan *Response)
			req := &Request{
				HReq:    createHTTPRequest(),
				Ctx:     ctx,
				Name:    "test request",
				Timeout: 5 * time.Second,
				ResChan: resChan,
			}

			go fulfiller.fulfill(req)

			select {
			case <-req.Ctx.Done():
				if tc.ctxMode == contextDoneNotCalled {
					t.Errorf("context done not expected")
				}
			case resp, ok := <-req.ResChan:
				if !ok {
					t.Errorf("could not read from a closed channel")
				}
				if resp == nil {
					t.Errorf("empty response")
				}
				if resp.Err != nil {
					t.Errorf("response with errors: %v", resp.Err)
				}
				if !called {
					t.Errorf("did not call client.Do")
				}

				defer resp.HRes.Body.Close()

				if resp.HRes.StatusCode != http.StatusOK {
					t.Errorf("unexpected http status: %v", resp.HRes.Status)
				}

				bodyBytes, err := ioutil.ReadAll(resp.HRes.Body)
				if err != nil {
					t.Errorf("unable to read the response body: %v", err)
				}
				bodyString := string(bodyBytes)
				if bodyString != tc.testData {
					t.Errorf("expected this test data: %v; got %v", tc.testData, bodyString)
				}
			}
		})
	}
}
