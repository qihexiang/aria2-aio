<template>
  <div>
    <div class="header-row">
      <h1>Create Instance</h1>
      <router-link to="/instances" class="btn btn-sm" style="background: var(--bg)">Cancel</router-link>
    </div>

    <div class="card">
      <div class="form-group">
        <label>Name</label>
        <input v-model="name" placeholder="Instance name" required />
      </div>

      <div class="form-group">
        <label>Download Directory (optional)</label>
        <input v-model="dir" placeholder="Leave empty for default path" />
      </div>

      <h3 style="margin-top: 20px; margin-bottom: 12px">Aria2 Options</h3>
      <div v-for="(opt, index) in optionList" :key="index" class="form-group" style="display: flex; gap: 8px">
        <input v-model="opt.key" style="flex: 1" placeholder="Option key" />
        <input v-model="opt.value" style="flex: 1" placeholder="Option value" />
        <button class="btn btn-danger btn-sm" @click="optionList.splice(index, 1)">Remove</button>
      </div>
      <button class="btn btn-sm" style="background: var(--bg); margin-top: 8px" @click="optionList.push({ key: '', value: '' })">Add Option</button>

      <div style="margin-top: 24px">
        <button class="btn btn-primary" @click="submit">Create Instance</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useInstanceStore } from '../stores/instances'

const router = useRouter()
const instanceStore = useInstanceStore()

const name = ref('')
const dir = ref('')
const optionList = ref([
  { key: 'max-concurrent-downloads', value: '5' },
  { key: 'split', value: '5' },
  { key: 'continue', value: 'true' },
])

async function submit() {
  if (!name.value) {
    alert('Name is required')
    return
  }

  const cleanOptions: Record<string, string> = {}
  for (const opt of optionList.value) {
    if (opt.key.trim()) {
      cleanOptions[opt.key.trim()] = opt.value
    }
  }

  const inst = await instanceStore.createInstance({
    name: name.value,
    dir: dir.value || undefined,
    options: cleanOptions,
  })

  router.push(`/instances/${inst.id}`)
}
</script>