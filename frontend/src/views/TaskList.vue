<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { useTaskStore } from '@/stores/task'
import CreateTaskForm from '@/components/CreateTaskForm.vue'
import type { TestConfig } from '@/types'

const router = useRouter()
const store = useTaskStore()
const modalVisible = ref(false)
const creating = ref(false)

const columns = [
  { title: 'ID', dataIndex: 'id', width: 80 },
  { title: '名称', dataIndex: 'name' },
  { title: '状态', dataIndex: 'status', width: 120 },
  { title: '操作', key: 'action', width: 240 },
]

const statusColor: Record<string, string> = {
  Created: 'default',
  Running: 'processing',
  Stopped: 'warning',
  Finished: 'success',
  Failed: 'error',
}

onMounted(() => {
  store.fetchTasks()
})

function openCreate() {
  modalVisible.value = true
}

async function handleCreate(cfg: TestConfig) {
  creating.value = true
  try {
    const id = await store.createTask(cfg)
    message.success(`已创建任务 #${id}`)
    modalVisible.value = false
    router.push(`/tasks/${id}`)
  } catch (e) {
    message.error((e as Error).message)
  } finally {
    creating.value = false
  }
}

async function handleStart(id: number) {
  try {
    await store.startTask(id)
    message.success('已启动')
    router.push(`/tasks/${id}`)
  } catch (e) {
    message.error((e as Error).message)
  }
}

async function handleStop(id: number) {
  try {
    await store.stopTask(id)
    message.success('已停止')
  } catch (e) {
    message.error((e as Error).message)
  }
}

function goDetail(id: number) {
  router.push(`/tasks/${id}`)
}
</script>

<template>
  <div>
    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px">
      <h2 style="margin: 0">压测任务</h2>
      <a-space>
        <a-button @click="store.fetchTasks()">刷新</a-button>
        <a-button type="primary" @click="openCreate">新建任务</a-button>
      </a-space>
    </div>

    <a-alert v-if="store.error" :message="store.error" type="error" style="margin-bottom: 16px" />

    <a-table
      :columns="columns"
      :data-source="store.tasks"
      row-key="id"
      :loading="store.loading"
      :pagination="false"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.dataIndex === 'status'">
          <a-tag :color="statusColor[record.status] ?? 'default'">{{ record.status }}</a-tag>
        </template>
        <template v-else-if="column.key === 'action'">
          <a-space>
            <a-button size="small" @click="goDetail(record.id)">详情</a-button>
            <a-button
              v-if="record.status === 'Created' || record.status === 'Finished' || record.status === 'Stopped' || record.status === 'Failed'"
              size="small"
              type="primary"
              @click="handleStart(record.id)"
            >启动</a-button>
            <a-button
              v-if="record.status === 'Running'"
              size="small"
              danger
              @click="handleStop(record.id)"
            >停止</a-button>
          </a-space>
        </template>
      </template>
    </a-table>

    <a-modal v-model:open="modalVisible" title="新建压测任务" :footer="null" width="720px" :confirm-loading="creating">
      <CreateTaskForm @submit="handleCreate" @cancel="modalVisible = false" />
    </a-modal>
  </div>
</template>
