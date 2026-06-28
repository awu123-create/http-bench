import type { Task, TestConfig, MetricPoint, TestResult, ApiResponse } from '@/types'

const BASE = '/api'

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(BASE + path, {
    method,
    headers: body !== undefined ? { 'Content-Type': 'application/json' } : undefined,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })
  const text = await res.text()
  let json: ApiResponse<T> | undefined
  try {
    json = text ? JSON.parse(text) : undefined
  } catch {
    throw new Error(`非 JSON 响应: ${text}`)
  }
  if (!res.ok) {
    const msg = json?.message ?? `HTTP ${res.status}`
    throw new Error(msg)
  }
  return json?.data as T
}

export function createTask(cfg: TestConfig): Promise<{ task_id: number }> {
  return request('POST', '/tests', cfg)
}

export function listTasks(): Promise<{ tasks: Task[] }> {
  return request('GET', '/tests')
}

export function getTask(id: number): Promise<{ task: Task }> {
  return request('GET', `/tests/${id}`)
}

export function startTask(id: number): Promise<{ result: string }> {
  return request('POST', `/tests/${id}/start`)
}

export function stopTask(id: number): Promise<{ result: string }> {
  return request('POST', `/tests/${id}/stop`)
}

export function getMetrics(id: number): Promise<{ metrics: MetricPoint[] }> {
  return request('GET', `/tests/${id}/metrics`)
}

export function getResult(id: number): Promise<{ result: TestResult }> {
  return request('GET', `/tests/${id}/result`)
}
