package fastbreaker

import (
	"container/ring"
	"sync/atomic"
	"time"
)

type counters struct {
	executions atomic.Uint64
	failures   atomic.Uint64
}

func (c *counters) reset() {
	c.executions.Store(0)
	c.failures.Store(0)
}

type fastBreaker struct {
	configuration   Configuration
	state           atomic.Value
	ring            *ring.Ring
	advanceTicker   *time.Ticker
	totalCounters   *counters
	rejected        atomic.Uint64
	breakTimer      *time.Timer
	halfOpenAllowed atomic.Bool
}

// New creates a new CircuitBreaker with the passed Configuration.
func New(configuration Configuration) FastBreaker {

	// Read configuration and applies default values.
	if configuration.NumBuckets <= 0 {
		configuration.NumBuckets = DefaultNumBuckets
	}

	configuration.BucketDuration = configuration.BucketDuration.Truncate(time.Second)
	if configuration.BucketDuration <= 0 {
		configuration.BucketDuration = DefaultBucketDuration
	}

	configuration.DurationOfBreak = configuration.DurationOfBreak.Truncate(time.Second)
	if configuration.DurationOfBreak <= 0 {
		configuration.DurationOfBreak = DefaultDurationOfBreak
	}

	if configuration.ShouldTrip == nil {
		configuration.ShouldTrip = DefaultShouldTrip
	}

	// Build the circuit breaker.
	cb := &fastBreaker{
		configuration: configuration,
		ring:          ring.New(int(configuration.NumBuckets)),
		advanceTicker: time.NewTicker(configuration.BucketDuration),
		totalCounters: &counters{},
	}
	cb.state.Store(StateStopped)

	// Initialize ring counter.
	for i := 0; i < cb.ring.Len(); i++ {
		cb.ring.Value = &counters{}
		cb.ring = cb.ring.Next()
	}

	// Reset the counters.
	cb.totalCounters.reset()
	cb.reset()

	// Start the advance window goroutine.
	go cb.advanceWindow()

	return cb
}

func (cb *fastBreaker) Configuration() Configuration {
	return cb.configuration
}

func (cb *fastBreaker) Stop() {
	cb.state.Store(StateStopped)
	if cb.breakTimer != nil {
		cb.breakTimer.Stop()
	}
	cb.advanceTicker.Stop()
}

func (cb *fastBreaker) Allow() (func(bool), error) {
	switch cb.state.Load() {
	case StateStopped:
		// Stopped states rejects all executions.
		return nil, ErrCircuitStopped
	case StateClosed:
		// Closed state allows all executions.
		return cb.buildFeedbackFunc(StateClosed), nil
	case StateHalfOpen:
		// Half-open state allows just one execution.
		if cb.halfOpenAllowed.CompareAndSwap(true, false) {
			return cb.buildFeedbackFunc(StateHalfOpen), nil
		}
	}
	// Reject other executions.
	cb.rejected.Add(1)
	return nil, ErrCircuitOpen
}

func (cb *fastBreaker) State() State {
	return cb.state.Load().(State)
}

func (cb *fastBreaker) Executions() uint64 {
	return cb.totalCounters.executions.Load()
}

func (cb *fastBreaker) Failures() uint64 {
	return cb.totalCounters.failures.Load()
}

func (cb *fastBreaker) Rejected() uint64 {
	return cb.rejected.Load()
}

func (cb *fastBreaker) RollingCounters() (uint64, uint64) {
	var executions uint64 = 0
	var failures uint64 = 0
	cb.ring.Do(func(value any) {
		counter := value.(*counters)
		executions += counter.executions.Load()
		failures += counter.failures.Load()
	})
	return executions, failures
}

func (cb *fastBreaker) buildFeedbackFunc(state State) func(bool) {
	return func(success bool) {
		cb.handleFeedback(state, success)
	}
}

func (cb *fastBreaker) handleFeedback(executionState State, success bool) {
	state := cb.state.Load()
	// Ignore feedback of executions allowed then the circuit was in a different state.
	if executionState != state {
		return
	}

	switch state {
	case StateClosed:
		cb.incExecutions()
		if !success {
			cb.incFailures()
			executions, failures := cb.RollingCounters()
			// check if the circuit breaker should trip
			if cb.configuration.ShouldTrip(executions, failures) {
				cb.tripFrom(StateClosed)
			}
		}
	case StateHalfOpen:
		if success {
			cb.reset()
		} else {
			cb.tripFrom(StateHalfOpen)
		}
	}
}

func (cb *fastBreaker) tripFrom(state State) bool {
	if cb.state.CompareAndSwap(state, StateOpen) {
		if cb.breakTimer == nil {
			// Create a timer that will transition the circuit from StateOpen to StateHalfOpen.
			cb.breakTimer = time.AfterFunc(
				cb.configuration.DurationOfBreak,
				func() {
					if cb.state.CompareAndSwap(StateOpen, StateHalfOpen) {
						cb.halfOpenAllowed.Store(true)
					}
					cb.breakTimer = nil
				},
			)
		}
		return true
	}
	return false
}

// reset resets the circuit breaker.
func (cb *fastBreaker) reset() {
	// stop the breakTimer.
	if cb.breakTimer != nil {
		cb.breakTimer.Stop()
	}

	// reset the rolling counters.
	if cb.state.Swap(StateClosed) != StateClosed {
		for i := 0; i < cb.ring.Len(); i++ {
			cb.ring.Value.(*counters).reset()
			cb.ring = cb.ring.Next()
		}
	}
}

// incExecutions increments the number of executions.
func (cb *fastBreaker) incExecutions() {
	cb.totalCounters.executions.Add(1)
	cb.ring.Value.(*counters).executions.Add(1)
}

// incFailures increments the number of failures.
func (cb *fastBreaker) incFailures() {
	cb.totalCounters.failures.Add(1)
	cb.ring.Value.(*counters).failures.Add(1)
}

// advanceWindow moves the ring to the next value.
func (cb *fastBreaker) advanceWindow() {
	for range cb.advanceTicker.C {
		next := cb.ring.Next()

		// reset the next counters.
		next.Value.(*counters).reset()

		// advance the ring.
		cb.ring = next
	}
}
