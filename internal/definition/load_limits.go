package definition

import (
	"fmt"
	"math"
	"time"
)

type LoadLimits struct {
	concurrency    int
	qps            float64
	maxDuration    time.Duration
	requestTimeout time.Duration
	maxRequests    *int64
}

// NewLoadLimits validates the limits and copies the optional request count.
func NewLoadLimits(
	concurrency int,
	qps float64,
	maxDuration time.Duration,
	requestTimeout time.Duration,
	maxRequests *int64,
) (LoadLimits, error) {
	if concurrency <= 0 {
		return LoadLimits{}, fmt.Errorf("concurrency must be greater than zero")
	}
	if qps <= 0 || math.IsNaN(qps) || math.IsInf(qps, 0) {
		return LoadLimits{}, fmt.Errorf("QPS must be finite and greater than zero")
	}
	if maxDuration <= 0 {
		return LoadLimits{}, fmt.Errorf("maximum duration must be greater than zero")
	}
	if requestTimeout <= 0 {
		return LoadLimits{}, fmt.Errorf("request timeout must be greater than zero")
	}
	if maxRequests != nil && *maxRequests <= 0 {
		return LoadLimits{}, fmt.Errorf("maximum requests must be greater than zero when set")
	}

	var maxRequestsCopy *int64
	if maxRequests != nil {
		value := *maxRequests
		maxRequestsCopy = &value
	}

	return LoadLimits{
		concurrency:    concurrency,
		qps:            qps,
		maxDuration:    maxDuration,
		requestTimeout: requestTimeout,
		maxRequests:    maxRequestsCopy,
	}, nil
}

// Concurrency returns the maximum number of in-flight requests.
func (l LoadLimits) Concurrency() int {
	return l.concurrency
}

// QPS returns the request start rate limit.
func (l LoadLimits) QPS() float64 {
	return l.qps
}

// MaxDuration returns the maximum duration for starting requests.
func (l LoadLimits) MaxDuration() time.Duration {
	return l.maxDuration
}

func (l LoadLimits) RequestTimeout() time.Duration {
	return l.requestTimeout
}

// MaxRequests returns an independent copy of the optional request count limit.
func (l LoadLimits) MaxRequests() *int64 {
	if l.maxRequests == nil {
		return nil
	}
	value := *l.maxRequests
	return &value
}
