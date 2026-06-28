package engine

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/awu123-create/http-bench/internal/config"
	"github.com/awu123-create/http-bench/internal/metrics"
)

type recordingCollector struct {
	mu      sync.Mutex
	results []metrics.RequestResult
}

func (c *recordingCollector) Record(result metrics.RequestResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.results = append(c.results, result)
}

func (c *recordingCollector) Snapshots() <-chan metrics.Snapshot {
	return nil
}

func (c *recordingCollector) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.results)
}

func (c *recordingCollector) successCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	var count int
	for _, result := range c.results {
		if result.Err == nil && result.StatusCode == http.StatusOK {
			count++
		}
	}
	return count
}

func TestRunSendsRequestsAndRecordsResults(t *testing.T) {
	var serverRequests atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverRequests.Add(1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	collector := &recordingCollector{}
	cfg := config.TestConfig{
		Request: config.RequestConfig{
			Method: "GET",
			URL:    server.URL,
		},
		Concurrency: 2,
		QPS:         5,
		Duration:    1,
		TimeoutMillis:     2000,
	}

	Run(context.Background(), cfg, collector)

	recorded := collector.count()
	if recorded == 0 {
		t.Fatal("Run recorded no requests")
	}
	if recorded > cfg.QPS {
		t.Fatalf("recorded %d requests, want at most configured QPS %d", recorded, cfg.QPS)
	}
	if got, success := int(serverRequests.Load()), collector.successCount(); success > got {
		t.Fatalf("collector recorded %d successful responses, but server handled %d requests", success, got)
	}
}

func TestRunUnlimitedQPS(t *testing.T) {
	var serverRequests atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverRequests.Add(1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	collector := &recordingCollector{}
	cfg := config.TestConfig{
		Request: config.RequestConfig{
			Method: "GET",
			URL:    server.URL,
		},
		Concurrency: 2,
		QPS:         0,
		Duration:    1,
		TimeoutMillis:     2000,
	}

	Run(context.Background(), cfg, collector)

	if collector.successCount() <= 5 {
		t.Fatalf("unlimited QPS recorded %d successful requests, want more than fixed 5 QPS baseline", collector.successCount())
	}
	if got, success := int(serverRequests.Load()), collector.successCount(); success > got {
		t.Fatalf("collector recorded %d successful responses, but server handled %d requests", success, got)
	}
}

func TestNewHTTPClientBundleMaxConnsFallbackToConcurrency(t *testing.T) {
	cfg := config.TestConfig{
		Concurrency:     4,
		TimeoutMillis:   1000,
		MaxConnsPerHost: 0,
	}

	bundle := NewHTTPClientBundle(cfg)
	defer bundle.Close()

	if bundle.Transport.MaxConnsPerHost != 4 {
		t.Fatalf("MaxConnsPerHost = %d, want 4 (fallback to concurrency)", bundle.Transport.MaxConnsPerHost)
	}
	if bundle.Transport.MaxIdleConns != 4 {
		t.Fatalf("MaxIdleConns = %d, want 4", bundle.Transport.MaxIdleConns)
	}
	if bundle.Transport.MaxIdleConnsPerHost != 4 {
		t.Fatalf("MaxIdleConnsPerHost = %d, want 4", bundle.Transport.MaxIdleConnsPerHost)
	}
}

func TestNewHTTPClientBundleMaxConnsExplicit(t *testing.T) {
	cfg := config.TestConfig{
		Concurrency:     4,
		TimeoutMillis:   1000,
		MaxConnsPerHost: 7,
	}

	bundle := NewHTTPClientBundle(cfg)
	defer bundle.Close()

	if bundle.Transport.MaxConnsPerHost != 7 {
		t.Fatalf("MaxConnsPerHost = %d, want 7 (explicit value)", bundle.Transport.MaxConnsPerHost)
	}
	if bundle.Transport.MaxIdleConns != 7 {
		t.Fatalf("MaxIdleConns = %d, want 7", bundle.Transport.MaxIdleConns)
	}
	if bundle.Transport.MaxIdleConnsPerHost != 7 {
		t.Fatalf("MaxIdleConnsPerHost = %d, want 7", bundle.Transport.MaxIdleConnsPerHost)
	}
}
