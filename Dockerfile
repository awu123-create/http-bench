# 阶段 1: 构建前端
FROM docker.m.daocloud.io/library/node:20-alpine AS frontend
WORKDIR /fe
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# 阶段 2: 构建后端（嵌入前端产物）
FROM docker.m.daocloud.io/library/golang:1.25-alpine AS backend
WORKDIR /be
COPY backend/go.* ./
RUN go mod download
COPY backend/ ./
# 用真实前端构建产物覆盖占位 dist
COPY --from=frontend /fe/dist ./internal/static/dist
RUN CGO_ENABLED=0 go build -o /http-bench ./cmd/server

# 阶段 3: 最小运行时镜像
FROM docker.m.daocloud.io/library/alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=backend /http-bench /usr/local/bin/http-bench
COPY backend/migrations /migrations
ENV HTTP_BENCH_ADDR=:8080 \
    HTTP_BENCH_DB=/data/http-bench.db \
    HTTP_BENCH_MIGRATIONS=/migrations
RUN mkdir -p /data
EXPOSE 8080
VOLUME ["/data"]
ENTRYPOINT ["http-bench"]
