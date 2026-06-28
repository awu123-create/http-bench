<script setup lang="ts">
import { computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { message } from 'ant-design-vue'
import { useTaskStore } from '@/stores/task'
import MetricsCharts from '@/components/MetricsCharts.vue'

const route = useRoute()
const store = useTaskStore()
const taskId = computed(() => Number(route.params.id))
let pollTimer: ReturnType<typeof setInterval> | null = null

const task = computed(() => store.currentTask)
const points = computed(() => store.metricPoints)
const result = computed(() => store.finalResult)

const isRunning = computed(() => task.value?.status === 'Running')

async function loadAll() {
  try {
    await store.fetchTask(taskId.value)
    const status = store.currentTask?.status
    if (status === 'Running') {
      // 开 SSE 实时
      store.subscribeMetrics(taskId.value)
      startPolling()
    } else {
      // 已结束：拉历史 + 结果
      await store.fetchMetrics(taskId.value)
      await store.fetchResult(taskId.value)
    }
  } catch (e) {
    message.error((e as Error).message)
  }
}

function startPolling() {
  stopPolling()
  pollTimer = setInterval(async () => {
    try {
      await store.fetchTask(taskId.value)
      if (store.currentTask?.status !== 'Running') {
        stopPolling()
        store.closeStream()
        await store.fetchResult(taskId.value)
      }
    } catch {
      // ignore poll error
    }
  }, 1500)
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

async function handleStart() {
  try {
    await store.startTask(taskId.value)
    store.metricPoints = []
    store.subscribeMetrics(taskId.value)
    startPolling()
    message.success('已启动')
  } catch (e) {
    message.error((e as Error).message)
  }
}

async function handleStop() {
  try {
    await store.stopTask(taskId.value)
    message.success('已停止')
  } catch (e) {
    message.error((e as Error).message)
  }
}

onMounted(() => {
  store.reset()
  loadAll()
})

onUnmounted(() => {
  stopPolling()
  store.closeStream()
})

watch(taskId, () => {
  stopPolling()
  store.closeStream()
  store.reset()
  loadAll()
})

const statusColor: Record<string, string> = {
  Created: 'default',
  Running: 'processing',
  Stopped: 'warning',
  Finished: 'success',
  Failed: 'error',
}
</script>

<template>
  <div>
    <a-page-header
      :title="task?.name ?? `任务 #${taskId}`"
      :sub-title="`ID: ${taskId}`"
      @back="$router.push('/')"
    >
      <template #tags>
        <a-tag v-if="task" :color="statusColor[task.status] ?? 'default'">{{ task.status }}</a-tag>
      </template>
      <template #extra>
        <a-space>
          <a-button
            v-if="!isRunning && task"
            type="primary"
            @click="handleStart"
          >启动</a-button>
          <a-button v-if="isRunning" danger @click="handleStop">停止</a-button>
          <a-button @click="loadAll">刷新</a-button>
        </a-space>
      </template>
    </a-page-header>

    <div v-if="task" style="margin: 16px 0">
      <a-descriptions :column="3" size="small" bordered>
        <a-descriptions-item label="方法">{{ task.config.request.method }}</a-descriptions-item>
        <a-descriptions-item label="URL" :span="2">{{ task.config.request.url }}</a-descriptions-item>
        <a-descriptions-item label="并发">{{ task.config.concurrency }}</a-descriptions-item>
        <a-descriptions-item label="QPS">{{ task.config.qps }}</a-descriptions-item>
        <a-descriptions-item label="时长(秒)">{{ task.config.duration }}</a-descriptions-item>
        <a-descriptions-item label="超时(ms)">{{ task.config.timeout_millis }}</a-descriptions-item>
        <a-descriptions-item label="最大连接数">{{ task.config.max_conns_per_host }}</a-descriptions-item>
        <a-descriptions-item label="body_type">{{ task.config.request.body_type }}</a-descriptions-item>
      </a-descriptions>
    </div>

    <a-divider />

    <h3>实时指标</h3>
    <a-empty v-if="points.length === 0" description="暂无指标数据" />
    <MetricsCharts v-else :points="points" />

    <template v-if="result">
      <a-divider />
      <h3>最终结果</h3>
      <a-descriptions :column="4" bordered>
        <a-descriptions-item label="总请求数">{{ result.total_requests }}</a-descriptions-item>
        <a-descriptions-item label="成功">{{ result.total_success }}</a-descriptions-item>
        <a-descriptions-item label="失败">{{ result.total_failure }}</a-descriptions-item>
        <a-descriptions-item label="平均延迟(ms)">{{ result.avg_latency_ms.toFixed(2) }}</a-descriptions-item>
        <a-descriptions-item label="最小延迟(ms)">{{ result.min_latency_ms }}</a-descriptions-item>
        <a-descriptions-item label="最大延迟(ms)">{{ result.max_latency_ms }}</a-descriptions-item>
        <a-descriptions-item label="P50">{{ result.p50_latency_ms }}</a-descriptions-item>
        <a-descriptions-item label="P90">{{ result.p90_latency_ms }}</a-descriptions-item>
        <a-descriptions-item label="P95">{{ result.p95_latency_ms }}</a-descriptions-item>
        <a-descriptions-item label="P99">{{ result.p99_latency_ms }}</a-descriptions-item>
      </a-descriptions>
    </template>
  </div>
</template>
