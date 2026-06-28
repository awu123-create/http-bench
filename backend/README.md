# HTTP 压测工具后端开发文档

## 1. 项目目标

本后端用于支撑单机版 HTTP 压测工具。它的核心目标不是简单地发送 HTTP 请求，而是稳定、可控地执行高并发压测任务，并将请求结果进行高吞吐聚合，最终向前端提供实时和历史指标数据。

后端需要支持用户配置请求方法、目标 URL、QPS、并发数、请求体类型、请求体数据和请求头，并根据这些配置发起压测请求。每批请求完成后，后端需要把请求耗时、状态码、错误类型等结果聚合成指标，用于前端绘制 QPS、延迟变化、错误率、状态码分布等图表。

## 2. 核心能力

### 2.1 压测任务配置

后端应支持以下压测配置：

- `method`：HTTP 方法，例如 `GET`、`POST`、`PUT`、`PATCH`、`DELETE`
- `url`：目标接口地址
- `headers`：自定义请求头
- `body_type`：请求体类型，支持 `none`、`raw`、`x-www-form-urlencoded`、`form-data`（`raw` 常用 JSON 文本）
- `body_raw`：请求体原始内容
- `concurrency`：并发 worker 数量
- `qps`：全局 QPS 限制
- `duration_seconds`：压测持续时间
- `timeout_millis`：单个请求超时时间
- `max_conns_per_host`：单个目标主机最大连接数

请求体和请求头建议先以字符串或 JSON 字符串形式保存，避免过早拆分成复杂表结构。

### 2.2 HTTP 请求执行

压测任务执行时采用任务级 HTTP 客户端复用：

```text
一个压测任务
  -> 创建一个 http.Client
  -> 创建一个 http.Transport
  -> 所有 worker goroutine 共享这个 client
  -> 每次请求单独创建 http.Request
  -> 请求结束后读取并关闭 resp.Body
  -> 任务结束后 CloseIdleConnections()
```

注意：

- `http.Client` 和 `http.Transport` 不应按请求创建。
- `http.Request` 必须按请求创建，尤其是带 body 的 POST/PUT/PATCH 请求。
- `resp.Body` 必须读取完并关闭，否则连接无法回到连接池。
- 任务结束时必须调用 `Transport.CloseIdleConnections()` 释放空闲连接。

推荐使用 Go 标准库 `net/http`，核心连接池能力由 `http.Transport` 提供。

### 2.3 高可用连接池配置

压测工具本身不能比被测服务先被压垮。连接池需要明确限制连接数量，避免文件描述符耗尽。

建议配置：

```go
transport := &http.Transport{
    MaxIdleConns:        maxConns,
    MaxIdleConnsPerHost: maxConns,
    MaxConnsPerHost:     maxConns,
    IdleConnTimeout:     60 * time.Second,
    TLSHandshakeTimeout:  10 * time.Second,
    DisableKeepAlives:   false,
}
```

其中 `maxConns` 默认可以等于 `concurrency`，也可以由用户通过 `max_conns_per_host` 单独配置。

后端还应在任务启动前进行配置校验：

- 并发数必须大于 0
- QPS 必须大于等于 0，0 可表示不限速
- 超时时间必须大于 0
- `max_conns_per_host` 不应超过合理上限
- 检查目标 URL 是否合法

### 2.4 QPS 控制

QPS 控制建议在 engine 层实现，保证所有 worker 共享同一个限速器。

第一版可以使用 `time.Ticker` 或 token channel：

```text
限速器按 QPS 产生 token
worker 每次发请求前获取 token
拿到 token 后才允许发送请求
```

如果 `qps = 0`，则表示不限速，worker 按并发能力持续发送请求。

### 2.5 高吞吐指标聚合

后端不应为每个请求写一行数据库记录。高并发场景下，请求级写库会让压测工具自身成为瓶颈。

推荐流程：

```text
worker 请求完成
  -> 生成 RequestResult
  -> 写入 metrics collector
  -> collector 在内存中更新当前时间桶
  -> 每秒生成一次聚合快照
  -> 快照写入数据库并推送给前端
```

请求结果可包含：

```go
type RequestResult struct {
    StatusCode int
    Latency    int64
    Err        error
    ErrorType  ErrorType
    ErrorMsg   string
    FinishedAt time.Time
}
```

每秒聚合数据包含：

- 当前秒请求数
- 当前秒成功数
- 当前秒失败数
- 平均延迟
- 最小延迟
- 最大延迟
- 状态码分布
- 错误类型分布

每秒桶不计算分位数。分位数（P50/P90/P95/P99）仅在任务结束时由任务级固定线性直方图（60000 桶，1ms 粒度，覆盖 0~60s 延迟）计算，写入最终结果表。该直方图在 collector 中随每个请求累加，O(1) 插入、O(桶数) 查询，内存固定约 469KB，与 QPS/时长无关。

### 2.6 实时数据返回

前端需要绘制延迟变化图、QPS 曲线、错误率曲线和状态码分布，因此后端需要提供实时聚合数据。

推荐第一版使用 SSE：

```text
GET /api/tests/{id}/events
```

服务端每秒推送一条聚合快照：

```json
{
  "second_offset": 12,
  "snapshot": {
    "total_requests": 820,
    "success": 800,
    "failed": 20,
    "avg_latency": 24.5,
    "min_latency": 3,
    "max_latency": 92,
    "status_codes": {
      "200": 800,
      "500": 12
    },
    "timeout_errors": 8
  }
}
```

启动、停止、查询任务仍使用普通 REST API。SSE 只负责后端向前端单向推送实时指标。

## 3. 推荐目录结构

```text
backend/
  cmd/
    server/
      main.go
  internal/
    api/
      handler.go
      handler_test.go
      response.go
      router.go
    config/
      types.go
      validate.go
      validate_test.go
    engine/
      engine.go
      engine_test.go
      request.go
      request_test.go
      errors.go
      errors_test.go
    metrics/
      metrics.go
      bucket.go
      bucket_test.go
      collector.go
      collector_test.go
      histogram.go
      histogram_test.go
    storage/
      sqlite.go
      storage_test.go
      queries/
        tasks.sql
        metrics.sql
      db/                    # sqlc 生成，勿手改
        db.go
        models.go
        querier.go
        tasks.sql.go
        metrics.sql.go
    task/
      manager.go
      manager_test.go
      result.go
      stream.go
      metric_mapper.go
  migrations/
    001_init.sql
    002_add_max_conns_per_host.sql
    003_timeout_millis.sql
  sqlc.yaml
  README.md
```

模块职责：

- `api`：Gin 路由、请求参数绑定、统一响应格式
- `config`：压测任务配置结构和校验逻辑
- `engine`：压测执行器、HTTP client 复用、QPS 控制、worker 调度、请求构造
- `metrics`：请求结果收集、按秒聚合、延迟分位数计算（固定线性直方图）
- `storage`：SQLite 数据访问（sqlc 生成查询代码）
- `task`：任务生命周期管理、SSE 推送、最终结果聚合
- `db`：sqlc 根据 `queries/*.sql` 生成的类型安全查询代码，勿手动编辑

## 4. 后端 API 设计

### 4.1 创建任务

```text
POST /api/tests
```

请求示例：

```json
{
  "name": "create order pressure test",
  "request": {
    "method": "POST",
    "url": "http://localhost:8081/api/orders",
    "headers": {
      "Content-Type": "application/json",
      "Authorization": "Bearer token"
    },
    "body_type": "raw",
    "body_raw": "{\"product_id\":1001,\"quantity\":2}"
  },
  "concurrency": 100,
  "qps": 500,
  "duration_seconds": 60,
  "timeout_millis": 3000,
  "max_conns_per_host": 100
}
```

### 4.2 启动任务

```text
POST /api/tests/{id}/start
```

启动后，后端创建 `context.Context` 和 `cancel`，并由 task manager 保存运行状态。

### 4.3 停止任务

```text
POST /api/tests/{id}/stop
```

停止时调用对应任务的 `cancel()`，engine 收到 `ctx.Done()` 后退出 worker，并关闭空闲连接。

### 4.4 查询任务状态

```text
GET /api/tests/{id}
```

返回任务配置、运行状态、开始时间、结束时间。

### 4.5 列出所有任务

```text
GET /api/tests
```

返回所有压测任务的列表（不分页），含每个任务的配置与当前状态。

### 4.6 查询每秒指标历史

```text
GET /api/tests/{id}/metrics
```

返回该任务的所有每秒聚合桶（按 `second_offset` 升序），用于前端绘制历史曲线。

### 4.7 查询最终结果

```text
GET /api/tests/{id}/result
```

返回任务的最终聚合结果，含总请求数、成功/失败数、平均/最小/最大延迟，以及 P50/P90/P95/P99 分位数。任务结束后才有该记录，未结束或不存在返回 404。

### 4.8 实时指标推送

```text
GET /api/tests/{id}/events
```

使用 SSE 按秒推送聚合快照。

## 5. 数据库存储设计

第一版使用 SQLite + sqlc（根据 SQL 查询生成类型安全的 Go 代码）。

### 5.1 任务表

```sql
CREATE TABLE IF NOT EXISTS test_tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    method TEXT NOT NULL,
    url TEXT NOT NULL,
    headers_json TEXT NOT NULL DEFAULT '{}',
    body_type TEXT NOT NULL DEFAULT 'none',
    body_raw TEXT NOT NULL DEFAULT '',
    concurrency INTEGER NOT NULL,
    qps INTEGER NOT NULL,
    duration_seconds INTEGER NOT NULL,
    timeout_millis INTEGER NOT NULL DEFAULT 30000,
    max_conns_per_host INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL,
    created_at TEXT NOT NULL,
    started_at TEXT,
    finished_at TEXT
);
```

注：`timeout_millis` 与 `max_conns_per_host` 分别由迁移 003、002 加入；`max_conns_per_host = 0` 表示回退到 `concurrency`。时间字段以 RFC3339Nano 文本存储。

### 5.2 每秒聚合指标表

```sql
CREATE TABLE IF NOT EXISTS metrics_buckets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    test_id INTEGER NOT NULL,
    second_offset INTEGER NOT NULL,
    requests INTEGER NOT NULL,
    success INTEGER NOT NULL,
    failure INTEGER NOT NULL,
    avg_latency_ms REAL NOT NULL,
    min_latency_ms INTEGER NOT NULL,
    max_latency_ms INTEGER NOT NULL,
    errors_json TEXT NOT NULL DEFAULT '{}',
    status_code_json TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL,
    FOREIGN KEY (test_id) REFERENCES test_tasks(id)
);
```

每秒桶不含分位数——只记录 min/max/avg 与计数、状态码分布、错误类型分布。

### 5.3 最终结果表

```sql
CREATE TABLE IF NOT EXISTS test_results (
    test_id INTEGER NOT NULL,
    total_requests INTEGER NOT NULL,
    total_success INTEGER NOT NULL,
    total_failure INTEGER NOT NULL,
    avg_latency_ms REAL NOT NULL,
    min_latency_ms INTEGER NOT NULL,
    max_latency_ms INTEGER NOT NULL,
    p99_latency_ms INTEGER NOT NULL,
    p95_latency_ms INTEGER NOT NULL,
    p90_latency_ms INTEGER NOT NULL,
    p50_latency_ms INTEGER NOT NULL,
    errors_json TEXT NOT NULL DEFAULT '{}',
    status_code_json TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL,
    FOREIGN KEY (test_id) REFERENCES test_tasks(id)
);
```

分位数仅在任务结束时由任务级直方图计算一次写入此表。

## 6. 任务生命周期

任务状态建议定义为：

```text
Created   -> 已创建，未启动
Running   -> 正在压测
Stopped   -> 用户主动停止
Finished  -> 正常完成（duration 到达）
Failed    -> 执行失败
```

执行流程：

```text
创建任务
  -> 保存配置
  -> 启动任务
  -> 创建 http.Client/http.Transport
  -> 启动 worker
  -> worker 发送请求并记录结果
  -> metrics 每秒聚合
  -> 聚合结果写库并推送前端
  -> duration 到达或用户停止
  -> worker 退出
  -> 关闭空闲连接
  -> 保存最终结果
```

## 7. 关键实现约束

### 7.1 请求构造

每次请求都要重新构造 `http.Request`。

POST/PUT/PATCH 请求体不能复用同一个 reader，应每次使用新的 `bytes.Reader`：

```go
body := bytes.NewReader([]byte(cfg.Request.BodyRaw))
req, err := http.NewRequestWithContext(ctx, method, url, body)
```

### 7.2 响应体处理

每次请求结束后必须处理响应体：

```go
io.Copy(io.Discard, resp.Body)
resp.Body.Close()
```

否则连接无法复用，连接池会失效。

### 7.3 错误分类

错误不应只记录字符串。`engine` 对常见错误进行归类，`metrics` 按类型累计计数：

- `timeout` —— 请求超时（context deadline）
- `canceled` —— 请求被取消（context canceled，如用户停止任务）
- `dns` —— DNS 解析失败
- `connection_refused` —— 连接被拒绝
- `connection_reset` —— 连接被重置
- `too_many_open_files` —— 文件描述符耗尽
- `network` —— 其他网络错误（兜底）
- `request_build` —— 请求构造失败
- `read_body` —— 读取响应体失败
- `http_4xx` —— 4xx 状态码
- `http_5xx` —— 5xx 状态码
- `http_status` —— 其他异常状态码
- `business` —— 业务错误
- `client_overload` —— 客户端过载（worker 队列满，QPS 超过消费能力）

这样前端可以展示错误分布，用户也能判断瓶颈来自目标服务还是压测机自身。

### 7.4 系统保护

任务启动前进行保护性校验：

- 限制最大并发（`MaxConcurrency = 10000`）
- 限制最大 QPS（`MaxQPS = 100000`）
- 限制最大持续时间（`MaxDuration = 3600` 秒）
- 限制最大单主机连接数（`MaxMaxConnsPerHost = 10000`）
- 检查目标 URL 是否合法

## 8. 第一阶段实现顺序

建议按以下顺序开发：

1. 完成 `config` 配置结构和校验。
2. 完成 `metrics` 内存聚合，先支持总数、成功数、失败数、平均延迟。
3. 完成 `engine`，实现任务级 `http.Client` 复用和 worker 并发发送。
4. 加入 QPS 限制。
5. 加入按秒 bucket 聚合和 P95/P99。
6. 加入 Gin API 和 task manager。
7. 加入 SSE 实时推送。
8. 加入 SQLite/sqlx 持久化。
9. 编写基础单元测试和本地 mock server 压测测试。

第一阶段的最小完成标准：

```text
给定一个 TestConfig
  -> engine 能发起真实 HTTP 请求
  -> metrics 能聚合结果
  -> 控制台或 API 能看到请求数、成功数、失败数、平均延迟、P95/P99
```

## 9. 测试建议

需要覆盖以下测试：

- `metrics` 聚合结果是否正确
- P95/P99 计算是否正确
- status code 分布是否正确
- 错误分类是否正确
- `engine` 是否能在 context cancel 后停止
- `engine` 是否复用同一个 client/transport
- POST JSON 请求体是否正确发送
- QPS 限制是否大致符合预期
- 任务结束后是否调用 `CloseIdleConnections`

可以使用 `httptest.Server` 构造本地被压测接口，避免依赖外部服务。

## 10. 后续扩展

后续可以逐步加入：

- 变量替换，例如 `{{seq}}`、`{{timestamp}}`、`{{random_int}}`
- 登录前置请求和 token 提取
- 结果报告导出
- 请求结果采样
- CLI 模式
- 多 worker 分布式压测
- Redis 存储运行态指标
- 消息队列派发分布式任务

Redis 和消息队列不是单机版第一阶段必需能力，应在单机压测核心稳定后再引入。
