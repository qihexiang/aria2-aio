import { defineStore } from 'pinia'
import { ref } from 'vue'
import * as api from '../api/client'
import { wsManager } from '../api/ws'
import type { TaskProgress, AddTaskRequest } from '../types'

export const useTaskStore = defineStore('tasks', () => {
  // All in-progress tasks per instance (active + waiting + paused)
  const tasks = ref<Map<string, TaskProgress[]>>(new Map())

  async function fetchTasks(instanceId: string) {
    const [active, waiting, stopped] = await Promise.all([
      api.listActiveTasks(instanceId),
      api.listWaitingTasks(instanceId),
      api.listStoppedTasks(instanceId),
    ])
    const merged = [...(active || []), ...(waiting || []), ...(stopped || [])]
    tasks.value.set(instanceId, merged)
  }

  async function addTask(instanceId: string, data: AddTaskRequest) {
    return await api.addTask(instanceId, data)
  }

  async function pauseTask(instanceId: string, gid: string) {
    await api.pauseTask(instanceId, gid)
  }

  async function unpauseTask(instanceId: string, gid: string) {
    await api.unpauseTask(instanceId, gid)
  }

  async function removeTask(instanceId: string, gid: string, deleteFiles?: boolean) {
    await api.removeTask(instanceId, gid, deleteFiles)
    removeLocal(instanceId, gid)
  }

  function removeLocal(instanceId: string, gid: string) {
    const current = tasks.value.get(instanceId) || []
    tasks.value.set(instanceId, current.filter((t) => t.gid !== gid))
  }

  // WebSocket: receive full task list from tracker poll
  wsManager.on('task_progress', (msg: any) => {
    const id = msg.instance_id
    const taskList: TaskProgress[] = msg.data || []
    // Keep all in-progress tasks (active, waiting, paused)
    // Exclude completed/error tasks that have moved to history
    const inProgress = taskList.filter((t) =>
      t.status === 'active' || t.status === 'waiting' || t.status === 'paused'
    )
    tasks.value.set(id, inProgress)
  })

  // WebSocket: a task was removed from active list (completed or error)
  wsManager.on('task_event', (msg: any) => {
    const id = msg.instance_id
    const event = msg.data?.event
    const gid = msg.data?.gid
    if (event === 'onDownloadComplete' || event === 'onDownloadError') {
      removeLocal(id, gid)
    }
  })

  return {
    tasks,
    fetchTasks,
    addTask,
    pauseTask,
    unpauseTask,
    removeTask,
  }
})