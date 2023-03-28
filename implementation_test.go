package fastbreaker_test

import (
	"testing"
	"time"

	"github.com/bluekiri/fastbreaker"
)

func TestStoppedCircuitBreaker(t *testing.T) {
	// Close the circuit breaker.
	cb := fastbreaker.New(fastbreaker.Configuration{})
	cb.Stop()

	if cb.State() != fastbreaker.StateStopped {
		t.Error("a stopped circuit breaker should return a StateStopped state.")
	}

	_, err := cb.Allow()
	if err == nil {
		t.Error("a stopped circuit breaker should not allow executions.")
	}

	if err != fastbreaker.ErrCircuitStopped {
		t.Error("a stopped circuit breaker should an ErrCircuitStopped error.")
	}
}

func TestCircuitBreaker(t *testing.T) {
	const numExecutions = 1_000

	// ShouldTripFunc wrapping the default implementation
	shouldTripExecutions := 0
	shouldTripWrapper := func(executions uint64, failures uint64) bool {
		shouldTripExecutions++
		return fastbreaker.DefaultShouldTrip(executions, failures)
	}

	// Build circuit breaker and assert initial state.
	cb := fastbreaker.New(fastbreaker.Configuration{
		DurationOfBreak: 1 * time.Second,
		ShouldTrip:      shouldTripWrapper,
	})
	defer cb.Stop()
	assertStateAndCounters(t, cb, fastbreaker.StateClosed, 0, 0)

	// Perform numExecutions successful executions and assert state.
	for i := 0; i < numExecutions; i++ {
		report := allowAndAssert(t, cb, true)
		report(true)
		if shouldTripExecutions != 0 {
			t.Fatalf("expected 0 executions of the ShouldTripFunc but got %d", shouldTripExecutions)
		}
	}
	totalExecutions := numExecutions
	assertStateAndCounters(t, cb, fastbreaker.StateClosed, totalExecutions, 0)

	// Perform numExecutions - 1 failed executions. Failure rate should be just 1 failure short
	// before reaching 50%.
	const numFailures = numExecutions - 1
	for i := 0; i < numFailures; i++ {
		report := allowAndAssert(t, cb, true)
		report(false)
		if shouldTripExecutions != i+1 {
			t.Fatalf("expected %d executions of the ShouldTripFunc but got %d", i+1, shouldTripExecutions)
		}
	}
	totalExecutions += numFailures
	totalFailures := numFailures
	assertStateAndCounters(t, cb, fastbreaker.StateClosed, totalExecutions, totalFailures)

	// One more failure should trip the circuit breaker.
	feedback := allowAndAssert(t, cb, true)
	feedback(false)
	totalExecutions++
	totalFailures++
	if shouldTripExecutions != totalFailures {
		t.Fatalf("expected %d executions of the ShouldTripFunc but got %d", totalFailures, shouldTripExecutions)
	}
	assertStateAndCounters(t, cb, fastbreaker.StateOpen, totalExecutions, totalFailures)

	// Keep a reference to the last feedback function when in the open state.
	openFeedbackFunc := feedback

	// The circuit breaker should remain open for DurationOfBreak and then should change to half-open.
	// During this time no execution should be allowed.
	assertStateOpenForDurationOfBreak(t, cb, openFeedbackFunc)
	if shouldTripExecutions != totalFailures {
		t.Fatalf("expected %d executions of the ShouldTripFunc but got %d", totalFailures, shouldTripExecutions)
	}
	assertStateAndCounters(t, cb, fastbreaker.StateHalfOpen, totalExecutions, totalFailures)

	// The circuit breaker should allow only one execution when in the half-open state.
	feedback = allowAndAssert(t, cb, true)
	assertStateAndCounters(t, cb, fastbreaker.StateHalfOpen, totalExecutions, totalFailures)
	allowAndAssert(t, cb, false)
	assertStateAndCounters(t, cb, fastbreaker.StateHalfOpen, totalExecutions, totalFailures)

	// A late success feedback should not close the circuit.
	openFeedbackFunc(true)
	assertStateAndCounters(t, cb, fastbreaker.StateHalfOpen, totalExecutions, totalFailures)

	// A failure should make the circuit breaker to open again for DurationOfBreak.
	feedback(false)
	if shouldTripExecutions != totalFailures {
		t.Fatalf("expected %d executions of the ShouldTripFunc but got %d", totalFailures, shouldTripExecutions)
	}
	assertStateAndCounters(t, cb, fastbreaker.StateOpen, totalExecutions, totalFailures)

	assertStateOpenForDurationOfBreak(t, cb, feedback)
	if shouldTripExecutions != totalFailures {
		t.Fatalf("expected %d executions of the ShouldTripFunc but got %d", totalFailures, shouldTripExecutions)
	}
	assertStateAndCounters(t, cb, fastbreaker.StateHalfOpen, totalExecutions, totalFailures)

	// The circuit breaker should allow again only one execution when in the half-open state.
	feedback = allowAndAssert(t, cb, true)
	assertStateAndCounters(t, cb, fastbreaker.StateHalfOpen, totalExecutions, totalFailures)
	allowAndAssert(t, cb, false)
	assertStateAndCounters(t, cb, fastbreaker.StateHalfOpen, totalExecutions, totalFailures)

	// A success should make the circuit breaker to reset.
	feedback(true)
	if shouldTripExecutions != totalFailures {
		t.Fatalf("expected %d executions of the ShouldTripFunc but got %d", totalFailures, shouldTripExecutions)
	}
	assertStateAndCounters(t, cb, fastbreaker.StateClosed, totalExecutions, totalFailures)

	// The circuit breaker should not change to Half-Open after DurationOfBreak.
	time.Sleep(cb.Configuration().DurationOfBreak)
	assertStateAndCounters(t, cb, fastbreaker.StateClosed, totalExecutions, totalFailures)
	time.Sleep(cb.Configuration().DurationOfBreak)
	assertStateAndCounters(t, cb, fastbreaker.StateClosed, totalExecutions, totalFailures)
}

func TestRollingCounters(t *testing.T) {
	cb := fastbreaker.New(fastbreaker.Configuration{})
	configuration := cb.Configuration()

	time.Sleep(configuration.BucketDuration / 2)

	// generate a successful execution in every bucket.
	for i := 1; i <= configuration.NumBuckets; i++ {
		feedback := allowAndAssert(t, cb, true)
		feedback(true)
		assertRollingCounters(t, cb, i, 0)
		time.Sleep(configuration.BucketDuration)
	}

	// check that after every BucketDuration an execution is removed.
	for i := configuration.NumBuckets - 1; i > 0; i-- {
		assertRollingCounters(t, cb, i, 0)
		time.Sleep(configuration.BucketDuration)
	}

	cb.Stop()
}

func allowAndAssert(t *testing.T, cb fastbreaker.FastBreaker, allowed bool) func(bool) {
	t.Helper()

	report, err := cb.Allow()
	if allowed != (err == nil) {
		if allowed {
			t.Fatal("executions should be allowed.")
		} else {
			t.Fatal("executions should not be allowed.")
		}
	}

	return report
}

func assertStateOpenForDurationOfBreak(t *testing.T, cb fastbreaker.FastBreaker, feedbackFunc func(bool)) {
	t.Helper()

	tripedAt := time.Now()
	prevRejected := cb.Rejected()
	for cb.State() == fastbreaker.StateOpen {
		// Executions should not be allowed when in the open state.
		allowAndAssert(t, cb, false)

		// Feedback should be ignored during the open state.
		feedbackFunc(true)
		feedbackFunc(false)

		// Rejected executions counter should increment.
		rejected := cb.Rejected()
		if rejected != prevRejected+1 {
			t.Fatalf("%d rejected executions expected gut got %d instead.", prevRejected+1, rejected)
		}
		prevRejected = rejected

		time.Sleep(1 * time.Millisecond)
	}

	if time.Since(tripedAt) < cb.Configuration().DurationOfBreak {
		t.Fatal("circuit should remain open for DurationOfBreak.")
	}
}

func assertStateAndCounters(t *testing.T, cb fastbreaker.FastBreaker, expectedState fastbreaker.State, expectedExecutions int, expectedFailures int) {
	t.Helper()

	actualState := cb.State()
	if actualState != expectedState {
		t.Fatalf("cirbuit breaker should be %s gut it is %s.", expectedState, actualState)
	}

	actualExecutions := cb.Executions()
	if actualExecutions != uint64(expectedExecutions) {
		t.Fatalf("%d executions expected but got %d instead.", expectedExecutions, actualExecutions)
	}

	actualFailures := cb.Failures()
	if actualFailures != uint64(expectedFailures) {
		t.Fatalf("%d failures expected but got %d instead.", expectedFailures, actualFailures)
	}
}

func assertRollingCounters(t *testing.T, cb fastbreaker.FastBreaker, expectedExecutions int, expectedFailures int) {
	t.Helper()

	actualExecutions, actualFailures := cb.RollingCounters()
	if actualFailures != uint64(expectedFailures) {
		t.Fatalf("%d failures expected instead of %d", expectedFailures, actualFailures)
	}

	if actualExecutions != uint64(expectedExecutions) {
		t.Fatalf("%d executions expected instead of %d.", expectedExecutions, actualExecutions)
	}
}
