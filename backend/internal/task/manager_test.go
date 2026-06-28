package task

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/awu123-create/http-bench/internal/config"
	"github.com/awu123-create/http-bench/internal/storage"
	"github.com/awu123-create/http-bench/internal/storage/db"
)

func newTestManager(t *testing.T) (*Manager, *db.Queries) {
	t.Helper()

	sqlDB, err := storage.Open(
		context.Background(),
		filepath.Join(t.TempDir(), "http-bench-test.db"),
		filepath.Join("..", "..", "migrations"),
	)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	queries := db.New(sqlDB)
	manager := NewManager(queries)
	t.Cleanup(func() {
		manager.Close()
		if err := sqlDB.Close(); err != nil {
			t.Fatalf("close test database: %v", err)
		}
	})

	return manager, queries
}

func validConfig(targetURL string) config.TestConfig {
	return config.TestConfig{
		Name: "manager-test",
		Request: config.RequestConfig{
			Method:   http.MethodGet,
			URL:      targetURL,
			Headers:  map[string]string{"X-Test": "task-manager"},
			BodyType: "none",
		},
		Concurrency:     2,
		QPS:             5,
		Duration:        1,
		TimeoutMillis:   2000,
		MaxConnsPerHost: 3,
	}
}

func waitFor(t *testing.T, timeout time.Duration, check func() bool) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("condition was not satisfied within %s", timeout)
}

func TestManagerCreateAndGet(t *testing.T) {
	manager, _ := newTestManager(t)
	cfg := validConfig("http://example.com")

	created, err := manager.Create(context.Background(), cfg)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("created task ID is zero")
	}
	if created.Status != TaskCreated {
		t.Fatalf("created status = %q, want %q", created.Status, TaskCreated)
	}

	got, err := manager.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if got.Name != cfg.Name {
		t.Fatalf("task name = %q, want %q", got.Name, cfg.Name)
	}
	if got.Status != TaskCreated {
		t.Fatalf("task status = %q, want %q", got.Status, TaskCreated)
	}
	if got.Config.Request.Headers["X-Test"] != "task-manager" {
		t.Fatalf("headers were not restored from database: %#v", got.Config.Request.Headers)
	}
	if got.Config.MaxConnsPerHost != 3 {
		t.Fatalf("max_conns_per_host = %d, want 3 (round-trip from database)", got.Config.MaxConnsPerHost)
	}
}

func TestManagerCreateRejectsInvalidConfig(t *testing.T) {
	manager, _ := newTestManager(t)
	cfg := validConfig("not-a-url")

	_, err := manager.Create(context.Background(), cfg)
	if !errors.Is(err, ErrInvalidConfig) {
		t.Fatalf("create invalid config error = %v, want %v", err, ErrInvalidConfig)
	}
}

func TestManagerStartRunsTaskAndWritesMetricBuckets(t *testing.T) {
	manager, queries := newTestManager(t)
	var hits int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&hits, 1)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	task, err := manager.Create(context.Background(), validConfig(server.URL))
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if err := manager.Start(context.Background(), task.ID); err != nil {
		t.Fatalf("start task: %v", err)
	}

	waitFor(t, 4*time.Second, func() bool {
		got, err := manager.Get(context.Background(), task.ID)
		return err == nil && got.Status == TaskFinished
	})

	got, err := queries.GetTask(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("get persisted task: %v", err)
	}
	if !got.StartedAt.Valid {
		t.Fatal("started_at was not persisted")
	}
	if !got.FinishedAt.Valid {
		t.Fatal("finished_at was not persisted")
	}
	if atomic.LoadInt64(&hits) == 0 {
		t.Fatal("target server did not receive any requests")
	}

	var buckets []db.MetricsBucket
	waitFor(t, 2*time.Second, func() bool {
		var err error
		buckets, err = queries.ListMetricsBucketsByTestID(context.Background(), task.ID)
		if err != nil {
			t.Fatalf("list metric buckets: %v", err)
		}
		return len(buckets) > 0
	})

	var totalRequests int64
	statusCodes := map[string]int64{}
	for i, bucket := range buckets {
		if bucket.SecondOffset != int64(i) {
			t.Fatalf("bucket %d second_offset = %d, want %d", i, bucket.SecondOffset, i)
		}
		totalRequests += bucket.Requests
		var bucketStatusCodes map[string]int64
		if err := json.Unmarshal([]byte(bucket.StatusCodeJson), &bucketStatusCodes); err != nil {
			t.Fatalf("unmarshal status_code_json %q: %v", bucket.StatusCodeJson, err)
		}
		for code, count := range bucketStatusCodes {
			statusCodes[code] += count
		}
	}
	if totalRequests == 0 {
		t.Fatal("metric buckets did not record any requests")
	}
	if statusCodes["204"] == 0 {
		t.Fatalf("status_code_json did not record HTTP 204 responses: %#v", statusCodes)
	}

	result, err := queries.GetTestResult(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("get final test result: %v", err)
	}
	if result.TotalRequests != totalRequests {
		t.Fatalf("final total requests = %d, want %d", result.TotalRequests, totalRequests)
	}
	// Duration 与 QPS 的边界处（最后一个 tick 与 deadline 同时就绪）可能产生
	// 至多 1 个被 deadline 取消的失败请求，这里允许该边界容差。
	if result.TotalFailure > 1 {
		t.Fatalf("final total failure = %d, want at most 1 (boundary tolerance)", result.TotalFailure)
	}
	if result.TotalSuccess != totalRequests-result.TotalFailure {
		t.Fatalf("final total success = %d, want %d (failure=%d)", result.TotalSuccess, totalRequests-result.TotalFailure, result.TotalFailure)
	}
}

func TestManagerStartRejectsAlreadyRunning(t *testing.T) {
	manager, _ := newTestManager(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := validConfig(server.URL)
	cfg.Duration = 2
	task, err := manager.Create(context.Background(), cfg)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if err := manager.Start(context.Background(), task.ID); err != nil {
		t.Fatalf("start task: %v", err)
	}
	t.Cleanup(func() {
		_ = manager.Stop(context.Background(), task.ID)
	})

	err = manager.Start(context.Background(), task.ID)
	if !errors.Is(err, ErrTaskAlreadyRunning) {
		t.Fatalf("second start error = %v, want %v", err, ErrTaskAlreadyRunning)
	}
}

func TestManagerSubscribeMetricsPublishesAndUnsubscribes(t *testing.T) {
	manager, _ := newTestManager(t)
	task, err := manager.Create(context.Background(), validConfig("http://example.com"))
	if err != nil {
		t.Fatalf("create task: %v", err)
	}

	_, _, err = manager.SubscribeMetrics(context.Background(), task.ID)
	if !errors.Is(err, ErrTaskNotRunning) {
		t.Fatalf("subscribe stopped task error = %v, want %v", err, ErrTaskNotRunning)
	}

	manager.mu.Lock()
	manager.running[task.ID] = task
	manager.mu.Unlock()

	ch, unsubscribe, err := manager.SubscribeMetrics(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("subscribe running task: %v", err)
	}

	want := MetricPoint{SecondOffset: 1}
	manager.publishMetric(task.ID, want)

	select {
	case got := <-ch:
		if got.SecondOffset != want.SecondOffset {
			t.Fatalf("published second_offset = %d, want %d", got.SecondOffset, want.SecondOffset)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for published metric")
	}

	done := make(chan struct{})
	go func() {
		for i := 0; i < 32; i++ {
			manager.publishMetric(task.ID, MetricPoint{SecondOffset: int64(i)})
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("publishMetric blocked on a slow subscriber")
	}

	unsubscribe()
	manager.publishMetric(task.ID, MetricPoint{SecondOffset: 99})

	for point := range ch {
		if point.SecondOffset == 99 {
			t.Fatal("received metric after unsubscribe")
		}
	}
}

func TestManagerStopUpdatesStatusAndCancelsTask(t *testing.T) {
	manager, _ := newTestManager(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := validConfig(server.URL)
	cfg.Duration = 10
	task, err := manager.Create(context.Background(), cfg)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if err := manager.Start(context.Background(), task.ID); err != nil {
		t.Fatalf("start task: %v", err)
	}

	if err := manager.Stop(context.Background(), task.ID); err != nil {
		t.Fatalf("stop task: %v", err)
	}

	waitFor(t, 2*time.Second, func() bool {
		got, err := manager.Get(context.Background(), task.ID)
		return err == nil && got.Status == TaskStopped
	})
	waitFor(t, 2*time.Second, func() bool {
		manager.mu.Lock()
		defer manager.mu.Unlock()
		_, exists := manager.running[task.ID]
		return !exists
	})

	got, err := manager.Get(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("get stopped task: %v", err)
	}
	if got.Status != TaskStopped {
		t.Fatalf("task status = %q, want %q", got.Status, TaskStopped)
	}
}

func TestManagerStopNotFound(t *testing.T) {
	manager, _ := newTestManager(t)

	err := manager.Stop(context.Background(), 99999)
	if !errors.Is(err, ErrTaskNotFound) {
		t.Fatalf("stop non-existent task error = %v, want %v", err, ErrTaskNotFound)
	}
}

func TestManagerStopNotRunning(t *testing.T) {
	manager, _ := newTestManager(t)

	task, err := manager.Create(context.Background(), validConfig("http://example.com"))
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	// 不 Start，直接 Stop —— 任务存在但未运行
	err = manager.Stop(context.Background(), task.ID)
	if !errors.Is(err, ErrTaskNotRunning) {
		t.Fatalf("stop not-running task error = %v, want %v", err, ErrTaskNotRunning)
	}
}

func TestManagerFinalResultHasPercentiles(t *testing.T) {
	manager, queries := newTestManager(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 引入可测的延迟，确保 latency.Milliseconds() >= 1
		time.Sleep(3 * time.Millisecond)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	cfg := validConfig(server.URL)
	cfg.QPS = 10
	cfg.Duration = 1
	task, err := manager.Create(context.Background(), cfg)
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if err := manager.Start(context.Background(), task.ID); err != nil {
		t.Fatalf("start task: %v", err)
	}

	waitFor(t, 4*time.Second, func() bool {
		got, err := manager.Get(context.Background(), task.ID)
		return err == nil && got.Status == TaskFinished
	})

	result, err := queries.GetTestResult(context.Background(), task.ID)
	if err != nil {
		t.Fatalf("get final test result: %v", err)
	}
	if result.TotalRequests == 0 {
		t.Fatal("final result recorded no requests")
	}
	// 慢服务器延迟约 3ms，分位数应 >= 1（证明直方图端到端贯通）
	if result.P50LatencyMs < 1 {
		t.Fatalf("P50 = %d, want >= 1", result.P50LatencyMs)
	}
	if result.P90LatencyMs < 1 {
		t.Fatalf("P90 = %d, want >= 1", result.P90LatencyMs)
	}
	if result.P95LatencyMs < 1 {
		t.Fatalf("P95 = %d, want >= 1", result.P95LatencyMs)
	}
	if result.P99LatencyMs < 1 {
		t.Fatalf("P99 = %d, want >= 1", result.P99LatencyMs)
	}
	// 分位数单调性
	if result.P50LatencyMs > result.P90LatencyMs ||
		result.P90LatencyMs > result.P95LatencyMs ||
		result.P95LatencyMs > result.P99LatencyMs {
		t.Fatalf("percentiles not monotonic: P50=%d P90=%d P95=%d P99=%d",
			result.P50LatencyMs, result.P90LatencyMs, result.P95LatencyMs, result.P99LatencyMs)
	}
}
