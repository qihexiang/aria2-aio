<template>
  <nav>
    <router-link to="/" class="nav-brand">aria2-aio</router-link>
    <router-link to="/" :class="{ active: $route.path === '/' }">Dashboard</router-link>
    <router-link to="/instances" :class="{ active: $route.path.startsWith('/instances') }">Instances</router-link>
    <div class="ws-status" style="margin-left: auto">
      <div :class="['ws-dot', wsStore.connected ? 'connected' : 'disconnected']"></div>
      <span>{{ wsStore.connected ? 'Connected' : 'Disconnected' }}</span>
    </div>
  </nav>
  <div class="container">
    <router-view />
  </div>
</template>

<script setup lang="ts">
import { useWsStore } from './stores/ws'

const wsStore = useWsStore()
</script>