package task

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/awu123-create/http-bench/internal/config"
	"github.com/awu123-create/http-bench/internal/engine"
	"github.com/awu123-create/http-bench/internal/metrics"
	"github.com/awu123-create/http-bench/internal/storage/db"
)

// task manager 负责：
// 1. 创建任务
// 2. 启动任务
// 3. 停止任务
// 4. 查询任务状态
// 5. 保存 cancel 函数/运行状态/实时快照

type TaskStatus string

const (
	TaskCreated  = "Created"
	TaskRunning  = "Running"
	TaskStopped  = "Stopped"
	TaskFinished = "Finished"
	TaskFailed   = "Failed"
)

var (
	ErrInvalidConfig      = errors.New("invalid config")
	ErrTaskAlreadyRunning = errors.New("task already running")
	ErrTaskNotFound       = errors.New("task not found")
	ErrTaskNotRunning     = errors.New("task not running")
	ErrInvalidTaskID      = errors.New("invalid task ID")
)

type Task struct {
	ID     int64           `json:"id"`
	Name   string          `json:"name"`
	Config config.TestConfig `json:"config"`
	Status TaskStatus      `json:"status"`

	cancel    context.CancelFunc
	latest    metrics.Snapshot
	startedAt string
}

func (t *Task) SetName(name string) {
	if name == "" {
		t.Name = defaultTaskName()
	} else {
		t.Name = name
	}
}

func defaultTaskName() string {
	return fmt.Sprintf("http-bench-task-%s", time.Now().Format("20060102-150405"))
}

func (t *Task) SetCancel(cancel context.CancelFunc) {
	t.cancel = cancel
}

type Manager struct {
	queries     db.Querier
	mu          sync.Mutex
	running     map[int64]*Task
	subscribers map[int64]map[chan MetricPoint]struct{}
	wg          sync.WaitGroup
}

func NewManager(queries db.Querier) *Manager {
	return &Manager{
		queries:     queries,
		running:     make(map[int64]*Task),
		subscribers: make(map[int64]map[chan MetricPoint]struct{}),
	}
}

func configTransform(record db.TestTask) (config.TestConfig, error) {
	var headers map[string]string
	if record.HeadersJson == "" {
		headers = map[string]string{}
	} else {
		if err := json.Unmarshal([]byte(record.HeadersJson), &headers); err != nil {
			return config.TestConfig{}, err
		}
	}

	return config.TestConfig{
		Request: config.RequestConfig{
			Method:   record.Method,
			URL:      record.Url,
			Headers:  headers,
			BodyType: record.BodyType,
			BodyRaw:  record.BodyRaw,
		},
		Concurrency:     int(record.Concurrency),
		QPS:             int(record.Qps),
		Duration:        int(record.DurationSeconds),
		TimeoutMillis:   int(record.TimeoutMillis),
		MaxConnsPerHost: int(record.MaxConnsPerHost),
	}, nil
}

func turnToSnapshot(result db.MetricsBucket) metrics.Snapshot {
	s := metrics.Snapshot{
		TotalRequests: result.Requests,
		Success:       result.Success,
		Failed:        result.Failure,
		AvgLatency:    result.AvgLatencyMs,
		MinLatency:    result.MinLatencyMs,
		MaxLatency:    result.MaxLatencyMs,
		StatusCodes:   statusCodeFromJSON(result.StatusCodeJson),
	}

	errs := errorsFromJSON(result.ErrorsJson)
	for k, v := range errs {
		switch k {
		case "request_build":
			s.RequestBuildErrors = v
		case "timeout":
			s.TimeoutErrors = v
		case "canceled":
			s.CanceledErrors = v
		case "network":
			s.NetworkErrors = v
		case "dns":
			s.DNSErrors = v
		case "connection_refused":
			s.ConnectionRefusedErrors = v
		case "connection_reset":
			s.ConnectionResetErrors = v
		case "too_many_open_files":
			s.TooManyOpenFilesErrors = v
		case "read_body":
			s.ReadBodyErrors = v
		case "http_4xx":
			s.HTTP4xxErrors = v
		case "http_5xx":
			s.HTTP5xxErrors = v
		case "http_status":
			s.HTTPStatusErrors = v
		case "business":
			s.BusinessErrors = v
		case "client_overload":
			s.ClientOverloadErrors = v
		}
	}
	return s
}

func stringToNullString(s string) sql.NullString {
	return sql.NullString{
		String: s,
		Valid:  s != "",
	}
}

func nullStringValue(s sql.NullString) string {
	if !s.Valid {
		return ""
	}
	return s.String
}

func (m *Manager) Create(ctx context.Context, cfg config.TestConfig) (*Task, error) {
	if config.ValidateConfig(cfg) != config.None {
		return nil, ErrInvalidConfig
	}

	if cfg.Name == "" {
		cfg.Name = defaultTaskName()
	}

	headerJSON, err := json.Marshal(cfg.Request.Headers)
	if err != nil {
		return nil, err
	}

	result, err := m.queries.CreateTask(ctx, db.CreateTaskParams{
		Name:            cfg.Name,
		Method:          cfg.Request.Method,
		Url:             cfg.Request.URL,
		HeadersJson:     string(headerJSON),
		BodyType:        cfg.Request.BodyType,
		BodyRaw:         cfg.Request.BodyRaw,
		Concurrency:     int64(cfg.Concurrency),
		Qps:             int64(cfg.QPS),
		DurationSeconds: int64(cfg.Duration),
		TimeoutMillis:   int64(cfg.TimeoutMillis),
		Status:          TaskCreated,
		CreatedAt:       time.Now().Format(time.RFC3339Nano),
		MaxConnsPerHost: int64(cfg.MaxConnsPerHost),
	})
	if err != nil {
		return nil, err
	}

	task := &Task{
		ID:     result.ID,
		Name:   result.Name,
		Config: cfg,
		Status: TaskCreated,
	}
	return task, nil
}

func (m *Manager) Start(ctx context.Context, id int64) error {
	m.mu.Lock()
	if _, exists := m.running[id]; exists {
		m.mu.Unlock()
		return ErrTaskAlreadyRunning
	}
	m.mu.Unlock()

	task, err := m.queries.GetTask(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrTaskNotFound
		}
		return err
	}

	cfg, err := configTransform(task)
	if err != nil {
		return err
	}

	taskCtx, cancel := context.WithCancel(context.Background())
	collector := metrics.NewRealtimeCollector(cfg.QPS)
	startedAt := time.Now()
	startedAtString := startedAt.Format(time.RFC3339Nano)
	runningTask := &Task{
		ID:        task.ID,
		Name:      task.Name,
		Config:    cfg,
		Status:    TaskRunning,
		cancel:    cancel,
		startedAt: startedAtString,
	}

	m.mu.Lock()
	if _, exists := m.running[runningTask.ID]; exists {
		m.mu.Unlock()
		cancel()
		collector.Close()
		return ErrTaskAlreadyRunning
	}
	m.running[runningTask.ID] = runningTask
	m.mu.Unlock()

	// 更新任务的状态为running
	err = m.queries.UpdateTask(ctx, db.UpdateTaskParams{
		Status:    TaskRunning,
		StartedAt: stringToNullString(startedAtString),
		ID:        task.ID,
	})
	if err != nil {
		m.mu.Lock()
		delete(m.running, runningTask.ID)
		m.mu.Unlock()
		cancel()
		collector.Close()
		return err
	}

	snapshotWriteDone := make(chan struct{})
	m.wg.Add(1)
	go func() {
		defer close(snapshotWriteDone)
		var secondOffset int64
		for snapshot := range collector.Snapshots() {
			m.mu.Lock()
			runningTask.latest = snapshot
			m.mu.Unlock()

			startedTask := task
			writeCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			currentOffset := secondOffset
			secondOffset++

			_, writeErr := m.queries.CreateMetricBucket(writeCtx, metricBucketParams(startedTask.ID, snapshot, currentOffset))
			cancel()
			if writeErr != nil {
				log.Printf("write metric bucket for task %d failed: %v", startedTask.ID, writeErr)
			}

			m.publishMetric(startedTask.ID, MetricPoint{
				SecondOffset: currentOffset,
				Snapshot:     snapshot,
			})
		}
	}()
	go func() {
		engineErr := engine.Run(taskCtx, cfg, collector)
		defer m.wg.Done()
		collector.Close()
		<-snapshotWriteDone

		percentiles := collector.Percentiles()
		resultCtx, cancelResult := context.WithTimeout(context.Background(), 3*time.Second)
		resultErr := m.createFinalResult(resultCtx, task.ID, percentiles)
		cancelResult()
		if resultErr != nil {
			log.Printf("create final result for task %d failed: %v", task.ID, resultErr)
		}

		finalStatus := TaskFinished
		if engineErr != nil {
			switch {
			case errors.Is(engineErr, context.DeadlineExceeded):
				finalStatus = TaskFinished
			case errors.Is(engineErr, context.Canceled):
				finalStatus = ""
			default:
				finalStatus = TaskFailed
			}
		}

		if finalStatus != "" {
			updateCtx, cancelUpdate := context.WithTimeout(context.Background(), 3*time.Second)
			updateErr := m.queries.UpdateTask(updateCtx, db.UpdateTaskParams{
				Status:     finalStatus,
				StartedAt:  stringToNullString(startedAtString),
				FinishedAt: stringToNullString(time.Now().Format(time.RFC3339Nano)),
				ID:         task.ID,
			})
			cancelUpdate()
			if updateErr != nil {
				log.Printf("update final status for task %d failed: %v", task.ID, updateErr)
			}
		}

		m.mu.Lock()
		delete(m.running, runningTask.ID)
		m.mu.Unlock()
	}()

	return nil
}

func (m *Manager) Get(ctx context.Context, id int64) (*Task, error) {
	record, err := m.queries.GetTask(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	cfg, err := configTransform(record)
	if err != nil {
		return nil, err
	}

	return &Task{
		ID:     record.ID,
		Name:   record.Name,
		Config: cfg,
		Status: TaskStatus(record.Status),
	}, nil
}

func (m *Manager) ListTasks(ctx context.Context) ([]*Task, error) {
	results, err := m.queries.ListTasks(ctx)
	if err != nil {
		return nil, err
	}

	tasks := make([]*Task, 0)
	for _, result := range results {
		cfg, err := configTransform(result)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, &Task{
			ID:     result.ID,
			Name:   result.Name,
			Config: cfg,
			Status: TaskStatus(result.Status),
		})
	}
	return tasks, nil
}

type MetricPoint struct {
	SecondOffset int64            `json:"second_offset"`
	Snapshot     metrics.Snapshot `json:"snapshot"`
}

func (m *Manager) GetAllMetrics(ctx context.Context, id int64) ([]MetricPoint, error) {

	results, err := m.queries.ListMetricsBucketsByTestID(ctx, id)
	if err != nil {
		return nil, err
	}

	var snapshots []MetricPoint
	for _, result := range results {
		snapshots = append(snapshots, MetricPoint{
			SecondOffset: result.SecondOffset,
			Snapshot:     turnToSnapshot(result),
		})
	}
	return snapshots, nil
}

func (m *Manager) GetTestResult(ctx context.Context, id int64) (db.TestResult, error) {
	result, err := m.queries.GetTestResult(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return db.TestResult{}, ErrTaskNotFound
		}
		return db.TestResult{}, err
	}
	return result, nil
}

func (m *Manager) Stop(ctx context.Context, id int64) error {
	m.mu.Lock()
	task, exists := m.running[id]
	m.mu.Unlock()

	if !exists {
		_, err := m.queries.GetTask(ctx, id)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrTaskNotFound
			}
			return err
		}
		return ErrTaskNotRunning
	}

	// 更新任务的状态为stopped
	err := m.queries.UpdateTask(ctx, db.UpdateTaskParams{
		Status:     TaskStopped,
		StartedAt:  stringToNullString(task.startedAt),
		FinishedAt: stringToNullString(time.Now().Format(time.RFC3339Nano)),
		ID:         task.ID,
	})
	if err != nil {
		return err
	}
	task.cancel()
	return nil
}

func (m *Manager) Close() {
	m.wg.Wait()
}
