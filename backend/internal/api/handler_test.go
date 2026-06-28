package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/awu123-create/http-bench/internal/config"
	"github.com/awu123-create/http-bench/internal/storage"
	"github.com/awu123-create/http-bench/internal/storage/db"
	"github.com/awu123-create/http-bench/internal/task"
	"github.com/gin-gonic/gin"
)

func newTestRouter(t *testing.T) (*gin.Engine, *db.Queries) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	sqlDB, err := storage.Open(
		context.Background(),
		filepath.Join(t.TempDir(), "api-test.db"),
		filepath.Join("..", "..", "migrations"),
	)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	queries := db.New(sqlDB)
	manager := task.NewManager(queries)
	t.Cleanup(func() {
		manager.Close()
		_ = sqlDB.Close()
	})

	return NewRouter(manager), queries
}

func validRequestConfig(targetURL string) config.TestConfig {
	return config.TestConfig{
		Name: "api-test",
		Request: config.RequestConfig{
			Method:   http.MethodGet,
			URL:      targetURL,
			BodyType: "none",
		},
		Concurrency:     2,
		QPS:             5,
		Duration:        1,
		TimeoutMillis:   2000,
		MaxConnsPerHost: 2,
	}
}

func doJSON(t *testing.T, router *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func do(t *testing.T, router *gin.Engine, method, path string) *httptest.ResponseRecorder {
	return doJSON(t, router, method, path, nil)
}

func decodeData(t *testing.T, w *httptest.ResponseRecorder, target any) {
	t.Helper()
	var resp struct {
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v (body=%s)", err, w.Body.String())
	}
	if err := json.Unmarshal(resp.Data, target); err != nil {
		t.Fatalf("decode data: %v (data=%s)", err, string(resp.Data))
	}
}

// waitFor 等待条件成立，超时失败
func waitFor(t *testing.T, timeout time.Duration, check func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("condition not satisfied within %s", timeout)
}

func TestListTests(t *testing.T) {
	router, _ := newTestRouter(t)

	for i := 0; i < 2; i++ {
		w := doJSON(t, router, http.MethodPost, "/api/tests", validRequestConfig("http://example.com"))
		if w.Code != http.StatusCreated {
			t.Fatalf("create #%d: status = %d, body=%s", i, w.Code, w.Body.String())
		}
	}

	w := do(t, router, http.MethodGet, "/api/tests")
	if w.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200, body=%s", w.Code, w.Body.String())
	}

	var data struct {
		Tasks []*task.Task `json:"tasks"`
	}
	decodeData(t, w, &data)
	if len(data.Tasks) != 2 {
		t.Fatalf("len(tasks) = %d, want 2", len(data.Tasks))
	}
}

func TestGetTestResultNotFound(t *testing.T) {
	router, _ := newTestRouter(t)

	w := do(t, router, http.MethodGet, "/api/tests/9999/result")
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404, body=%s", w.Code, w.Body.String())
	}
}

func TestGetTestResultAfterRun(t *testing.T) {
	router, _ := newTestRouter(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	// 创建任务
	w := doJSON(t, router, http.MethodPost, "/api/tests", validRequestConfig(server.URL))
	if w.Code != http.StatusCreated {
		t.Fatalf("create: status = %d, body=%s", w.Code, w.Body.String())
	}
	var created struct {
		TaskID int64 `json:"task_id"`
	}
	decodeData(t, w, &created)
	taskID := strconv.FormatInt(created.TaskID, 10)

	// 启动任务
	w = do(t, router, http.MethodPost, "/api/tests/"+taskID+"/start")
	if w.Code != http.StatusOK {
		t.Fatalf("start: status = %d, body=%s", w.Code, w.Body.String())
	}

	// 等待任务结束
	waitFor(t, 4*time.Second, func() bool {
		w := do(t, router, http.MethodGet, "/api/tests/"+taskID)
		if w.Code != http.StatusOK {
			return false
		}
		var d struct {
			Task *task.Task `json:"task"`
		}
		decodeData(t, w, &d)
		return d.Task.Status == task.TaskFinished
	})

	// 获取最终结果
	w = do(t, router, http.MethodGet, "/api/tests/"+taskID+"/result")
	if w.Code != http.StatusOK {
		t.Fatalf("result: status = %d, want 200, body=%s", w.Code, w.Body.String())
	}

	var data struct {
		Result db.TestResult `json:"result"`
	}
	decodeData(t, w, &data)
	if data.Result.TotalRequests == 0 {
		t.Fatal("total_requests = 0, want > 0")
	}
}

func TestStopTest(t *testing.T) {
	router, _ := newTestRouter(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := validRequestConfig(server.URL)
	cfg.Duration = 30 // 长任务，确保 stop 前仍在运行

	w := doJSON(t, router, http.MethodPost, "/api/tests", cfg)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: status = %d, body=%s", w.Code, w.Body.String())
	}
	var created struct {
		TaskID int64 `json:"task_id"`
	}
	decodeData(t, w, &created)
	taskID := strconv.FormatInt(created.TaskID, 10)

	w = do(t, router, http.MethodPost, "/api/tests/"+taskID+"/start")
	if w.Code != http.StatusOK {
		t.Fatalf("start: status = %d, body=%s", w.Code, w.Body.String())
	}

	w = do(t, router, http.MethodPost, "/api/tests/"+taskID+"/stop")
	if w.Code != http.StatusOK {
		t.Fatalf("stop: status = %d, want 200, body=%s", w.Code, w.Body.String())
	}

	// 等待状态变为 Stopped
	waitFor(t, 4*time.Second, func() bool {
		w := do(t, router, http.MethodGet, "/api/tests/"+taskID)
		if w.Code != http.StatusOK {
			return false
		}
		var d struct {
			Task *task.Task `json:"task"`
		}
		decodeData(t, w, &d)
		return d.Task.Status == task.TaskStopped
	})
}

func TestStopTestNotFound(t *testing.T) {
	router, _ := newTestRouter(t)

	w := do(t, router, http.MethodPost, "/api/tests/9999/stop")
	if w.Code != http.StatusNotFound {
		t.Fatalf("stop non-existent: status = %d, want 404, body=%s", w.Code, w.Body.String())
	}
}

func TestStopTestNotRunning(t *testing.T) {
	router, _ := newTestRouter(t)

	// 创建任务但不启动
	w := doJSON(t, router, http.MethodPost, "/api/tests", validRequestConfig("http://example.com"))
	if w.Code != http.StatusCreated {
		t.Fatalf("create: status = %d, body=%s", w.Code, w.Body.String())
	}
	var created struct {
		TaskID int64 `json:"task_id"`
	}
	decodeData(t, w, &created)
	taskID := strconv.FormatInt(created.TaskID, 10)

	// 直接 stop —— 任务存在但未运行 → 409
	w = do(t, router, http.MethodPost, "/api/tests/"+taskID+"/stop")
	if w.Code != http.StatusConflict {
		t.Fatalf("stop not-running: status = %d, want 409, body=%s", w.Code, w.Body.String())
	}
}
