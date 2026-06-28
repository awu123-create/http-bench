package engine

import (
	"context"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/awu123-create/http-bench/internal/config"
	"github.com/awu123-create/http-bench/internal/metrics"
)

type HTTPClientBundle struct {
	Client    *http.Client
	Transport *http.Transport
}

func NewHTTPClientBundle(cfg config.TestConfig) *HTTPClientBundle {
	cfg = normalizeConfig(cfg)
	maxConns := cfg.MaxConnsPerHost
	if maxConns <= 0 {
		maxConns = cfg.Concurrency
	}

	transport := &http.Transport{
		MaxIdleConns:          maxConns,
		MaxIdleConnsPerHost:   maxConns,
		MaxConnsPerHost:       maxConns,
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableKeepAlives:     false,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(cfg.TimeoutMillis) * time.Millisecond,
	}

	return &HTTPClientBundle{
		Client:    client,
		Transport: transport,
	}
}

func Run(ctx context.Context, cfg config.TestConfig, collector metrics.Collector) error {
	cfg = normalizeConfig(cfg)

	bundle := NewHTTPClientBundle(cfg)
	defer bundle.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Duration(cfg.Duration)*time.Second)
	defer cancel()

	jobs := make(chan struct{}, cfg.Concurrency)
	var wg sync.WaitGroup

	for i := 0; i < cfg.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			worker(ctx, cfg, bundle.Client, collector, jobs)
		}()
	}
	defer wg.Wait()
	defer close(jobs)

	if cfg.QPS == 0 {
		return dispatchUnlimited(ctx, jobs)
	}
	return dispatchWithQPS(ctx, cfg.QPS, jobs, collector)
}

func dispatchUnlimited(ctx context.Context, jobs chan<- struct{}) error {
	for {
		select {
		case jobs <- struct{}{}:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func dispatchWithQPS(ctx context.Context, qps int, jobs chan<- struct{}, collector metrics.Collector) error {
	interval := time.Second / time.Duration(qps)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 在派发作业前检查 context 是否已过期，避免在 duration 边界
			// 派发注定会被 deadline 取消的请求（QPS*Duration 边界竞态）。
			if err := ctx.Err(); err != nil {
				return err
			}
			select {
			case jobs <- struct{}{}:
			default:
				collector.Record(metrics.RequestResult{
					ErrorType:  metrics.ErrorClientOverload,
					ErrorMsg:   "worker queue is full",
					FinishedAt: time.Now(),
				})
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func normalizeConfig(cfg config.TestConfig) config.TestConfig {
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 10
	}
	if cfg.Duration <= 0 {
		cfg.Duration = 10
	}
	if cfg.TimeoutMillis <= 0 {
		cfg.TimeoutMillis = 30000
	}
	return cfg
}

func (b *HTTPClientBundle) Close() {
	b.Transport.CloseIdleConnections()
}

func worker(ctx context.Context, cfg config.TestConfig, client *http.Client, collector metrics.Collector, jobs <-chan struct{}) {
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-jobs:
			if !ok {
				return
			}

			// 构造 HTTP 请求
			start := time.Now()
			req, err := buildRequest(ctx, cfg)
			if err != nil {
				// 处理错误
				collector.Record(metrics.RequestResult{
					Err:        err,
					ErrorType:  metrics.ErrorRequestBuild,
					ErrorMsg:   errorMessage(err),
					Latency:    time.Since(start).Milliseconds(),
					FinishedAt: time.Now(),
				})
				continue
			}

			// 发送 HTTP 请求
			resp, err := client.Do(req)
			latency := time.Since(start).Milliseconds()

			if err != nil {
				if ctx.Err() != nil {
					continue // 如果是 context 取消或超时导致的错误，则忽略
				}
				collector.Record(metrics.RequestResult{
					Err:        err,
					ErrorType:  classifyDoError(err),
					ErrorMsg:   errorMessage(err),
					Latency:    latency,
					FinishedAt: time.Now(),
				})
				continue
			}

			// 读取响应体以确保连接被正确回收
			_, readErr := io.Copy(io.Discard, resp.Body)
			closeErr := resp.Body.Close()
			if readErr != nil {
				if ctx.Err() != nil {
					continue // 如果是 context 取消或超时导致的错误，则忽略
				}
				collector.Record(metrics.RequestResult{
					StatusCode: resp.StatusCode,
					Err:        readErr,
					ErrorType:  metrics.ErrorReadBody,
					ErrorMsg:   errorMessage(readErr),
					Latency:    latency,
					FinishedAt: time.Now(),
				})
				continue
			}
			if closeErr != nil {
				if ctx.Err() != nil {
					continue // 如果是 context 取消或超时导致的错误，则忽略
				}
				collector.Record(metrics.RequestResult{
					StatusCode: resp.StatusCode,
					Err:        closeErr,
					ErrorType:  metrics.ErrorReadBody,
					ErrorMsg:   errorMessage(closeErr),
					Latency:    latency,
					FinishedAt: time.Now(),
				})
				continue
			}

			errorType := classifyStatusCode(resp.StatusCode)
			result := metrics.RequestResult{
				StatusCode: resp.StatusCode,
				Latency:    latency,
				FinishedAt: time.Now(),
			}
			if errorType != metrics.ErrorNone {
				result.ErrorType = errorType
				result.ErrorMsg = httpStatusMessage(resp.StatusCode)
			}
			collector.Record(result)
		}
	}
}
