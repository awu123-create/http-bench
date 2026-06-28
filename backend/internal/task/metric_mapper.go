package task

import (
	"encoding/json"
	"github.com/awu123-create/http-bench/internal/metrics"
	"github.com/awu123-create/http-bench/internal/storage/db"
	"time"
)

func errorsJSON(snapshot metrics.Snapshot) (string, error) {
	errorsMap := map[string]int64{
		"request_build":       snapshot.RequestBuildErrors,
		"timeout":             snapshot.TimeoutErrors,
		"canceled":            snapshot.CanceledErrors,
		"network":             snapshot.NetworkErrors,
		"dns":                 snapshot.DNSErrors,
		"connection_refused":  snapshot.ConnectionRefusedErrors,
		"connection_reset":    snapshot.ConnectionResetErrors,
		"too_many_open_files": snapshot.TooManyOpenFilesErrors,
		"read_body":           snapshot.ReadBodyErrors,
		"http_4xx":            snapshot.HTTP4xxErrors,
		"http_5xx":            snapshot.HTTP5xxErrors,
		"http_status":         snapshot.HTTPStatusErrors,
		"business":            snapshot.BusinessErrors,
		"client_overload":     snapshot.ClientOverloadErrors,
	}

	data, err := json.Marshal(errorsMap)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func errorsFromJSON(jsonStr string) map[string]int64 {
	var errorsMap map[string]int64
	if err := json.Unmarshal([]byte(jsonStr), &errorsMap); err != nil {
		return nil
	}
	return errorsMap
}

func statusCodeJSON(snapshot metrics.Snapshot) (string, error) {
	data, err := json.Marshal(snapshot.StatusCodes)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func statusCodeFromJSON(jsonStr string) map[int]int64 {
	var statusCodes map[int]int64
	if err := json.Unmarshal([]byte(jsonStr), &statusCodes); err != nil {
		return nil
	}
	return statusCodes
}

func metricBucketParams(testID int64, snapshot metrics.Snapshot, secondOffset int64) db.CreateMetricBucketParams {
	params := db.CreateMetricBucketParams{
		TestID:         testID,
		SecondOffset:   secondOffset,
		Requests:       snapshot.TotalRequests,
		Success:        snapshot.Success,
		Failure:        snapshot.Failed,
		AvgLatencyMs:   snapshot.AvgLatency,
		MinLatencyMs:   snapshot.MinLatency,
		MaxLatencyMs:   snapshot.MaxLatency,
		ErrorsJson:     "",
		StatusCodeJson: "",
		CreatedAt:      time.Now().Format(time.RFC3339Nano),
	}

	params.ErrorsJson, _ = errorsJSON(snapshot)
	params.StatusCodeJson, _ = statusCodeJSON(snapshot)
	return params
}
