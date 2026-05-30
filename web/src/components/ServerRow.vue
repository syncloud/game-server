<script setup>
import { ref } from 'vue'

const props = defineProps({ server: { type: Object, required: true } })
const emit = defineEmits(['open', 'start', 'stop', 'install', 'delete'])

const copied = ref(false)

async function copyConnect () {
  const text = `${props.server.localIp || 'server-ip'}:${props.server.port}`
  try {
    await navigator.clipboard.writeText(text)
    copied.value = true
    setTimeout(() => { copied.value = false }, 1200)
  } catch (_) {  }
}
</script>

<template>
  <div class="card" :data-testid="`server-${server.id}`" @click="emit('open')">
    <div class="card-head">
      <h3 class="card-title">
        {{ server.gameName || server.gameId }}
        <span :class="['badge', `status-${server.status}`]">{{ server.status }}</span>
      </h3>
    </div>
    <div class="card-foot">
      <span class="connect">
        <span class="port" data-testid="server-connect">{{ server.localIp || 'server-ip' }}:{{ server.port }}</span>
        <button
          class="copy-btn"
          :class="{ copied }"
          data-testid="server-copy"
          :title="copied ? 'Copied' : 'Copy address'"
          @click.stop="copyConnect"
        >
          <svg v-if="copied" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>
          <svg v-else width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
        </button>
      </span>
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
.card-title { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.connect { display: inline-flex; align-items: center; gap: 6px; }
.copy-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 4px;
  border: 0;
  background: transparent;
  color: var(--text-muted);
  border-radius: 6px;
  cursor: pointer;
}
.copy-btn:hover { color: var(--text); background: var(--bg); }
.copy-btn.copied { color: var(--success); }
.actions { display: flex; gap: 6px; }

@media (max-width: 640px) {
  .card-foot { flex-wrap: wrap; }
  .actions { width: 100%; margin-top: 8px; }
  .actions .btn { flex: 1; }
}
</style>
