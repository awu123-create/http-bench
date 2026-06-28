package metrics

import (
	"sync"
	"time"
)

type RealtimeCollector struct {
	results   chan RequestResult // 接收请求结果的通道
	snapshots chan Snapshot      // 存放聚合结果的通道
	done      chan struct{}      // 控制收集器生命周期
	hist      *latencyHistogram  // 延迟直方图，用于计算分位数
	once      sync.Once
	wg        sync.WaitGroup
}

func NewRealtimeCollector(qps int) *RealtimeCollector {
	return newRealtimeCollector(qps, time.Second)
}

func newRealtimeCollector(qps int, interval time.Duration) *RealtimeCollector {
	if qps <= 0 {
		qps = 1
	}
	if interval <= 0 {
		interval = time.Second
	}

	c := &RealtimeCollector{
		results:   make(chan RequestResult, qps*2), // 缓冲区大小为 QPS 的两倍
		snapshots: make(chan Snapshot, qps),
		done:      make(chan struct{}),
		hist:      newLatencyHistogram(),
	}

	c.wg.Add(1)
	go c.run(interval)
	return c
}

func (c *RealtimeCollector) Record(result RequestResult) {
	c.results <- result
}

func (c *RealtimeCollector) Snapshots() <-chan Snapshot {
	return c.snapshots
}

func (c *RealtimeCollector) Percentiles() Percentiles {
	return c.hist.Percentiles()
}

func shouldRecord(result RequestResult) bool {
	return result.Latency > 0 || result.StatusCode > 0
}

func (c *RealtimeCollector) ingest(b *bucket, result RequestResult) {
	b.add(result)
	if shouldRecord(result) {
		c.hist.Record(result.Latency)
	}
}

func (c *RealtimeCollector) run(interval time.Duration) {
	defer c.wg.Done()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	bucket := newBucket()
	for {
		select {
		case result := <-c.results:
			c.ingest(bucket, result)
		case <-ticker.C:
			c.publishSnapshot(bucket)
		case <-c.done:
			c.drainResults(bucket)
			c.flushSnapshot(bucket)
			return
		}
	}
}

func (c *RealtimeCollector) drainResults(bucket *bucket) {
	for {
		select {
		case result := <-c.results:
			c.ingest(bucket, result)
		default:
			return
		}
	}
}

func (c *RealtimeCollector) publishSnapshot(bucket *bucket) {
	snapshot := bucket.snapshot()
	if snapshot.TotalRequests == 0 {
		return
	}

	select {
	case c.snapshots <- snapshot:
	case <-c.done:
		return
	}
	bucket.reset()
}

func (c *RealtimeCollector) flushSnapshot(bucket *bucket) {
	snapshot := bucket.snapshot()
	if snapshot.TotalRequests == 0 {
		return
	}
	c.snapshots <- snapshot
	bucket.reset()
}

func (c *RealtimeCollector) Close() {
	c.once.Do(func() {
		close(c.done)
		c.wg.Wait()
		close(c.snapshots)
	})
}
