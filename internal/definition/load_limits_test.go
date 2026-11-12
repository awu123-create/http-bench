package definition_test

import (
	"math"
	"testing"
	"time"

	"github.com/awu123-create/http-bench/internal/definition"
)

func TestNewLoadLimitsAcceptsValidConfigurations(t *testing.T) {
	maxRequests := int64(250)
	limits, err := definition.NewLoadLimits(20, 50.5, 2*time.Minute, 5*time.Second, &maxRequests)
	if err != nil {
		t.Fatalf("NewLoadLimits() error = %v", err)
	}
	if limits.Concurrency() != 20 {
		t.Errorf("Concurrency() = %d, want 20", limits.Concurrency())
	}
	if limits.QPS() != 50.5 {
		t.Errorf("QPS() = %v, want 50.5", limits.QPS())
	}
	if limits.MaxDuration() != 2*time.Minute {
		t.Errorf("MaxDuration() = %v, want 2m", limits.MaxDuration())
	}
	if limits.RequestTimeout() != 5*time.Second {
		t.Errorf("RequestTimeout() = %v, want 5s", limits.RequestTimeout())
	}
	if limits.MaxRequests() == nil || *limits.MaxRequests() != maxRequests {
		t.Errorf("MaxRequests() = %v, want %d", limits.MaxRequests(), maxRequests)
	}

	withoutMaxRequests, err := definition.NewLoadLimits(1, 0.1, time.Nanosecond, time.Nanosecond, nil)
	if err != nil {
		t.Fatalf("NewLoadLimits() without maximum requests error = %v", err)
	}
	if withoutMaxRequests.MaxRequests() != nil {
		t.Errorf("MaxRequests() = %v, want nil", withoutMaxRequests.MaxRequests())
	}
}

func TestNewLoadLimitsRejectsInvalidConfigurations(t *testing.T) {
	zero := int64(0)
	negative := int64(-1)
	tests := []struct {
		name           string
		concurrency    int
		qps            float64
		maxDuration    time.Duration
		requestTimeout time.Duration
		maxRequests    *int64
	}{
		{name: "zero concurrency", concurrency: 0, qps: 1, maxDuration: time.Second, requestTimeout: time.Second},
		{name: "negative concurrency", concurrency: -1, qps: 1, maxDuration: time.Second, requestTimeout: time.Second},
		{name: "zero QPS", concurrency: 1, qps: 0, maxDuration: time.Second, requestTimeout: time.Second},
		{name: "negative QPS", concurrency: 1, qps: -1, maxDuration: time.Second, requestTimeout: time.Second},
		{name: "NaN QPS", concurrency: 1, qps: math.NaN(), maxDuration: time.Second, requestTimeout: time.Second},
		{name: "positive infinite QPS", concurrency: 1, qps: math.Inf(1), maxDuration: time.Second, requestTimeout: time.Second},
		{name: "negative infinite QPS", concurrency: 1, qps: math.Inf(-1), maxDuration: time.Second, requestTimeout: time.Second},
		{name: "zero maximum duration", concurrency: 1, qps: 1, maxDuration: 0, requestTimeout: time.Second},
		{name: "negative maximum duration", concurrency: 1, qps: 1, maxDuration: -time.Second, requestTimeout: time.Second},
		{name: "zero request timeout", concurrency: 1, qps: 1, maxDuration: time.Second, requestTimeout: 0},
		{name: "negative request timeout", concurrency: 1, qps: 1, maxDuration: time.Second, requestTimeout: -time.Second},
		{name: "zero maximum requests", concurrency: 1, qps: 1, maxDuration: time.Second, requestTimeout: time.Second, maxRequests: &zero},
		{name: "negative maximum requests", concurrency: 1, qps: 1, maxDuration: time.Second, requestTimeout: time.Second, maxRequests: &negative},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := definition.NewLoadLimits(tt.concurrency, tt.qps, tt.maxDuration, tt.requestTimeout, tt.maxRequests)
			if err == nil {
				t.Fatal("NewLoadLimits() error = nil, want validation error")
			}
		})
	}
}

func TestNewLoadLimitsDoesNotImposeArbitraryUpperBounds(t *testing.T) {
	maxRequests := int64(math.MaxInt64)
	limits, err := definition.NewLoadLimits(
		math.MaxInt,
		math.MaxFloat64,
		time.Duration(math.MaxInt64),
		time.Duration(math.MaxInt64),
		&maxRequests,
	)
	if err != nil {
		t.Fatalf("NewLoadLimits() error = %v", err)
	}
	if limits.Concurrency() != math.MaxInt || limits.QPS() != math.MaxFloat64 || *limits.MaxRequests() != maxRequests {
		t.Errorf("NewLoadLimits() did not preserve large finite values")
	}
}

func TestLoadLimitsCopiesMaxRequestsAtInputAndOutputBoundaries(t *testing.T) {
	input := int64(100)
	limits, err := definition.NewLoadLimits(1, 1, time.Second, time.Second, &input)
	if err != nil {
		t.Fatalf("NewLoadLimits() error = %v", err)
	}

	input = 200
	if got := limits.MaxRequests(); got == nil || *got != 100 {
		t.Fatalf("MaxRequests() after input mutation = %v, want 100", got)
	}

	output := limits.MaxRequests()
	*output = 300
	if got := limits.MaxRequests(); got == nil || *got != 100 {
		t.Errorf("MaxRequests() after output mutation = %v, want 100", got)
	}
}
