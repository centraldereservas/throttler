package throttler_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/centraldereservas/throttler"
)

var duration35min = 35 * time.Minute
var duration30min = 30 * time.Minute
var duration5min = 5 * time.Minute

var duration35s = 35 * time.Second
var duration30s = 30 * time.Second
var duration5s = 5 * time.Second
var duration550ms = 550 * time.Millisecond
var duration500ms = 500 * time.Millisecond
var duration50ms = 50 * time.Millisecond
var duration10s = 10 * time.Second
var duration1ns = 1 * time.Nanosecond

type MockRate struct {
	CalculateRateMock func() time.Duration
}

func (m *MockRate) CalculateRate() time.Duration {
	return m.CalculateRateMock()
}

func buildThrottler(rate throttler.Rate, maxCallsPerSecond int, guardTime time.Duration, requestChannelCapacity int, verbose bool, client *http.Client) (throttler.Limiter, error) {
	limiter, err := throttler.New(rate, requestChannelCapacity, verbose, client)
	if err != nil || limiter == nil {
		return nil, fmt.Errorf("unable to create a new throttler: %v", err)
	}
	return limiter, nil
}

func TestNew(t *testing.T) {
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
		{"Negative TC: new listener with errors", 2, 10, 10, "maxCalls must be greater than zero"},
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
				var client *http.Client
				throttler, err := throttler.New(rate, tc.reqChanCapacity, false, client)
				if throttler == nil {
					checkError(tc.errMsg, err, t)
				}
			}
		})
	}
}

func TestRate(t *testing.T) {
	tt := []struct {
		name         string
		expectedRate time.Duration
	}{
		{"Positive TC", 5 * time.Second},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			mockRate := &MockRate{
				CalculateRateMock: func() time.Duration {
					called = true
					return tc.expectedRate
				},
			}
			throttler, err := buildThrottler(mockRate, 2, duration50ms, 10, false, nil)
			if err != nil {
				t.Fatalf("unable to create a throttler")
			}
			rateDuration := throttler.Rate()
			if !called {
				t.Fatalf("did not call CalculateRate")
			}
			if rateDuration != tc.expectedRate {
				t.Errorf("expected rate duration %v; got %v", tc.expectedRate, rateDuration)
			}
			throttler.Run()
		})
	}
}

func TestRun(t *testing.T) {
	tt := []struct {
		name         string
		expectedRate time.Duration
	}{
		{"Positive TC", 5 * time.Second},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			mockRate := &MockRate{
				CalculateRateMock: func() time.Duration {
					called = true
					return tc.expectedRate
				},
			}

			throttler, err := buildThrottler(mockRate, 2, duration50ms, 10, false, nil)
			if err != nil {
				t.Fatalf("unable to create a throttler")
			}
			rateDuration := throttler.Rate()
			if !called {
				t.Fatalf("did not call CalculateRate")
			}
			if rateDuration != tc.expectedRate {
				t.Errorf("expected rate duration %v; got %v", tc.expectedRate, rateDuration)
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
			throttler, err := throttler.New(rate, tc.reqChanCapacity, tc.verbose, nil)

			if tc.startRequestHandler {
				throttler.Run()
			}
			if err != nil {
				t.Fatalf("unable to create throttler: %v", err)
			}
			ctx := context.Background()
			req, err := http.NewRequest("GET", "https://golang.org/", nil)
			if err != nil {
				t.Fatalf("unable to create request: %v", err)
			}
			res, err := throttler.Queue(ctx, tc.name, req, tc.timeout)
			if !checkError(tc.errMsg, err, t) {
				if res.StatusCode != http.StatusOK {
					t.Fatalf("expected http.StatusOK (200); got: %v", res.Status)
				}
			}
		})
	}
}

/*

func TestRun(t *testing.T) {
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

			// TODO: could we test the function Run() without instanciating the NewHandler?
			handler, err := throttler.New(rate, tc.reqChanCapacity, false, nil)
			if err != nil {
				t.Fatalf("unable to create a handler")
			}

			handler.Run()
			// TODO: how can we check that this was set correctly: h.requestHandlerStarted = true (if it is not public)
		})
	}
}


*/

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
