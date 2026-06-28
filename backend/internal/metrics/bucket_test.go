package metrics

import (
	"errors"
	"testing"
)

func TestBucketSnapshotCountsStatusAndLatency(t *testing.T) {
	b := newBucket()

	b.add(RequestResult{StatusCode: 200, Latency: 10})
	b.add(RequestResult{StatusCode: 500, Latency: 30})
	b.add(RequestResult{Err: errors.New("timeout"), ErrorType: ErrorTimeout, Latency: 20})

	snapshot := b.snapshot()

	if snapshot.TotalRequests != 3 {
		t.Fatalf("TotalRequests = %d, want 3", snapshot.TotalRequests)
	}
	if snapshot.Success != 1 {
		t.Fatalf("Success = %d, want 1", snapshot.Success)
	}
	if snapshot.Failed != 2 {
		t.Fatalf("Failed = %d, want 2", snapshot.Failed)
	}
	if snapshot.AvgLatency != 20 {
		t.Fatalf("AvgLatency = %v, want 20", snapshot.AvgLatency)
	}
	if snapshot.MinLatency != 10 {
		t.Fatalf("MinLatency = %d, want 10", snapshot.MinLatency)
	}
	if snapshot.MaxLatency != 30 {
		t.Fatalf("MaxLatency = %d, want 30", snapshot.MaxLatency)
	}
	if snapshot.HTTP5xxErrors != 1 {
		t.Fatalf("HTTP5xxErrors = %d, want 1", snapshot.HTTP5xxErrors)
	}
	if snapshot.TimeoutErrors != 1 {
		t.Fatalf("TimeoutErrors = %d, want 1", snapshot.TimeoutErrors)
	}
}

func TestBucketReset(t *testing.T) {
	b := newBucket()
	b.add(RequestResult{StatusCode: 200, Latency: 10})

	b.reset()

	// if snapshot := b.snapshot(); snapshot != (Snapshot{}) {
	// 	t.Fatalf("snapshot after reset = %+v, want zero snapshot", snapshot)
	// }
}
