package prometheus_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bluekiri/fastbreaker"
	"github.com/bluekiri/fastbreaker/prometheus"
	prom "github.com/prometheus/client_golang/prometheus"
	client_model "github.com/prometheus/client_model/go"
)

func FuzzRegisterMetrics(f *testing.F) {
	f.Add("test", uint32(fastbreaker.StateClosed), uint64(0), uint64(0), uint64(0), uint64(0), uint64(0))
	f.Add("test2", uint32(fastbreaker.StateOpen), uint64(100), uint64(75), uint64(25), uint64(100), uint64(75))

	f.Fuzz(func(t *testing.T, cbName string, state uint32, executions uint64, failures uint64, rejected uint64, rollingExecutions uint64, rollingFailures uint64) {
		registry := prom.NewRegistry()

		// Register the circuit breaker.
		cb, err := prometheus.RegisterMetrics(
			cbName,
			&mockCircuitBreaker{
				state:             fastbreaker.State(state),
				executions:        executions,
				failures:          failures,
				rejected:          rejected,
				rollingExecutions: rollingExecutions,
				rollingFailures:   rollingFailures,
			},
			registry)

		// Assert the registration was successful
		if err == prometheus.ErrInvalidCircuitBreakerName {
			t.Skip("circuit breaker names ")
		}
		if err != nil {
			t.Error("RegisterMetrics should not return an error")
		}

		// Gether the metrics
		metricFamilies, err := registry.Gather()
		if err != nil {
			t.Fatalf("registerer.Gather() should not return an error.")
		}

		if len(metricFamilies) == 0 {
			t.Fatalf("registerer.Gather() should return metrics.")
		}

		// Parse the gathered metrics
		for _, metricFamily := range metricFamilies {
			// All metrics should have a common prefix
			if !strings.HasPrefix(metricFamily.GetName(), prometheus.MetricsNamespace) {
				t.Errorf("metric name %s does not start with %s.", metricFamily.GetName(), prometheus.MetricsNamespace)
			}

			switch metricFamily.GetName() {
			case prom.BuildFQName(prometheus.MetricsNamespace, "", prometheus.ExecutionsMetricName):
				// The metric should be a counter
				if metricFamily.GetType() != client_model.MetricType_COUNTER {
					t.Errorf("%s should be a counter", metricFamily.GetName())
				}

				// The metric should have the CircuitBreakerName label
				assertCircuitBreakerLabel(t, metricFamily, cbName)

				for _, metric := range metricFamily.Metric {
					// The metric should have the ExecutionStatusLabel label
					statusLabelValue, err := getLabelValue(metric, prometheus.ExecutionStatusLabel)
					if err != nil {
						t.Error(err.Error())
					}
					// Validate the metrics value
					switch statusLabelValue {
					case "success":
						assertMetric(t, metricFamily, metric.GetCounter().GetValue(), float64(cb.Executions()-cb.Failures()))
					case "failure":
						assertMetric(t, metricFamily, metric.GetCounter().GetValue(), float64(cb.Failures()))
					case "rejected":
						assertMetric(t, metricFamily, metric.GetCounter().GetValue(), float64(cb.Rejected()))
					default:
						t.Errorf("unexpected metric %s", metric.String())
					}
				}
			case prom.BuildFQName(prometheus.MetricsNamespace, "", prometheus.OpenStateMetricName):
				// The metric should be a gauge
				if metricFamily.GetType() != client_model.MetricType_GAUGE {
					t.Errorf("%s should be a gauge", metricFamily.GetName())
				}

				// The metric should have the CircuitBreakerName label
				assertCircuitBreakerLabel(t, metricFamily, cbName)

				// Validate the metrics value
				expectedOpenCircuits := 0.0
				if cb.State() != fastbreaker.StateClosed {
					expectedOpenCircuits = 1.0
				}
				assertMetric(t, metricFamily, metricFamily.Metric[0].GetGauge().GetValue(), expectedOpenCircuits)
			case prom.BuildFQName(prometheus.MetricsNamespace, "", prometheus.SlidingFailureRateMetricName):
				// The metric should be a gauge
				if metricFamily.GetType() != client_model.MetricType_GAUGE {
					t.Errorf("%s should be a gauge", metricFamily.GetName())
				}

				// The metric should have the CircuitBreakerName label
				assertCircuitBreakerLabel(t, metricFamily, cbName)

				// Validate the metrics value
				expectedRollingExecutions, expectedRollingFailures := cb.RollingCounters()
				expectedFailureRate := 0.0
				if expectedRollingExecutions > 0 {
					expectedFailureRate = float64(expectedRollingFailures) / float64(expectedRollingExecutions)
				}
				assertMetric(t, metricFamily, metricFamily.Metric[0].GetGauge().GetValue(), expectedFailureRate)
			default:
				t.Errorf("unexpected metric %s", metricFamily.GetName())
			}
		}
	})
}

func assertCircuitBreakerLabel(t *testing.T, metricFamily *client_model.MetricFamily, cbName string) {
	t.Helper()
	for _, metric := range metricFamily.GetMetric() {
		labelValue, err := getLabelValue(metric, prometheus.CircuitBreakerNameLabel)
		if err != nil {
			t.Error(err.Error())
		}
		if labelValue != cbName {
			t.Errorf("actual label %s value %s is not the expected %s", prometheus.CircuitBreakerNameLabel, labelValue, cbName)
		}
	}
}

func assertMetric(t *testing.T, metricFamily *client_model.MetricFamily, actualValue float64, expectedValue float64) {
	t.Helper()
	if actualValue != expectedValue {
		t.Errorf(
			"actual metric %s value %f is not the expected %f",
			metricFamily.GetName(),
			actualValue,
			expectedValue,
		)
	}
}

func getLabelValue(metric *client_model.Metric, labelName string) (string, error) {
	for _, label := range metric.GetLabel() {
		if label.GetName() == labelName {
			return label.GetValue(), nil
		}
	}
	return "", fmt.Errorf("label %s not found", labelName)
}

type mockCircuitBreaker struct {
	state             fastbreaker.State
	executions        uint64
	failures          uint64
	rejected          uint64
	rollingExecutions uint64
	rollingFailures   uint64
}

// Stop implements fastbreaker.FastBreaker
func (m *mockCircuitBreaker) Stop() {
	panic("unimplemented")
}

// Allow implements fastbreaker.FastBreaker
func (m *mockCircuitBreaker) Allow() (func(bool), error) {
	panic("unimplemented")
}

// Configuration implements fastbreaker.FastBreaker
func (m *mockCircuitBreaker) Configuration() fastbreaker.Configuration {
	panic("unimplemented")
}

// State implements fastbreaker.FastBreaker
func (m *mockCircuitBreaker) State() fastbreaker.State {
	return m.state
}

// Executions implements fastbreaker.FastBreaker
func (m *mockCircuitBreaker) Executions() uint64 {
	return m.executions
}

// Failures implements fastbreaker.FastBreaker
func (m *mockCircuitBreaker) Failures() uint64 {
	return m.failures
}

// Rejected implements fastbreaker.FastBreaker
func (m *mockCircuitBreaker) Rejected() uint64 {
	return m.rejected
}

// RollingCounters implements fastbreaker.FastBreaker
func (m *mockCircuitBreaker) RollingCounters() (uint64, uint64) {
	return m.rollingExecutions, m.rollingFailures
}
