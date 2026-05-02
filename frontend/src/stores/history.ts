import { defineStore } from 'pinia'
import { ref } from 'vue'
import * as api from '../api/client'
import { wsManager } from '../api/ws'
import type { PaginatedResult } from '../types'

export const useHistoryStore = defineStore('history', () => {
  const history = ref<Map<string, PaginatedResult>>(new Map())

  async function fetchHistory(instanceId: string, page = 1, perPage = 50) {
    const result = await api.listHistory(instanceId, page, perPage)
    history.value.set(instanceId, result)
  }

  async function deleteRecord(instanceId: string, gid: string, deleteFiles?: boolean) {
    await api.deleteHistoryRecord(instanceId, gid, deleteFiles)
    const current = history.value.get(instanceId)
    if (current) {
      current.records = current.records.filter((r) => r.gid !== gid)
      current.total -= 1
      history.value.set(instanceId, current)
    }
  }

  // When a task_completed WS message arrives, refresh history from server
  wsManager.on('task_completed', (msg: any) => {
    const id = msg.instance_id
    // Refresh the full history page from server to stay consistent
    fetchHistory(id)
  })

  return {
    history,
    fetchHistory,
    deleteRecord,
  }
})