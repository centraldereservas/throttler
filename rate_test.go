package throttler_test

import (
	"testing"
	"time"

	"github.com/centraldereservas/throttler"
)

func TestNewRateByCallsPerSecond(t *testing.T) {
	tt := []struct {
		name                 string
		maxCallsPerSecond    int
		guardTime            time.Duration
		expectedRateDuration time.Duration
		errMsg               string
	}{
		{"Positive TC", 2, duration50ms, duration550ms, ""},
		{"Negative TC: maxCallsPerSecond zero", 0, duration50ms, 0, "maxCalls must be greater than zero"},
		{"Negative TC: maxCallsPerSecond", -2, duration50ms, 0, "maxCalls must be greater than zero"},
		{"Negative TC: guardTime", 2, -duration50ms, 0, "guardTime must be greater or equal than zero"},
		{"Positive TC: guardTime zero", 2, 0, duration500ms, ""},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			rate, err := throttler.NewRateByCallsPerSecond(tc.maxCallsPerSecond, tc.guardTime)
			if !checkError(tc.errMsg, err, t) {
				rateDuration := rate.CalculateRate()
				if rateDuration != tc.expectedRateDuration {
					t.Errorf("expected rate duration %v; got %v", tc.expectedRateDuration, rateDuration)
				}
			}
		})
	}
}

func TestNewRateByCallsPerMinute(t *testing.T) {
	tt := []struct {
		name                 string
		maxCallsPerMinute    int
		guardTime            time.Duration
		expectedRateDuration time.Duration
		errMsg               string
	}{
		{"Positive TC", 2, duration5s, duration35s, ""},
		{"Negative TC: maxCallsPerMinute zero", 0, duration50ms, 0, "maxCalls must be greater than zero"},
		{"Negative TC: maxCallsPerMinute", -2, duration50ms, 0, "maxCalls must be greater than zero"},
		{"Negative TC: guardTime", 2, -duration50ms, 0, "guardTime must be greater or equal than zero"},
		{"Positive TC: guardTime zero", 2, 0, duration30s, ""},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			rate, err := throttler.NewRateByCallsPerMinute(tc.maxCallsPerMinute, tc.guardTime)
			if !checkError(tc.errMsg, err, t) {
				rateDuration := rate.CalculateRate()
				if rateDuration != tc.expectedRateDuration {
					t.Errorf("expected rate duration %v; got %v", tc.expectedRateDuration, rateDuration)
				}
			}
		})
	}
}

func TestNewRateByCallsPerHour(t *testing.T) {
	tt := []struct {
		name                 string
		maxCallsPerHour      int
		guardTime            time.Duration
		expectedRateDuration time.Duration
		errMsg               string
	}{
		{"Positive TC", 2, duration5min, duration35min, ""},
		{"Negative TC: maxCallsPerHour zero", 0, duration50ms, 0, "maxCalls must be greater than zero"},
		{"Negative TC: maxCallsPerHour", -2, duration50ms, 0, "maxCalls must be greater than zero"},
		{"Negative TC: guardTime", 2, -duration50ms, 0, "guardTime must be greater or equal than zero"},
		{"Positive TC: guardTime zero", 2, 0, duration30min, ""},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			rate, err := throttler.NewRateByCallsPerHour(tc.maxCallsPerHour, tc.guardTime)
			if !checkError(tc.errMsg, err, t) {
				rateDuration := rate.CalculateRate()
				if rateDuration != tc.expectedRateDuration {
					t.Errorf("expected rate duration %v; got %v", tc.expectedRateDuration, rateDuration)
				}
			}
		})
	}
}

func TestCalculateRate(t *testing.T) {
	tt := []struct {
		name              string
		maxCallsPerSecond int
		guardTime         time.Duration
		expectedRate      time.Duration
	}{
		{"Positive TC", 2, duration50ms, 550 * time.Millisecond},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			rate, err := throttler.NewRateByCallsPerSecond(tc.maxCallsPerSecond, duration50ms)
			if err != nil {
				t.Fatalf("unable to create a rate")
			}
			rateDuration := rate.CalculateRate()
			if rateDuration != tc.expectedRate {
				t.Errorf("unexpected rate duration, expected 550ms; got %v", rateDuration)
			}
		})
	}
}
