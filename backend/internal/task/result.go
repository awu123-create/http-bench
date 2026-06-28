package task

import (
	"context"
	"time"

	"github.com/awu123-create/http-bench/internal/metrics"
	"github.com/awu123-create/http-bench/internal/storage/db"
)

func (m *Manager) createFinalResult(ctx context.Context, taskID int64, percentiles metrics.Percentiles) error {
	buckets, err := m.queries.ListMetricsBucketsByTestID(ctx, taskID)
	if err != nil {
		return err
	}

	summary := metrics.Snapshot{
		StatusCodes: make(map[int]int64),
	}
	var latencySum float64
	var hasLatency bool

	for _, bucket := range buckets {
		summary.TotalRequests += bucket.Requests
		summary.Success += bucket.Success
		summary.Failed += bucket.Failure

		if bucket.Requests > 0 {
			latencySum += bucket.AvgLatencyMs * float64(bucket.Requests)
			if !hasLatency || bucket.MinLatencyMs < summary.MinLatency {
				summary.MinLatency = bucket.MinLatencyMs
			}
			if bucket.MaxLatencyMs > summary.MaxLatency {
				summary.MaxLatency = bucket.MaxLatencyMs
			}
			hasLatency = true
		}

		for k, v := range errorsFromJSON(bucket.ErrorsJson) {
			switch k {
			case "request_build":
				summary.RequestBuildErrors += v
			case "timeout":
				summary.TimeoutErrors += v
			case "canceled":
				summary.CanceledErrors += v
			case "network":
				summary.NetworkErrors += v
			case "dns":
				summary.DNSErrors += v
			case "connection_refused":
				summary.ConnectionRefusedErrors += v
			case "connection_reset":
				summary.ConnectionResetErrors += v
			case "too_many_open_files":
				summary.TooManyOpenFilesErrors += v
			case "read_body":
				summary.ReadBodyErrors += v
			case "http_4xx":
				summary.HTTP4xxErrors += v
			case "http_5xx":
				summary.HTTP5xxErrors += v
			case "http_status":
				summary.HTTPStatusErrors += v
			case "business":
				summary.BusinessErrors += v
			case "client_overload":
				summary.ClientOverloadErrors += v
			}
		}

		for code, count := range statusCodeFromJSON(bucket.StatusCodeJson) {
			summary.StatusCodes[code] += count
		}
	}

	if summary.TotalRequests > 0 {
		summary.AvgLatency = latencySum / float64(summary.TotalRequests)
	}
	errorsJSON, err := errorsJSON(summary)
	if err != nil {
		return err
	}
	statusCodeJSON, err := statusCodeJSON(summary)
	if err != nil {
		return err
	}

	_, err = m.queries.CreateTestResult(ctx, db.CreateTestResultParams{
		TestID:         taskID,
		TotalRequests:  summary.TotalRequests,
		TotalSuccess:   summary.Success,
		TotalFailure:   summary.Failed,
		AvgLatencyMs:   summary.AvgLatency,
		MinLatencyMs:   summary.MinLatency,
		MaxLatencyMs:   summary.MaxLatency,
		ErrorsJson:     errorsJSON,
		StatusCodeJson: statusCodeJSON,
		CreatedAt:      time.Now().Format(time.RFC3339Nano),

		P99LatencyMs: percentiles.P99,
		P95LatencyMs: percentiles.P95,
		P90LatencyMs: percentiles.P90,
		P50LatencyMs: percentiles.P50,
	})
	return err
}
