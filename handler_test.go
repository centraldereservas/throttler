package throttler_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"bitbucket.org/differenttravel/pase-common/throttler"
)

var duration500ms = 500 * time.Millisecond
var duration50ms = 50 * time.Millisecond
var duration10s = 10 * time.Second
var duration1ns = 1 * time.Nanosecond

func TestStartRequestsHandler(t *testing.T) {
	tt := []struct {
		name              string
		maxCallsPerSecond int
		guardTime         time.Duration
		reqChanCapacity   int
		errMsg            string
	}{
		{"Positive TC", 2, duration50ms, 5, ""},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			rate, err := throttler.NewRateByCallsPerSecond(tc.maxCallsPerSecond, tc.guardTime)
			if err != nil {
				t.Fatalf("unable to create a rate")
			}

			// TODO: could we test the function StartRequestsHandler() without instanciating the NewHandler?
			handler, err := throttler.NewHandler(rate, tc.reqChanCapacity, false)
			if err != nil {
				t.Fatalf("unable to create a handler")
			}

			handler.StartRequestsHandler()
			// TODO: how can we check that this was set correctly: h.requestHandlerStarted = true (if it is not public)
		})
	}
}

func TestSetClient(t *testing.T) {
	tt := []struct {
		name              string
		maxCallsPerSecond int
		client            *http.Client
	}{
		{name: "Positive TC", maxCallsPerSecond: 2, client: &http.Client{Timeout: 123 * time.Millisecond}},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			rate, err := throttler.NewRateByCallsPerSecond(tc.maxCallsPerSecond, duration50ms)
			if err != nil {
				t.Fatalf("unable to create a rate")
			}
			handler, err := throttler.NewHandler(rate, 10, false)
			if err != nil {
				t.Fatalf("unable to create a handler")
			}
			handler.SetClient(tc.client)
			// TODO: how to check that nothing went wrong?
		})
	}
}

func TestNewHandler(t *testing.T) {
	tt := []struct {
		name              string
		maxCallsPerSecond int
		guardTime         time.Duration
		reqChanCapacity   int
		errMsg            string
	}{
		{"Positive TC", 2, duration50ms, 5, ""},
		{"Positive TC: reqChanCapacity zero", 2, duration50ms, 0, ""},
		{"Negative TC: reqChanCapacity negative", 2, duration50ms, -5, "reqChanCapacity must be greater than zero"},
		{"Negative TC: rate nil", 0, 0, 10, "rate can not be nil"},
		{"Negative TC: maxCalls zero", 0, 2, 10, "maxCalls must be greater than zero"},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			var rate throttler.Rate
			var err error
			var foundError = false
			if tc.name != "Negative TC: rate nil" {
				rate, err = throttler.NewRateByCallsPerSecond(tc.maxCallsPerSecond, tc.guardTime)
				if err != nil {
					if checkError(tc.errMsg, err, t) {
						foundError = true
					}

				}
			}
			if !foundError {
				handler, err := throttler.NewHandler(rate, tc.reqChanCapacity, false)
				if handler == nil {
					checkError(tc.errMsg, err, t)
				}
			}
		})
	}
}

func TestQueue(t *testing.T) {
	tt := []struct {
		name                string
		maxCallsPerSecond   int
		guardTime           time.Duration
		reqChanCapacity     int
		verbose             bool
		timeout             time.Duration
		startRequestHandler bool
		errMsg              string
	}{
		{"Positive TC", 2, duration50ms, 5, true, duration10s, true, ""},
		{"Negative TC: requestHandler not started", 2, duration50ms, 5, false, duration10s, false, "requestHandler has not been started"},
		{"Negative TC: force timeout in Queue", 2, duration50ms, 5, true, duration1ns, true, "context deadline exceeded"},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			rate, err := throttler.NewRateByCallsPerSecond(tc.maxCallsPerSecond, tc.guardTime)
			if err != nil {
				t.Fatalf("unable to create a rate")
			}
			handler, err := throttler.NewHandler(rate, tc.reqChanCapacity, tc.verbose)

			if tc.startRequestHandler {
				handler.StartRequestsHandler()
			}
			if err != nil {
				t.Fatalf("unable to create handler: %v", err)
			}
			ctx := context.Background()
			req, err := http.NewRequest("GET", "https://golang.org/", nil)
			if err != nil {
				t.Fatalf("unable to create request: %v", err)
			}
			res, err := handler.Queue(ctx, tc.name, req, tc.timeout)
			if !checkError(tc.errMsg, err, t) {
				if res.StatusCode != http.StatusOK {
					t.Fatalf("expected http.StatusOK (200); got: %v", res.Status)
				}
			}
			/*defer res.Body.Close()
			bodyBytes, err := ioutil.ReadAll(res.Body)
			bodyString := string(bodyBytes)
			fmt.Print(bodyString)*/
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
