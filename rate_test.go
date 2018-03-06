package throttler_test

import (
	"testing"
	"time"

	"bitbucket.org/differenttravel/pase-common/throttler"
)

func TestNewRateByCallsPerSecond(t *testing.T) {
	tt := []struct {
		name              string
		maxCallsPerSecond int
		guardTime         time.Duration
		errMsg            string
	}{
		{"Positive TC", 2, duration50ms, ""},
		{"Negative TC: maxCallsPerSecond zero", 0, duration50ms, "maxCalls must be greater than zero"},
		{"Negative TC: maxCallsPerSecond", -2, duration50ms, "maxCalls must be greater than zero"},
		{"Negative TC: guardTime", 2, -duration50ms, "guardTime must be greater or equal than zero"},
		{"Positive TC: guardTime zero", 2, 0, ""},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			_, err := throttler.NewRateByCallsPerSecond(tc.maxCallsPerSecond, tc.guardTime)
			checkError(tc.errMsg, err, t)
		})
	}
}

func TestNewRateByCallsPerMinute(t *testing.T) {
	tt := []struct {
		name              string
		maxCallsPerMinute int
		guardTime         time.Duration
		errMsg            string
	}{
		{"Positive TC", 2, duration50ms, ""},
		{"Negative TC: maxCallsPerMinute zero", 0, duration50ms, "maxCalls must be greater than zero"},
		{"Negative TC: maxCallsPerMinute", -2, duration50ms, "maxCalls must be greater than zero"},
		{"Negative TC: guardTime", 2, -duration50ms, "guardTime must be greater or equal than zero"},
		{"Positive TC: guardTime zero", 2, 0, ""},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			_, err := throttler.NewRateByCallsPerMinute(tc.maxCallsPerMinute, tc.guardTime)
			checkError(tc.errMsg, err, t)
		})
	}
}

func TestNewRateByCallsPerHour(t *testing.T) {
	tt := []struct {
		name            string
		maxCallsPerHour int
		guardTime       time.Duration
		errMsg          string
	}{
		{"Positive TC", 2, duration50ms, ""},
		{"Negative TC: maxCallsPerHour zero", 0, duration50ms, "maxCalls must be greater than zero"},
		{"Negative TC: maxCallsPerHour", -2, duration50ms, "maxCalls must be greater than zero"},
		{"Negative TC: guardTime", 2, -duration50ms, "guardTime must be greater or equal than zero"},
		{"Positive TC: guardTime zero", 2, 0, ""},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			_, err := throttler.NewRateByCallsPerHour(tc.maxCallsPerHour, tc.guardTime)
			checkError(tc.errMsg, err, t)
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
