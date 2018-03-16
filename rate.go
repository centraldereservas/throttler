package throttler

import (
	"fmt"
	"time"
)

// Rate is the interface that contains the CalculateRate method.
//
// CalculateRate returns the calculated rate duration used by the listener.
type Rate interface {
	CalculateRate() time.Duration
}

type rate struct {
	Period    time.Duration
	GuardTime time.Duration
}

// CalculateRate calculates the request rate as the period + guardTime
func (r *rate) CalculateRate() time.Duration {
	return r.Period + r.GuardTime
}

// NewRateByCallsPerSecond initializes the Rate based on the maxCallsPerSecond
func NewRateByCallsPerSecond(maxCallsPerSecond int, guardTime time.Duration) (Rate, error) {
	return newRate(maxCallsPerSecond, guardTime, time.Second)
}

// NewRateByCallsPerMinute initializes the Rate based on the maxCallsPerMin
func NewRateByCallsPerMinute(maxCallsPerMin int, guardTime time.Duration) (Rate, error) {
	return newRate(maxCallsPerMin, guardTime, time.Minute)
}

// NewRateByCallsPerHour initializes the Rate based on the maxCallsPerHour
func NewRateByCallsPerHour(maxCallsPerHour int, guardTime time.Duration) (Rate, error) {
	return newRate(maxCallsPerHour, guardTime, time.Hour)
}

func newRate(maxCalls int, guardTime time.Duration, timeReference time.Duration) (Rate, error) {
	if maxCalls <= 0 {
		return nil, fmt.Errorf("maxCalls must be greater than zero")
	}
	if guardTime.Nanoseconds() < 0 {
		return nil, fmt.Errorf("guardTime must be greater or equal than zero")
	}
	return &rate{
		Period:    timeReference / time.Duration(maxCalls),
		GuardTime: guardTime,
	}, nil
}
