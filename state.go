package fastbreaker

import (
	"fmt"
)

// State represents the state of a circuit breaker.
type State uint32

func (state State) String() string {
	switch state {
	case StateStopped:
		return "stopped"
	case StateClosed:
		return "closed"
	case StateHalfOpen:
		return "half-open"
	case StateOpen:
		return "open"
	default:
		return fmt.Sprintf("unknown state %d", state)
	}
}

const (
	// StateStopped is the circuit breaker state when it is stopped.
	StateStopped State = iota
	// StateClosed is the circuit breaker state when it is allowing executions.
	StateClosed
	// StateHalfOpen is the circuit breaker state when it tests if it should remain open
	// or reset to the closed state.
	StateHalfOpen
	// StateOpen is the circuit breaker state when it is rejecting executions.
	StateOpen
)
