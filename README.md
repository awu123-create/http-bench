# HTTP Bench

一个单机版 HTTP 压测工具，支持高并发请求、QPS 限速、实时指标聚合与分位数计算，提供 Web UI 实时查看 QPS/延迟/状态码/错误分布图表。

## 功能特性

- **高并发压测**：可配置并发数、QPS、持续时间，支持 QPS 限速与不限速两种模式
- **多种请求体**：`none` / `raw`(JSON) / `x-www-form-urlencoded` / `form-data`(multipart 文件上传)
- **实时指标**：SSE 每秒推送聚合快照，前端实时绘制 4 张图表
- **分位数计算**：任务级固定线性直方图，O(1) 插入，计算 P50/P90/P95/P99
- **细粒度错误分类**：timeout / dns / connection_refused / connection_reset / too_many_open_files / network 等 14 类
- **连接池可控**：`max_conns_per_host` 独立配置，防 FD 耗尽
- **任务管理**：创建/启动/停止/查询/列表，SQLite 持久化，历史可回看
- **单容器部署**：前端 embed 进后端二进制，`docker run` 即用

## 快速开始（Docker）

```bash
docker run -d -p 8080:8080 -v hb-data:/data --name http-bench ghcr.io/awu123-create/http-bench:latest
```

浏览器打开 `http://localhost:8080`，创建压测任务、启动、查看实时图表。

数据持久化在 `hb-data` 卷中，容器删除后历史任务仍保留。

### docker compose

```yaml
services:
  http-bench:
    image: ghcr.io/awu123-create/http-bench:latest
    ports:
      - "8080:8080"
    volumes:
      - http-bench-data:/data
    restart: unless-stopped
volumes:
  http-bench-data:
```

## 本地开发

### 环境要求

- Go 1.25+
- Node.js 20+
- npm

### 启动后端

```bash
cd backend
go run ./cmd/server
```
后端监听 `:8080`，自动运行数据库迁移。

### 启动前端

```bash
cd frontend
npm install
npm run dev
```
前端开发服务器监听 `:5173`，自动代理 `/api` 到后端。

浏览器打开 `http://localhost:5173` 开发调试。

## 配置说明

创建压测任务时的配置字段：

| 字段 | 类型 | 说明 |
|---|---|---|
| `name` | string | 任务名称，留空自动生成 |
| `request.method` | string | HTTP 方法 |
| `request.url` | string | 目标 URL |
| `request.headers` | object | 自定义请求头 |
| `request.body_type` | string | 请求体类型：`none`/`raw`/`x-www-form-urlencoded`/`form-data` |
| `request.body_raw` | string | 请求体内容；`form-data` 时填 JSON 配方 |
| `concurrency` | int | 并发 worker 数（上限 10000） |
| `qps` | int | 全局 QPS 限制，0 表示不限速（上限 100000） |
| `duration` | int | 持续时间，秒（上限 3600） |
| `timeout_millis` | int | 单请求超时，毫秒（上限 60000） |
| `max_conns_per_host` | int | 单主机最大连接数，0 表示用 concurrency（上限 10000） |

### form-data 请求体格式

`body_type` 为 `form-data` 时，`body_raw` 填如下 JSON：

```json
{
  "fields": {"username": "alice"},
  "files": [
    {"name": "upload", "filename": "a.txt", "content": "文本内容"},
    {"name": "img", "filename": "b.bin", "content_base64": "AAECAw=="}
  ]
}
```

## API 接口

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/tests` | 创建压测任务 |
| GET | `/api/tests` | 列出所有任务 |
| GET | `/api/tests/:id` | 查询单个任务 |
| POST | `/api/tests/:id/start` | 启动任务 |
| POST | `/api/tests/:id/stop` | 停止任务 |
| GET | `/api/tests/:id/metrics` | 查询每秒指标历史 |
| GET | `/api/tests/:id/events` | SSE 实时指标推送 |
| GET | `/api/tests/:id/result` | 查询最终结果（含分位数） |

所有响应统一格式：

```json
{"message": "ok", "data": { ... }}
```

## 架构

```text
┌─────────────────────────────────────────────────┐
│                    浏览器                        │
│  Vue 3 + ECharts + Ant Design Vue               │
│  任务列表 / 创建表单 / 实时图表 / 最终报告        │
└────────────────────┬────────────────────────────┘
            REST API  │  SSE
┌────────────────────▼────────────────────────────┐
│                  后端 (Go)                       │
│  ┌─────────┐  ┌─────────┐  ┌────────────────┐   │
│  │  api    │  │  task   │  │   engine       │   │
│  │ router  │→ │ manager │→ │ worker goroutine│  │
│  └─────────┘  └────┬────┘  └───────┬────────┘   │
│                    │               │             │
│              ┌─────▼─────┐  ┌──────▼──────┐      │
│              │  storage  │  │  metrics    │      │
│              │  (sqlc)   │  │ collector + │      │
│              │  SQLite   │  │ histogram   │      │
│              └───────────┘  └─────────────┘      │
└─────────────────────────────────────────────────┘
```

**数据流**：worker 发请求 → collector 每秒聚合 → 写 SQLite + SSE 推前端 → 任务结束算分位数写最终结果。

## 技术栈

**后端**
- Go 1.25 + Gin（Web 框架）
- SQLite + sqlc（类型安全 SQL 代码生成）+ goose（迁移）
- modernc.org/sqlite（纯 Go SQLite，无 CGO 依赖）

**前端**
- Vue 3 + TypeScript + Vite
- Pinia（状态管理）+ Vue Router
- Ant Design Vue（UI 组件）
- ECharts + vue-echarts（图表）
- 原生 EventSource（SSE 消费）

**部署**
- Docker 多阶段构建，前端 embed 进后端二进制
- 单容器单端口，支持 amd64/arm64 双架构

## 项目结构

```text
.
├── backend/
│   ├── cmd/server/          # 入口
│   ├── internal/
│   │   ├── api/             # 路由、handler、响应封装
│   │   ├── config/          # 配置结构与校验
│   │   ├── engine/          # 压测引擎、请求构造、错误分类
│   │   ├── metrics/         # 指标聚合、直方图、分位数
│   │   ├── storage/         # SQLite 访问 + sqlc 生成代码
│   │   ├── static/          # embed 前端产物
│   │   └── task/            # 任务生命周期管理、SSE、结果聚合
│   ├── migrations/          # 数据库迁移
│   └── README.md            # 后端开发文档
├── frontend/
│   ├── src/
│   │   ├── api/             # HTTP 调用封装
│   │   ├── components/      # 表单、图表组件
│   │   ├── composables/     # SSE 封装
│   │   ├── router/          # 路由
│   │   ├── stores/          # Pinia 状态
│   │   ├── types/           # TypeScript 类型
│   │   └── views/           # 任务列表、任务详情页
│   └── package.json
├── Dockerfile               # 多阶段构建
├── docker-compose.yml
└── .dockerignore
```

## 测试

```bash
cd backend
go test ./... -count=2
```

覆盖：配置校验、引擎执行、QPS 限制、分位数计算、错误分类、任务管理、API 端点、存储迁移。

## 构建镜像

本地构建：

```bash
docker build -t http-bench .
```

多架构构建并推送到 ghcr.io：

```bash
docker buildx create --name multiarch --use
docker buildx build --platform linux/amd64,linux/arm64 \
  -t ghcr.io/awu123-create/http-bench:latest \
  --push .
```

## 任务状态

| 状态 | 说明 |
|---|---|
| `Created` | 已创建，未启动 |
| `Running` | 正在压测 |
| `Stopped` | 用户主动停止 |
| `Finished` | 正常完成（duration 到达） |
| `Failed` | 执行失败 |

## 环境变量

| 变量 | 默认值 | 说明 |
|---|---|---|
| `HTTP_BENCH_ADDR` | `:8080` | 监听地址 |
| `HTTP_BENCH_DB` | `data/http-bench.db` | SQLite 数据库路径 |
| `HTTP_BENCH_MIGRATIONS` | `migrations` | 迁移文件目录 |
