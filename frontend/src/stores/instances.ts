import { defineStore } from 'pinia'
import { ref } from 'vue'
import * as api from '../api/client'
import { wsManager } from '../api/ws'
import type { Instance, CreateInstanceRequest } from '../types'

export const useInstanceStore = defineStore('instances', () => {
  const instances = ref<Instance[]>([])
  const loading = ref(false)

  async function fetchInstances() {
    loading.value = true
    try {
      instances.value = await api.listInstances()
    } finally {
      loading.value = false
    }
  }

  async function createInstance(data: CreateInstanceRequest) {
    const inst = await api.createInstance(data)
    instances.value.push(inst)
    return inst
  }

  async function startInstance(id: string) {
    const inst = await api.startInstance(id)
    updateLocal(id, inst)
  }

  async function stopInstance(id: string) {
    const inst = await api.stopInstance(id)
    updateLocal(id, inst)
  }

  async function restartInstance(id: string) {
    const inst = await api.restartInstance(id)
    updateLocal(id, inst)
  }

  async function deleteInstance(id: string) {
    await api.deleteInstance(id)
    instances.value = instances.value.filter((i) => i.id !== id)
  }

  function updateLocal(id: string, inst: Instance) {
    const idx = instances.value.findIndex((i) => i.id === id)
    if (idx >= 0) {
      instances.value[idx] = inst
    }
  }

  // WebSocket handler
  wsManager.on('instance_status', (msg: any) => {
    const id = msg.instance_id
    const status = msg.data?.status
    if (status === 'deleted') {
      instances.value = instances.value.filter((i) => i.id !== id)
    } else {
      const idx = instances.value.findIndex((i) => i.id === id)
      if (idx >= 0 && status) {
        instances.value[idx].status = status
      }
    }
  })

  wsManager.on('connection', (data: any) => {
    if (data.connected) {
      fetchInstances()
    }
  })

  return {
    instances,
    loading,
    fetchInstances,
    createInstance,
    startInstance,
    stopInstance,
    restartInstance,
    deleteInstance,
  }
})