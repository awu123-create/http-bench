<script setup lang="ts">
import { computed, watch } from 'vue'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart, PieChart, BarChart } from 'echarts/charts'
import {
  TitleComponent,
  TooltipComponent,
  GridComponent,
  LegendComponent,
  DataZoomComponent,
} from 'echarts/components'
import type { MetricPoint } from '@/types'

use([CanvasRenderer, LineChart, PieChart, BarChart, TitleComponent, TooltipComponent, GridComponent, LegendComponent, DataZoomComponent])

const props = defineProps<{ points: MetricPoint[] }>()

const xData = computed(() => props.points.map((p) => p.second_offset))

const qpsOption = computed(() => ({
  title: { text: '每秒请求数', left: 'center', textStyle: { fontSize: 14 } },
  tooltip: { trigger: 'axis' },
  legend: { data: ['成功', '失败'], bottom: 0 },
  grid: { left: 50, right: 20, top: 40, bottom: 50 },
  xAxis: { type: 'category', data: xData.value, name: '秒' },
  yAxis: { type: 'value' },
  series: [
    { name: '成功', type: 'line', smooth: true, data: props.points.map((p) => p.snapshot.success) },
    { name: '失败', type: 'line', smooth: true, areaStyle: {}, data: props.points.map((p) => p.snapshot.failed) },
  ],
}))

const latencyOption = computed(() => ({
  title: { text: '延迟 (ms)', left: 'center', textStyle: { fontSize: 14 } },
  tooltip: { trigger: 'axis' },
  legend: { data: ['avg', 'min', 'max'], bottom: 0 },
  grid: { left: 50, right: 20, top: 40, bottom: 50 },
  xAxis: { type: 'category', data: xData.value, name: '秒' },
  yAxis: { type: 'value' },
  series: [
    { name: 'avg', type: 'line', smooth: true, data: props.points.map((p) => p.snapshot.avg_latency) },
    { name: 'min', type: 'line', data: props.points.map((p) => p.snapshot.min_latency) },
    { name: 'max', type: 'line', data: props.points.map((p) => p.snapshot.max_latency) },
  ],
}))

const statusPieData = computed(() => {
  const agg: Record<string, number> = {}
  for (const p of props.points) {
    for (const [code, cnt] of Object.entries(p.snapshot.status_codes ?? {})) {
      agg[code] = (agg[code] ?? 0) + cnt
    }
  }
  return Object.entries(agg).map(([name, value]) => ({ name, value }))
})

const statusOption = computed(() => ({
  title: { text: '状态码分布', left: 'center', textStyle: { fontSize: 14 } },
  tooltip: { trigger: 'item' },
  legend: { bottom: 0 },
  series: [{ type: 'pie', radius: ['40%', '70%'], data: statusPieData.value }],
}))

const errorKeys = [
  'timeout_errors',
  'network_errors',
  'dns_errors',
  'connection_refused_errors',
  'connection_reset_errors',
  'too_many_open_files_errors',
  'read_body_errors',
  'http_4xx_errors',
  'http_5xx_errors',
  'http_status_errors',
  'business_errors',
  'client_overload_errors',
  'canceled_errors',
  'request_build_errors',
]

const errorOption = computed(() => {
  const snap = (s: MetricPoint['snapshot']): Record<string, number> => s as unknown as Record<string, number>
  const present = errorKeys.filter((k) => props.points.some((p) => snap(p.snapshot)[k] > 0))
  return {
    title: { text: '错误类型分布', left: 'center', textStyle: { fontSize: 14 } },
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    legend: { bottom: 0, data: present },
    grid: { left: 50, right: 20, top: 40, bottom: 60 },
    xAxis: { type: 'category', data: xData.value, name: '秒' },
    yAxis: { type: 'value' },
    series: present.map((k) => ({
      name: k.replace('_errors', ''),
      type: 'bar',
      stack: 'err',
      data: props.points.map((p) => snap(p.snapshot)[k]),
    })),
  }
})

// 强制图表在 points 变化时刷新
const refreshKey = computed(() => props.points.length)
watch(refreshKey, () => {
  // 触发 computed 重算
})
</script>

<template>
  <a-row :gutter="16">
    <a-col :span="12">
      <VChart :option="qpsOption" autoresize style="height: 300px" />
    </a-col>
    <a-col :span="12">
      <VChart :option="latencyOption" autoresize style="height: 300px" />
    </a-col>
  </a-row>
  <a-row :gutter="16" style="margin-top: 16px">
    <a-col :span="12">
      <VChart :option="statusOption" autoresize style="height: 300px" />
    </a-col>
    <a-col :span="12">
      <VChart :option="errorOption" autoresize style="height: 300px" />
    </a-col>
  </a-row>
</template>
