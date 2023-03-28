package fastbreaker

import "time"

const (
	// DefaultNumBuckets is the default number of buckets of a circuit breaker. Value = 10.
	DefaultNumBuckets = 10
	// DefaultBucketDuration is the default duration of a bucket. Value = 1s.
	DefaultBucketDuration = 1 * time.Second
	// DefaultDurationOfBreak is the default duration of a circuit breaker break. Value = 5s
	DefaultDurationOfBreak = 5 * time.Second
)

// DefaultShouldTrip is the default implementation of the ShouldTrip function.
// If will trip the circuit when there has been at least 20 executions and at least 50% of the
// executions failed.
func DefaultShouldTrip(executions uint64, failures uint64) bool {
	return executions >= 20 && failures*2 >= executions
}

// Configuration is a struct used to configure a circuit breaker.
type Configuration struct {
	NumBuckets      int
	BucketDuration  time.Duration
	DurationOfBreak time.Duration
	ShouldTrip      ShouldTripFunc
}

// A ShouldTripFunc tells the circuit breaker to trip when it returns true. If it returns false,
// the circuit breaker will remain closed.
type ShouldTripFunc func(executions uint64, failures uint64) bool
