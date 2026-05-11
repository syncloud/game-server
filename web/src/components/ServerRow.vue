<script setup>
defineProps({ server: { type: Object, required: true } })
defineEmits(['delete'])
</script>

<template>
  <div class="card" :data-testid="`server-${server.id}`">
    <div class="card-head">
      <div class="card-icon">{{ server.name.charAt(0) }}</div>
      <div>
        <h3 class="card-title">{{ server.name }}</h3>
        <span :class="['badge', `status-${server.status}`]">{{ server.status }}</span>
      </div>
    </div>
    <div class="card-foot">
      <span class="port">{{ server.gameId.toUpperCase() }} · PORT {{ server.port }}</span>
      <div>
        <button class="btn ghost" data-testid="server-delete" @click="$emit('delete')">Delete</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.badge.status-running { background: rgba(22, 163, 74, 0.14); color: var(--success); }
.badge.status-stopped { background: rgba(100, 116, 139, 0.18); color: var(--text-muted); }
.badge.status-installing { background: rgba(217, 119, 6, 0.14); color: var(--warning); }
.badge.status-error { background: rgba(220, 38, 38, 0.14); color: var(--danger); }
</style>
