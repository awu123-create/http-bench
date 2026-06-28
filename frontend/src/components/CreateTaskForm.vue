<script setup lang="ts">
import { ref, reactive, computed } from 'vue'
import { Form, message } from 'ant-design-vue'
import type { TestConfig } from '@/types'

const emit = defineEmits<{ (e: 'submit', cfg: TestConfig): void; (e: 'cancel'): void }>()

const methods = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS']
const bodyTypes = [
  { label: 'none', value: 'none' },
  { label: 'raw (JSON 文本)', value: 'raw' },
  { label: 'x-www-form-urlencoded', value: 'x-www-form-urlencoded' },
  { label: 'form-data (JSON 配方)', value: 'form-data' },
]

const formState = reactive<TestConfig>({
  name: '',
  request: {
    method: 'GET',
    url: '',
    headers: {},
    body_type: 'none',
    body_raw: '',
  },
  concurrency: 10,
  qps: 0,
  duration: 10,
  timeout_millis: 30000,
  max_conns_per_host: 0,
})

const headersText = ref('{}')

const bodyRawPlaceholder = computed(() => {
  if (formState.request.body_type === 'form-data') {
    return '{"fields":{"k":"v"},"files":[{"name":"file","filename":"a.txt","content":"hi"}]}'
  }
  return '原始 body'
})

const { validate, validateInfos } = Form.useForm(formState, {
  url: [{ required: true, message: '请输入 URL' }],
  concurrency: [{ required: true, type: 'number', min: 1, message: '并发数 ≥ 1' }],
  duration: [{ required: true, type: 'number', min: 1, message: '时长 ≥ 1 秒' }],
  timeout_millis: [{ required: true, type: 'number', min: 1, message: '超时 ≥ 1 毫秒' }],
})

async function handleSubmit() {
  try {
    await validate()
  } catch {
    return
  }
  let headers: Record<string, string>
  try {
    headers = JSON.parse(headersText.value || '{}')
  } catch {
    message.error('headers 不是合法 JSON')
    return
  }
  emit('submit', { ...formState, request: { ...formState.request, headers } })
}
</script>

<template>
  <a-form layout="vertical" :model="formState">
    <a-form-item label="任务名称">
      <a-input v-model:value="formState.name" placeholder="留空则自动生成" />
    </a-form-item>
    <a-row :gutter="16">
      <a-col :span="6">
        <a-form-item label="HTTP 方法">
          <a-select v-model:value="formState.request.method">
            <a-select-option v-for="m in methods" :key="m" :value="m">{{ m }}</a-select-option>
          </a-select>
        </a-form-item>
      </a-col>
      <a-col :span="18">
        <a-form-item label="目标 URL" name="url" v-bind="validateInfos.url">
          <a-input v-model:value="formState.request.url" placeholder="http://localhost:8081/api" />
        </a-form-item>
      </a-col>
    </a-row>
    <a-form-item label="请求头 (JSON)">
      <a-textarea v-model:value="headersText" :rows="2" placeholder='{"Authorization":"Bearer x"}' />
    </a-form-item>
    <a-row :gutter="16">
      <a-col :span="8">
        <a-form-item label="请求体类型">
          <a-select v-model:value="formState.request.body_type">
            <a-select-option v-for="b in bodyTypes" :key="b.value" :value="b.value">{{ b.label }}</a-select-option>
          </a-select>
        </a-form-item>
      </a-col>
      <a-col :span="16">
        <a-form-item label="请求体内容">
          <a-textarea
            v-model:value="formState.request.body_raw"
            :rows="3"
            :placeholder="bodyRawPlaceholder"
          />
        </a-form-item>
      </a-col>
    </a-row>
    <a-row :gutter="16">
      <a-col :span="6">
        <a-form-item label="并发数" name="concurrency" v-bind="validateInfos.concurrency">
          <a-input-number v-model:value="formState.concurrency" :min="1" style="width: 100%" />
        </a-form-item>
      </a-col>
      <a-col :span="6">
        <a-form-item label="QPS (0=不限)">
          <a-input-number v-model:value="formState.qps" :min="0" style="width: 100%" />
        </a-form-item>
      </a-col>
      <a-col :span="6">
        <a-form-item label="时长 (秒)" name="duration" v-bind="validateInfos.duration">
          <a-input-number v-model:value="formState.duration" :min="1" style="width: 100%" />
        </a-form-item>
      </a-col>
      <a-col :span="6">
        <a-form-item label="超时 (毫秒)" name="timeout_millis" v-bind="validateInfos.timeout_millis">
          <a-input-number v-model:value="formState.timeout_millis" :min="1" style="width: 100%" />
        </a-form-item>
      </a-col>
    </a-row>
    <a-form-item label="单主机最大连接数 (0=用并发数)">
      <a-input-number v-model:value="formState.max_conns_per_host" :min="0" style="width: 100%" />
    </a-form-item>
    <a-space>
      <a-button type="primary" @click="handleSubmit">创建</a-button>
      <a-button @click="emit('cancel')">取消</a-button>
    </a-space>
  </a-form>
</template>
