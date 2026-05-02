<template>
  <div>
    <div class="header-row">
      <h1>Dashboard</h1>
      <router-link to="/instances/create" class="btn btn-primary">New Instance</router-link>
    </div>

    <div class="stats-row">
      <div class="stat-card">
        <h3>Global Download Speed</h3>
        <div class="value">{{ formatSpeed(globalStats.total_download_speed) }}</div>
      </div>
      <div class="stat-card">
        <h3>Global Upload Speed</h3>
        <div class="value">{{ formatSpeed(globalStats.total_upload_speed) }}</div>
      </div>
      <div class="stat-card">
        <h3>Active Downloads</h3>
        <div class="value">{{ globalStats.num_active_downloads }}</div>
      </div>
      <div class="stat-card">
        <h3>Running Instances</h3>
        <div class="value">{{ globalStats.num_running_instances }}</div>
      </div>
    </div>

    <div class="grid">
      <div v-for="inst in instanceStore.instances" :key="inst.id" class="card" @click="$router.push(`/instances/${inst.id}`)" style="cursor: pointer">
        <h3>{{ inst.name }}</h3>
        <span :class="['badge', `badge-${inst.status}`]">{{ inst.status }}</span>
        <div style="margin-top: 8px; font-size: 13px; color: var(--text-muted)">
          Port: {{ inst.rpc_port }} | PID: {{ inst.pid || 'N/A' }}
        </div>
      </div>
    </div>

    <div v-if="instanceStore.instances.length === 0" style="text-align: center; padding: 40px; color: var(--text-muted)">
      No instances yet. Create one to get started.
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useInstanceStore } from '../stores/instances'
import { getGlobalStats } from '../api/client'
import type { GlobalStats } from '../types'

const instanceStore = useInstanceStore()
const globalStats = ref<GlobalStats>({
  total_download_speed: 0,
  total_upload_speed: 0,
  num_active_downloads: 0,
  num_running_instances: 0,
})

function formatSpeed(bytes: number): string {
  if (bytes === 0) return '0 B/s'
  const units = ['B/s', 'KB/s', 'MB/s', 'GB/s']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`
}

onMounted(async () => {
  await instanceStore.fetchInstances()
  try {
    globalStats.value = await getGlobalStats()
  } catch {}
})
</script>