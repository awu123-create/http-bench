package metrics

import (
	"testing"
	"time"
)

func TestRealtimeCollectorAggregatesSnapshots(t *testing.T) {
	collector := newRealtimeCollector(2, 10*time.Millisecond)
	defer collector.Close()

	collector.Record(RequestResult{StatusCode: 200, Latency: 10})
	collector.Record(RequestResult{StatusCode: 500, Latency: 30})

	deadline := time.After(200 * time.Millisecond)
	for {
		select {
		case snapshot := <-collector.Snapshots():
			if snapshot.TotalRequests == 0 {
				continue
			}
			if snapshot.TotalRequests != 2 {
				t.Fatalf("TotalRequests = %d, want 2", snapshot.TotalRequests)
			}
			if snapshot.Success != 1 {
				t.Fatalf("Success = %d, want 1", snapshot.Success)
			}
			if snapshot.Failed != 1 {
				t.Fatalf("Failed = %d, want 1", snapshot.Failed)
			}
			if snapshot.AvgLatency != 20 {
				t.Fatalf("AvgLatency = %v, want 20", snapshot.AvgLatency)
			}
			return
		case <-deadline:
			t.Fatal("timed out waiting for snapshot")
		}
	}
}

func TestRealtimeCollectorCloseFlushesCurrentBucket(t *testing.T) {
	collector := newRealtimeCollector(2, time.Hour)

	collector.Record(RequestResult{StatusCode: 200, Latency: 15})
	collector.Record(RequestResult{StatusCode: 201, Latency: 25})
	collector.Close()

	snapshot, ok := <-collector.Snapshots()
	if !ok {
		t.Fatal("snapshots channel closed before final snapshot")
	}
	if snapshot.TotalRequests != 2 {
		t.Fatalf("TotalRequests = %d, want 2", snapshot.TotalRequests)
	}
	if snapshot.Success != 2 {
		t.Fatalf("Success = %d, want 2", snapshot.Success)
	}
	if snapshot.AvgLatency != 20 {
		t.Fatalf("AvgLatency = %v, want 20", snapshot.AvgLatency)
	}

	if _, ok := <-collector.Snapshots(); ok {
		t.Fatal("snapshots channel should be closed after final snapshot")
	}
}

func TestShouldRecordFilter(t *testing.T) {
	cases := []struct {
		name   string
		result RequestResult
		want   bool
	}{
		{"success", RequestResult{StatusCode: 200, Latency: 5}, true},
		{"sub-ms success", RequestResult{StatusCode: 200, Latency: 0}, true},
		{"timeout error", RequestResult{Latency: 30, ErrorType: ErrorTimeout}, true},
		{"network error", RequestResult{Latency: 12, ErrorType: ErrorNetwork}, true},
		{"build error", RequestResult{Latency: 0, ErrorType: ErrorRequestBuild}, false},
		{"client overload", RequestResult{Latency: 0, ErrorType: ErrorClientOverload}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldRecord(tc.result); got != tc.want {
				t.Fatalf("shouldRecord(%+v) = %v, want %v", tc.result, got, tc.want)
			}
		})
	}
}

func TestRealtimeCollectorPercentilesAfterClose(t *testing.T) {
	collector := newRealtimeCollector(2, time.Hour)

	collector.Record(RequestResult{StatusCode: 200, Latency: 5})
	collector.Record(RequestResult{StatusCode: 200, Latency: 10})
	// 这条 build/overload 类结果（Latency=0, StatusCode=0）应被过滤，不进入直方图
	collector.Record(RequestResult{Latency: 0, ErrorType: ErrorRequestBuild})
	collector.Close()

	// 过滤后直方图仅含 2 个样本（5ms, 10ms）：
	// P50 target = int64(2*0.5)=1 → 累加到 5ms 时 cum=1 → 5
	// 若过滤失效（count=3，含 0ms），P50 target=int64(3*0.5)=1 → 0ms cum=1 → 0
	pcts := collector.Percentiles()
	if pcts.P50 != 5 {
		t.Fatalf("P50 = %d, want 5 (build-error result should be filtered out)", pcts.P50)
	}
}

func TestRealtimeCollectorPercentilesManySamples(t *testing.T) {
	collector := newRealtimeCollector(2, time.Hour)
	for i := int64(1); i <= 100; i++ {
		collector.Record(RequestResult{StatusCode: 200, Latency: i})
	}
	collector.Close()

	pcts := collector.Percentiles()
	if pcts.P50 != 50 {
		t.Fatalf("P50 = %d, want 50", pcts.P50)
	}
	if pcts.P90 != 90 {
		t.Fatalf("P90 = %d, want 90", pcts.P90)
	}
	if pcts.P95 != 95 {
		t.Fatalf("P95 = %d, want 95", pcts.P95)
	}
	if pcts.P99 != 99 {
		t.Fatalf("P99 = %d, want 99", pcts.P99)
	}
}
