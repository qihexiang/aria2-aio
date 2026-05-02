import { defineStore } from 'pinia'
import { ref } from 'vue'
import { wsManager } from '../api/ws'

export const useWsStore = defineStore('ws', () => {
  const connected = ref(false)
  const reconnecting = ref(false)
  const lastError = ref<string | null>(null)

  wsManager.on('connection', (data: any) => {
    connected.value = data.connected
    reconnecting.value = !data.connected
  })

  function connect() {
    wsManager.connect()
  }

  function disconnect() {
    wsManager.disconnect()
    connected.value = false
  }

  return {
    connected,
    reconnecting,
    lastError,
    connect,
    disconnect,
  }
})