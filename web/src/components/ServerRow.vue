<script setup>
const props = defineProps({ server: { type: Object, required: true } })
const emit = defineEmits(['open', 'start', 'stop', 'install', 'delete'])
</script>

<template>
  <div class="card" :data-testid="`server-${server.id}`" @click="emit('open')">
    <div class="card-head">
      <div class="card-icon">{{ server.name.charAt(0) }}</div>
      <div>
        <h3 class="card-title">{{ server.name }}</h3>
        <span :class="['badge', `status-${server.status}`]">{{ server.status }}</span>
      </div>
    </div>
    <div class="card-foot">
      <span class="port">{{ server.gameId.toUpperCase() }} · PORT {{ server.port }}</span>
      <div class="actions" @click.stop>
        <button
          v-if="!server.installDir"
          class="btn"
          data-testid="server-install"
          @click="emit('install')"
        >Install</button>
        <button
          v-else-if="server.status === 'running'"
          class="btn ghost"
          data-testid="server-stop"
          @click="emit('stop')"
        >Stop</button>
        <button
          v-else
          class="btn"
          data-testid="server-start"
          @click="emit('start')"
        >Start</button>
        <button class="btn ghost" data-testid="server-delete" @click="emit('delete')">Delete</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.card { cursor: pointer; }
.badge.status-running { background: rgba(22, 163, 74, 0.14); color: var(--success); }
.badge.status-stopped { background: rgba(100, 116, 139, 0.18); color: var(--text-muted); }
.badge.status-installing { background: rgba(217, 119, 6, 0.14); color: var(--warning); }
.badge.status-install-error,
.badge.status-error { background: rgba(220, 38, 38, 0.14); color: var(--danger); }
.actions { display: flex; gap: 6px; }
</style>
