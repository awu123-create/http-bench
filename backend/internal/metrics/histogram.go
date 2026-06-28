package metrics

const maxlatencyMs = 60000

type latencyHistogram struct {
	buckets []int64
	count   int64
}

func newLatencyHistogram() *latencyHistogram {
	return &latencyHistogram{
		buckets: make([]int64, maxlatencyMs+1),
	}
}

func (h *latencyHistogram) Record(latencyMs int64) {
	if latencyMs < 0 {
		latencyMs = 0
	}
	if latencyMs > maxlatencyMs {
		latencyMs = maxlatencyMs
	}
	h.buckets[latencyMs]++
	h.count++
}

func (h *latencyHistogram) Percentile(percent float64) int64 {
	if h.count == 0 {
		return 0
	}
	target := int64(float64(h.count) * percent)
	if target < 1 {
		target = 1
	}

	var cnt int64
	for i, c := range h.buckets {
		cnt += c
		if cnt >= target {
			return int64(i)
		}
	}
	return maxlatencyMs
}

type Percentiles struct {
	P99, P95, P90, P50 int64
}

func (h *latencyHistogram) Percentiles() Percentiles {
	return Percentiles{
		P99: h.Percentile(0.99),
		P95: h.Percentile(0.95),
		P90: h.Percentile(0.90),
		P50: h.Percentile(0.50),
	}
}
