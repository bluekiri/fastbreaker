package fastbreaker_test

import (
	"testing"
	"time"

	"github.com/bluekiri/fastbreaker"
)

func TestConfigurationNumBuckets(t *testing.T) {
	type testSpec struct {
		name   string
		args   int
		expect int
	}

	tests := []testSpec{
		{"-1", -1, fastbreaker.DefaultNumBuckets},
		{"0", 0, fastbreaker.DefaultNumBuckets},
		{"1", 1, 1},
		{"5", 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := fastbreaker.New(fastbreaker.Configuration{NumBuckets: tt.args})
			configuration := cb.Configuration()
			if configuration.NumBuckets != tt.expect {
				t.Errorf("expected %d buckets but got %d", tt.expect, configuration.NumBuckets)
			}
			cb.Stop()
		})
	}
}

func TestConfigurationBucketDuration(t *testing.T) {
	type testSpec struct {
		name   string
		args   time.Duration
		expect time.Duration
	}

	tests := []testSpec{
		{"-1s", -1 * time.Second, fastbreaker.DefaultBucketDuration},
		{"0s", 0 * time.Second, fastbreaker.DefaultBucketDuration},
		{"0.5s", 500 * time.Millisecond, fastbreaker.DefaultBucketDuration},
		{"1.5s", 1500 * time.Millisecond, 1 * time.Second},
		{"2s", 2 * time.Second, 2 * time.Second},
		{"2.5s", 2500 * time.Millisecond, 2 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := fastbreaker.New(fastbreaker.Configuration{BucketDuration: tt.args})
			configuration := cb.Configuration()
			if configuration.BucketDuration != tt.expect {
				t.Errorf("expected %d duration but got %d", tt.expect, configuration.BucketDuration)
			}
			cb.Stop()
		})
	}
}

func TestConfigurationDurationOfBreak(t *testing.T) {
	type testSpec struct {
		name   string
		args   time.Duration
		expect time.Duration
	}

	tests := []testSpec{
		{"-1s", -1 * time.Second, fastbreaker.DefaultDurationOfBreak},
		{"0s", 0 * time.Second, fastbreaker.DefaultDurationOfBreak},
		{"0.5s", 500 * time.Millisecond, fastbreaker.DefaultDurationOfBreak},
		{"1.5s", 1500 * time.Millisecond, 1 * time.Second},
		{"2s", 2 * time.Second, 2 * time.Second},
		{"2.5s", 2500 * time.Millisecond, 2 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := fastbreaker.New(fastbreaker.Configuration{DurationOfBreak: tt.args})
			configuration := cb.Configuration()
			if configuration.DurationOfBreak != tt.expect {
				t.Errorf("expected %d duration but got %d", tt.expect, configuration.DurationOfBreak)
			}
			cb.Stop()
		})
	}
}

func TestConfigurationShouldTrip(t *testing.T) {
	var customShouldTripCalled bool
	customShouldTrip := func(executions uint64, failures uint64) bool {
		customShouldTripCalled = true
		return false
	}

	type testSpec struct {
		name   string
		args   fastbreaker.ShouldTripFunc
		expect bool
	}

	tests := []testSpec{
		{"nil", nil, false},
		{"notnil", customShouldTrip, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cb := fastbreaker.New(fastbreaker.Configuration{ShouldTrip: tt.args})
			configuration := cb.Configuration()
			if configuration.ShouldTrip == nil {
				t.Errorf("ShouldTrip should not be nil")
			}

			configuration.ShouldTrip(0, 0)
			if customShouldTripCalled != tt.expect {
				t.Errorf("expected customShouldTripCalled %t but was %t", tt.expect, customShouldTripCalled)
			}
			cb.Stop()
		})
	}
}
