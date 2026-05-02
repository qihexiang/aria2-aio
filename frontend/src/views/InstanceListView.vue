<template>
  <div>
    <div class="header-row">
      <h1>Instances</h1>
      <router-link to="/instances/create" class="btn btn-primary">New Instance</router-link>
    </div>

    <table>
      <thead>
        <tr>
          <th>Name</th>
          <th>Status</th>
          <th>Port</th>
          <th>PID</th>
          <th>Created</th>
          <th>Actions</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="inst in instanceStore.instances" :key="inst.id">
          <td>
            <router-link :to="`/instances/${inst.id}`" style="color: var(--primary); text-decoration: none; font-weight: 500">
              {{ inst.name }}
            </router-link>
          </td>
          <td><span :class="['badge', `badge-${inst.status}`]">{{ inst.status }}</span></td>
          <td>{{ inst.rpc_port }}</td>
          <td>{{ inst.pid || 'N/A' }}</td>
          <td>{{ formatDate(inst.created_at) }}</td>
          <td>
            <div class="actions-row">
              <button v-if="inst.status === 'stopped'" class="btn btn-success btn-sm" @click="instanceStore.startInstance(inst.id)">Start</button>
              <button v-if="inst.status === 'running'" class="btn btn-warning btn-sm" @click="instanceStore.stopInstance(inst.id)">Stop</button>
              <button v-if="inst.status === 'running'" class="btn btn-primary btn-sm" @click="instanceStore.restartInstance(inst.id)">Restart</button>
              <button class="btn btn-danger btn-sm" @click="confirmDelete(inst.id, inst.name)">Delete</button>
            </div>
          </td>
        </tr>
      </tbody>
    </table>

    <div v-if="instanceStore.instances.length === 0" style="text-align: center; padding: 40px; color: var(--text-muted)">
      No instances yet.
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useInstanceStore } from '../stores/instances'

const instanceStore = useInstanceStore()

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleDateString()
}

function confirmDelete(id: string, name: string) {
  if (confirm(`Delete instance "${name}"? This will remove all data including task history.`)) {
    instanceStore.deleteInstance(id)
  }
}

onMounted(() => {
  instanceStore.fetchInstances()
})
</script>