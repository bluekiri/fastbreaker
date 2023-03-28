package fastbreaker

import (
	"errors"
)

// ErrCircuitStopped is the error returned by FastCircuitBreaker.Allow() when the circuit is stopped.
var ErrCircuitStopped = errors.New("circuit breaker is stopped")

// ErrCircuitOpen is the error returned by FastCircuitBreaker.Allow() when the circuit is open.
var ErrCircuitOpen = errors.New("circuit breaker is open")

// FastBreaker is the interface implemented by the circuit breakers.
type FastBreaker interface {
	// Configuration returns the actual configuration used to create the circuit breaker.
	Configuration() Configuration

	// Stop releases the circuit breaker resources.
	Stop()

	// Allow checks if the circuit breaker should allow the execution to proceed.
	// Returns a function to report if the execution was successful when the execution is allowed or an
	// error when it is not.
	Allow() (func(bool), error)

	// State returns the current State of the circuit breaker.
	State() State

	// Executions returns the number of executions the circuit breaker has allowed.
	Executions() uint64

	// Failures returns the number of executions reported as failed.
	Failures() uint64

	// Rejected returns the number of executions the circuit breaker has rejected.
	Rejected() uint64

	// RollingCounters returns the rolling executions and failures.
	RollingCounters() (uint64, uint64)
}
