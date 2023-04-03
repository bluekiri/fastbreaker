package prometheus

import (
	"errors"
	"unicode/utf8"

	"github.com/bluekiri/fastbreaker"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	// MetricsNamespace is the common metric namespace (prefix).
	MetricsNamespace = "circuit_breaker"

	// ExecutionsMetricName is the suffix of the executions metric.
	ExecutionsMetricName = "executions_total"
	executionsMetricHelp = "Number of executions the circuit breaker allowed."

	// OpenStateMetricName is the suffix of the open metric.
	OpenStateMetricName = "open"
	openStateMetricHelp = "One if the circuit is not in the closed state."

	// SlidingFailureRateMetricName is the suffix of the sliding failure rate metric.
	SlidingFailureRateMetricName = "sliding_failure_rate"
	slidingFailureRateMetricHelp = "The sliding failure rate seen by the circuit breaker."

	// CircuitBreakerNameLabel is the label name for the circuit breaker name.
	CircuitBreakerNameLabel = "name"
	// ExecutionStatusLabel is the label name for the execution status.
	ExecutionStatusLabel = "status"
)

var ErrInvalidCircuitBreakerName = errors.New("invalid circuit breaker name")

// RegisterMetricsToDefaultRegisterer registers the FastBreaker metrics using the prometheus DefaultRegisterer.
// RegisterMetricsToDefaultRegisterer will label the FastBreaker metrics with the circuitBreakerName.
// RegisterMetricsToDefaultRegisterer will return an ErrInvalidCircuitBreakerName error if the circuitBreakerName string is not a valid utf-8 string.
func RegisterMetricsToDefaultRegisterer(circuitBreakerName string, cb fastbreaker.FastBreaker) (fastbreaker.FastBreaker, error) {
	return RegisterMetrics(circuitBreakerName, cb, prom.DefaultRegisterer)
}

// RegisterMetrics registers the FastBreaker metrics using the provided Registerer.
// RegisterMetrics will label the FastBreaker metrics with the circuitBreakerName.
// RegisterMetrics will return an ErrInvalidCircuitBreakerName error if the circuitBreakerName string is not a valid utf-8 string.
func RegisterMetrics(circuitBreakerName string, cb fastbreaker.FastBreaker, registerer prom.Registerer) (fastbreaker.FastBreaker, error) {
	return RegisterMetricsWithFactory(circuitBreakerName, cb, promauto.With(registerer))
}

// RegisterMetricsWithFactory registers the FastBreaker metrics using the provided Factory.
// RegisterMetricsWithFactory will label the FastBreaker metrics with the circuitBreakerName.
// RegisterMetricsWithFactory will return an ErrInvalidCircuitBreakerName error if the circuitBreakerName string is not a valid utf-8 string.
func RegisterMetricsWithFactory(circuitBreakerName string, cb fastbreaker.FastBreaker, factory promauto.Factory) (fastbreaker.FastBreaker, error) {
	if !utf8.ValidString(circuitBreakerName) {
		return nil, ErrInvalidCircuitBreakerName
	}
	circuitBreakerOpen(circuitBreakerName, cb, factory)
	slidingFailureRate(circuitBreakerName, cb, factory)
	executionsCounters(circuitBreakerName, cb, factory)

	return cb, nil
}

func circuitBreakerOpen(circuitBreakerName string, cb fastbreaker.FastBreaker, factory promauto.Factory) {
	factory.NewGaugeFunc(
		prom.GaugeOpts{
			Namespace:   MetricsNamespace,
			Name:        OpenStateMetricName,
			Help:        openStateMetricHelp,
			ConstLabels: prom.Labels{CircuitBreakerNameLabel: circuitBreakerName},
		},
		func() float64 {
			if cb.State() == fastbreaker.StateClosed {
				return 0.0
			}
			return 1.0
		},
	)
}

func slidingFailureRate(circuitBreakerName string, cb fastbreaker.FastBreaker, factory promauto.Factory) {
	factory.NewGaugeFunc(
		prom.GaugeOpts{
			Namespace:   MetricsNamespace,
			Name:        SlidingFailureRateMetricName,
			Help:        slidingFailureRateMetricHelp,
			ConstLabels: prom.Labels{CircuitBreakerNameLabel: circuitBreakerName},
		},
		func() float64 {
			executions, failures := cb.RollingCounters()
			if executions == 0 {
				return 0
			}
			return float64(failures) / float64(executions)
		},
	)
}

func executionsCounters(circuitBreakerName string, cb fastbreaker.FastBreaker, factory promauto.Factory) {
	factory.NewCounterFunc(
		prom.CounterOpts{
			Namespace:   MetricsNamespace,
			Name:        ExecutionsMetricName,
			Help:        executionsMetricHelp,
			ConstLabels: prom.Labels{CircuitBreakerNameLabel: circuitBreakerName, ExecutionStatusLabel: "success"},
		},
		func() float64 {
			return float64(cb.Executions() - cb.Failures())
		},
	)

	factory.NewCounterFunc(
		prom.CounterOpts{
			Namespace:   MetricsNamespace,
			Name:        ExecutionsMetricName,
			Help:        executionsMetricHelp,
			ConstLabels: prom.Labels{CircuitBreakerNameLabel: circuitBreakerName, ExecutionStatusLabel: "failure"},
		},
		func() float64 {
			return float64(cb.Failures())
		},
	)

	factory.NewCounterFunc(
		prom.CounterOpts{
			Namespace:   MetricsNamespace,
			Name:        ExecutionsMetricName,
			Help:        executionsMetricHelp,
			ConstLabels: prom.Labels{CircuitBreakerNameLabel: circuitBreakerName, ExecutionStatusLabel: "rejected"},
		},
		func() float64 {
			return float64(cb.Rejected())
		},
	)
}
