package metrics

import "testing"

func TestLatencyHistogramPercentiles(t *testing.T) {
	h := newLatencyHistogram()
	for i := int64(1); i <= 100; i++ {
		h.Record(i)
	}

	pcts := h.Percentiles()
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

func TestLatencyHistogramClampOverflow(t *testing.T) {
	h := newLatencyHistogram()
	h.Record(70000) // 超过 60s 上界，应被钳到 60000

	pcts := h.Percentiles()
	if pcts.P50 != maxlatencyMs {
		t.Fatalf("P50 = %d, want %d (clamped)", pcts.P50, maxlatencyMs)
	}
	if pcts.P99 != maxlatencyMs {
		t.Fatalf("P99 = %d, want %d (clamped)", pcts.P99, maxlatencyMs)
	}
}

func TestLatencyHistogramEmpty(t *testing.T) {
	h := newLatencyHistogram()
	pcts := h.Percentiles()
	if pcts.P50 != 0 || pcts.P90 != 0 || pcts.P95 != 0 || pcts.P99 != 0 {
		t.Fatalf("empty histogram percentiles = %+v, want all 0", pcts)
	}
}

func TestLatencyHistogramNegativeLatency(t *testing.T) {
	h := newLatencyHistogram()
	h.Record(-5)
	h.Record(10)

	pcts := h.Percentiles()
	if pcts.P50 != 0 {
		t.Fatalf("P50 = %d, want 0 (negative clamped to 0)", pcts.P50)
	}
}
