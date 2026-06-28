import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Task, TestConfig, MetricPoint, TestResult } from '@/types'
import * as api from '@/api/client'

export const useTaskStore = defineStore('task', () => {
  const tasks = ref<Task[]>([])
  const currentTask = ref<Task | null>(null)
  const metricPoints = ref<MetricPoint[]>([])
  const finalResult = ref<TestResult | null>(null)
  const loading = ref(false)
  const error = ref('')

  let es: EventSource | null = null

  async function fetchTasks() {
    loading.value = true
    error.value = ''
    try {
      const data = await api.listTasks()
      tasks.value = data.tasks ?? []
    } catch (e) {
      error.value = (e as Error).message
    } finally {
      loading.value = false
    }
  }

  async function createTask(cfg: TestConfig) {
    const data = await api.createTask(cfg)
    await fetchTasks()
    return data.task_id
  }

  async function fetchTask(id: number) {
    const data = await api.getTask(id)
    currentTask.value = data.task
    return data.task
  }

  async function startTask(id: number) {
    await api.startTask(id)
    await fetchTask(id)
  }

  async function stopTask(id: number) {
    await api.stopTask(id)
    await fetchTask(id)
  }

  async function fetchMetrics(id: number) {
    const data = await api.getMetrics(id)
    metricPoints.value = data.metrics ?? []
  }

  async function fetchResult(id: number) {
    try {
      const data = await api.getResult(id)
      finalResult.value = data.result
    } catch {
      finalResult.value = null
    }
  }

  function subscribeMetrics(id: number) {
    closeStream()
    metricPoints.value = []
    es = new EventSource(`/api/tests/${id}/events`)
    es.addEventListener('metric', (e: MessageEvent) => {
      try {
        const point = JSON.parse(e.data) as MetricPoint
        metricPoints.value.push(point)
        // 滚动保留近 300 点
        if (metricPoints.value.length > 300) {
          metricPoints.value = metricPoints.value.slice(-300)
        }
      } catch {
        // ignore
      }
    })
    es.onerror = () => {
      closeStream()
    }
  }

  function closeStream() {
    if (es) {
      es.close()
      es = null
    }
  }

  function reset() {
    metricPoints.value = []
    finalResult.value = null
    currentTask.value = null
    closeStream()
  }

  return {
    tasks,
    currentTask,
    metricPoints,
    finalResult,
    loading,
    error,
    fetchTasks,
    createTask,
    fetchTask,
    startTask,
    stopTask,
    fetchMetrics,
    fetchResult,
    subscribeMetrics,
    closeStream,
    reset,
  }
})
