export interface RequestConfig {
  method: string
  url: string
  headers: Record<string, string>
  body_type: string
  body_raw: string
}

export interface TestConfig {
  name: string
  request: RequestConfig
  concurrency: number
  qps: number
  duration: number
  timeout_millis: number
  max_conns_per_host: number
}

export interface Task {
  id: number
  name: string
  config: TestConfig
  status: string
}

export interface Snapshot {
  total_requests: number
  success: number
  failed: number
  avg_latency: number
  min_latency: number
  max_latency: number
  request_build_errors: number
  timeout_errors: number
  canceled_errors: number
  network_errors: number
  dns_errors: number
  connection_refused_errors: number
  connection_reset_errors: number
  too_many_open_files_errors: number
  read_body_errors: number
  http_4xx_errors: number
  http_5xx_errors: number
  http_status_errors: number
  business_errors: number
  client_overload_errors: number
  status_codes: Record<string, number>
}

export interface MetricPoint {
  second_offset: number
  snapshot: Snapshot
}

export interface TestResult {
  test_id: number
  total_requests: number
  total_success: number
  total_failure: number
  avg_latency_ms: number
  min_latency_ms: number
  max_latency_ms: number
  p99_latency_ms: number
  p95_latency_ms: number
  p90_latency_ms: number
  p50_latency_ms: number
  errors_json: string
  status_code_json: string
  created_at: string
}

interface ApiResponse<T> {
  message: string
  data: T
}

export type { ApiResponse }
