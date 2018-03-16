package throttler

import (
	"io/ioutil"
	"net/http"
	"testing"
	"time"
)

type MockFulfiller struct {
	fulfillMock func(req *Request)
}

func (m *MockFulfiller) fulfill(req *Request) {
	m.fulfillMock(req)
}

func TestListen(t *testing.T) {
	tt := []struct {
		name            string
		reqChanCapacity int
		fullfillerNil   bool
		testData        string
		errMsg          string
	}{
		{"Positive TC", 10, false, "body content", ""},
		{"Negative TC: nil request channel", -1, false, "body content", "request channel can not be nil"},
		{"Negative TC: nil fulfiller", 1, true, "body content", "fulfiller can not be nil"},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			var channel chan *Request
			var req *Request
			if tc.reqChanCapacity != -1 {
				channel = make(chan *Request, tc.reqChanCapacity)

				// add a dummy request to be processed in the listen() function
				req = createRequest()
				channel <- req
			}

			called := false
			var mockFulfiller fulfiller
			if !tc.fullfillerNil {
				mockFulfiller = &MockFulfiller{
					fulfillMock: func(req *Request) {
						called = true
						req.ResChan <- createResponse(req.HReq, tc.testData)
					},
				}
			}

			listener, err := NewListener(1*time.Second, channel, false, mockFulfiller)
			if !checkError(tc.errMsg, err, t) {
				go listener.listen()

				select {
				case <-req.Ctx.Done():
					t.Errorf("context done not expected")
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
						t.Fatalf("did not call fulfill")
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
			}
		})
	}
}

func checkError(errMsg string, err error, t *testing.T) bool {
	if err != nil {
		if errMsg == "" {
			// here the testcase didn't expect any error
			t.Errorf("unexpected error: %v", err)
		} else if errMsg != err.Error() {
			// here the testcase expected another error than the received
			t.Errorf("expected error message: %v; got: %v", errMsg, err.Error())
		}
		return true
	}
	return false
}
