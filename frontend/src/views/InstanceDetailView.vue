<template>
  <div>
    <div class="header-row">
      <h1>{{ instance?.name || 'Instance' }}</h1>
      <div class="actions-row">
        <button v-if="instance?.status === 'stopped'" class="btn btn-success" @click="start">Start</button>
        <button v-if="instance?.status === 'running'" class="btn btn-warning" @click="stop">Stop</button>
        <button v-if="instance?.status === 'running'" class="btn btn-primary" @click="restart">Restart</button>
        <button class="btn btn-danger" @click="confirmDelete">Delete</button>
        <router-link to="/instances" class="btn btn-sm" style="background: var(--bg)">Back</router-link>
      </div>
    </div>

    <div class="card">
      <span :class="['badge', `badge-${instance?.status}`]">{{ instance?.status }}</span>
      <span style="margin-left: 12px; color: var(--text-muted)">
        Port {{ instance?.rpc_port }} | PID {{ instance?.pid || 'N/A' }}
      </span>
    </div>

    <template v-if="instance?.status === 'running'">
      <div class="header-row" style="margin-top: 24px">
        <h2>Tasks</h2>
        <button class="btn btn-primary btn-sm" @click="showAddTask = true">Add Task</button>
      </div>

      <div v-if="showAddTask" class="card" style="margin-bottom: 16px">
        <div class="form-group">
          <label>URLs (comma-separated)</label>
          <input v-model="newTaskUrls" placeholder="https://example.com/file.zip" />
        </div>
        <div class="actions-row">
          <button class="btn btn-primary btn-sm" @click="addTask">Add</button>
          <button class="btn btn-sm" style="background: var(--bg)" @click="showAddTask = false">Cancel</button>
        </div>
      </div>

      <table v-if="taskList.length > 0">
        <thead>
          <tr>
            <th>Name</th>
            <th>Progress</th>
            <th>Speed</th>
            <th>Status</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="task in taskList" :key="task.gid">
            <td style="max-width: 300px; overflow: hidden; text-overflow: ellipsis">{{ task.name }}</td>
            <td>
              <div class="progress-bar">
                <div class="progress-bar-fill" :style="{ width: progressPercent(task) + '%' }"></div>
                <span class="progress-bar-text">{{ progressPercent(task).toFixed(1) }}%</span>
              </div>
            </td>
            <td>{{ formatSpeed(task.download_speed) }}</td>
            <td>
              <span :class="['badge', statusBadgeClass(task.status)]">{{ task.status }}</span>
            </td>
            <td>
              <div class="actions-row">
                <button v-if="task.status === 'active'" class="btn btn-warning btn-sm" @click="taskStore.pauseTask(instanceId!, task.gid)">Pause</button>
                <button v-if="task.status === 'paused'" class="btn btn-success btn-sm" @click="taskStore.unpauseTask(instanceId!, task.gid)">Resume</button>
                <button class="btn btn-danger btn-sm" @click="confirmRemoveTask(task)">Remove</button>
                <button class="btn btn-danger btn-sm" @click="confirmRemoveTaskWithFiles(task)">Remove+Files</button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>

      <div v-if="taskList.length === 0" style="text-align: center; padding: 20px; color: var(--text-muted)">
        No active tasks.
      </div>

      <div class="header-row" style="margin-top: 24px">
        <h2>Task History</h2>
      </div>

      <table v-if="(historyResult?.records?.length ?? 0) > 0">
        <thead>
          <tr>
            <th>Name</th>
            <th>Status</th>
            <th>Size</th>
            <th>Completed</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="record in historyResult?.records" :key="record.gid">
            <td style="max-width: 300px; overflow: hidden; text-overflow: ellipsis">{{ record.name }}</td>
            <td><span :class="['badge', record.status === 'complete' ? 'badge-running' : 'badge-error']">{{ record.status }}</span></td>
            <td>{{ formatSize(record.total_length) }}</td>
            <td>{{ formatDate(record.completed_at) }}</td>
            <td>
              <div class="actions-row">
                <button class="btn btn-danger btn-sm" @click="confirmDeleteRecord(record)">Delete</button>
                <button class="btn btn-danger btn-sm" @click="confirmDeleteRecordWithFiles(record)">Delete+Files</button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>

      <div v-else style="text-align: center; padding: 20px; color: var(--text-muted)">
        No completed tasks yet.
      </div>
    </template>

    <div v-else style="text-align: center; padding: 40px; color: var(--text-muted)">
      Instance is not running. Start it to manage tasks.
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useInstanceStore } from '../stores/instances'
import { useTaskStore } from '../stores/tasks'
import { useHistoryStore } from '../stores/history'
import type { TaskProgress } from '../types'

const route = useRoute()
const router = useRouter()
const instanceStore = useInstanceStore()
const taskStore = useTaskStore()
const historyStore = useHistoryStore()

const instanceId = computed(() => route.params.id as string)
const instance = computed(() => instanceStore.instances.find((i) => i.id === instanceId.value))

const showAddTask = ref(false)
const newTaskUrls = ref('')

const taskList = computed(() => taskStore.tasks.get(instanceId.value) || [])
const historyResult = computed(() => historyStore.history.get(instanceId.value))

function statusBadgeClass(status: string): string {
  if (status === 'active') return 'badge-running'
  if (status === 'paused') return 'badge-stopped'
  if (status === 'waiting') return 'badge-stopped'
  return 'badge-error'
}

function progressPercent(task: TaskProgress): number {
  if (task.total_length === 0) return 0
  return (task.completed_length / task.total_length) * 100
}

function formatSpeed(bytes: number): string {
  if (bytes === 0) return '0 B/s'
  const units = ['B/s', 'KB/s', 'MB/s', 'GB/s']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`
}

function formatSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`
}

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleString()
}

async function start() {
  await instanceStore.startInstance(instanceId.value!)
}

async function stop() {
  await instanceStore.stopInstance(instanceId.value!)
}

async function restart() {
  await instanceStore.restartInstance(instanceId.value!)
}

function confirmDelete() {
  if (confirm('Delete this instance? All data will be lost.')) {
    instanceStore.deleteInstance(instanceId.value!)
    router.push('/instances')
  }
}

async function addTask() {
  if (!newTaskUrls.value.trim()) return
  const uris = newTaskUrls.value.split(',').map((u) => u.trim()).filter(Boolean)
  await taskStore.addTask(instanceId.value!, { type: 'uri', uris })
  showAddTask.value = false
  newTaskUrls.value = ''
}

function confirmRemoveTask(task: TaskProgress) {
  const msg = 'Remove this task?\n\nClick OK to remove the task record only.\nClick Cancel to keep the task.'
  if (confirm(msg)) {
    taskStore.removeTask(instanceId.value!, task.gid, false)
  }
}

function confirmRemoveTaskWithFiles(task: TaskProgress) {
  const msg = 'Remove this task AND delete all downloaded files?\n\nThis cannot be undone!'
  if (confirm(msg)) {
    taskStore.removeTask(instanceId.value!, task.gid, true)
  }
}

function confirmDeleteRecord(record: any) {
  const msg = 'Delete this history record?\n\nClick OK to delete the record only.\nClick Cancel to keep the record.'
  if (confirm(msg)) {
    historyStore.deleteRecord(instanceId.value!, record.gid, false)
  }
}

function confirmDeleteRecordWithFiles(record: any) {
  const msg = 'Delete this history record AND delete all downloaded files?\n\nThis cannot be undone!'
  if (confirm(msg)) {
    historyStore.deleteRecord(instanceId.value!, record.gid, true)
  }
}

onMounted(async () => {
  await instanceStore.fetchInstances()
  if (instance.value?.status === 'running') {
    await taskStore.fetchTasks(instanceId.value!)
    await historyStore.fetchHistory(instanceId.value!)
  }
})

watch(() => instance.value?.status, (status) => {
  if (status === 'running') {
    taskStore.fetchTasks(instanceId.value!)
    historyStore.fetchHistory(instanceId.value!)
  }
})
</script>