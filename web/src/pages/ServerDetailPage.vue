<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api'
import ConfirmDialog from '../components/ConfirmDialog.vue'

const props = defineProps({ id: { type: String, required: true } })
const router = useRouter()
const server = ref(null)
const logs = ref([])
const queryInfo = ref(null)
const error = ref(null)
const tab = ref('overview')
const confirmingDelete = ref(false)
let pollServer, pollLogs

async function load () {
  try {
    server.value = await api.server(props.id)
    error.value = null
  } catch (e) { error.value = e.message }
}

async function loadLogs () {
  try {
    const r = await api.logs(props.id)
    logs.value = r.lines || []
  } catch (_) { /* ignore */ }
}

async function action (name) {
  await api.action(props.id, name)
  await load()
}

async function runQuery () {
  queryInfo.value = null
  try { queryInfo.value = await api.query(props.id) }
  catch (e) { queryInfo.value = { error: e.message } }
}

async function confirmDelete () {
  confirmingDelete.value = false
  await api.deleteServer(props.id)
  router.push('/servers')
}

const status = computed(() => server.value?.status || 'unknown')
const installed = computed(() => server.value?.installDir && server.value.installDir.length > 0)

onMounted(() => {
  load()
  loadLogs()
  pollServer = setInterval(load, 3000)
  pollLogs = setInterval(loadLogs, 3000)
})
onUnmounted(() => { clearInterval(pollServer); clearInterval(pollLogs) })
</script>

<template>
  <button class="link-back" data-testid="back" @click="router.push('/servers')">← Servers</button>

  <div v-if="error" class="error">{{ error }}</div>

  <div v-if="server" class="detail-head">
    <div class="card-icon big">{{ server.name.charAt(0) }}</div>
    <div class="detail-meta">
      <h1 data-testid="detail-name">{{ server.name }}</h1>
      <p class="muted">
        {{ server.gameId.toUpperCase() }} ·
        <span :class="['badge', `status-${status}`]" data-testid="detail-status">{{ status }}</span>
        ·
        PORT {{ server.port }}
      </p>
    </div>
    <div class="detail-actions">
      <button v-if="!installed" class="btn" data-testid="action-install" @click="action('install')">Install</button>
      <button v-else-if="status === 'running'" class="btn ghost" data-testid="action-stop" @click="action('stop')">Stop</button>
      <button v-else class="btn" data-testid="action-start" @click="action('start')">Start</button>
      <button class="btn ghost" data-testid="action-delete" @click="confirmingDelete = true">Delete</button>
    </div>
  </div>

  <div v-if="server?.lastError" class="error" data-testid="detail-last-error">
    Last error: {{ server.lastError }}
  </div>

  <nav class="subtabs">
    <button class="subtab" :class="{ active: tab === 'overview' }" data-testid="subtab-overview" @click="tab = 'overview'">Overview</button>
    <button class="subtab" :class="{ active: tab === 'logs' }" data-testid="subtab-logs" @click="tab = 'logs'">Logs</button>
    <button class="subtab" :class="{ active: tab === 'query' }" data-testid="subtab-query" @click="tab = 'query'">Server query</button>
  </nav>

  <section v-if="tab === 'overview' && server" class="detail-card" data-testid="detail-overview">
    <dl>
      <dt>Install dir</dt><dd>{{ server.installDir || '—' }}</dd>
      <dt>Start command</dt><dd class="mono">{{ server.startCmd || '—' }}</dd>
      <dt>Status</dt><dd>{{ status }}</dd>
    </dl>
  </section>

  <section v-if="tab === 'logs'" class="detail-card log-viewer" data-testid="detail-logs">
    <div v-if="logs.length === 0" class="muted">No log lines yet. They appear here once the server has been started.</div>
    <pre v-else>{{ logs.join('') }}</pre>
  </section>

  <section v-if="tab === 'query'" class="detail-card" data-testid="detail-query">
    <p class="muted">Send an A2S_INFO query to the running server (only works for Source-engine games like CS2, TF2, GMod).</p>
    <button class="btn" data-testid="query-run" @click="runQuery">Run query</button>
    <pre v-if="queryInfo" class="mono">{{ JSON.stringify(queryInfo, null, 2) }}</pre>
  </section>

  <ConfirmDialog
    v-if="confirmingDelete && server"
    title="Delete server?"
    :message="`Remove '${server.name}' from the catalog. Installed game files under /data/games/servers/${server.name} will stay on disk — delete them manually if you no longer want them.`"
    confirm-label="Delete"
    danger
    @cancel="confirmingDelete = false"
    @confirm="confirmDelete"
  />
</template>

<style scoped>
.link-back {
  background: transparent;
  border: 0;
  color: var(--text-muted);
  margin-bottom: 16px;
  padding: 4px 0;
  font-size: 14px;
}
.link-back:hover { color: var(--text); }
.detail-head {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 16px;
}
.card-icon.big { width: 56px; height: 56px; font-size: 22px; }
.detail-meta { flex: 1; }
.detail-meta h1 { margin: 0; font-size: 22px; }
.muted { color: var(--text-muted); font-size: 14px; margin: 4px 0 0; }
.detail-actions { display: flex; gap: 8px; }
.subtabs {
  display: flex;
  gap: 4px;
  background: var(--bg);
  border-radius: var(--radius-sm);
  padding: 4px;
  margin-bottom: 16px;
  width: fit-content;
}
.subtab {
  border: 0;
  background: transparent;
  color: var(--text-muted);
  padding: 6px 14px;
  border-radius: 6px;
  font-weight: 500;
}
.subtab.active {
  background: var(--bg-elevated);
  color: var(--text);
}
.detail-card {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 20px;
  box-shadow: var(--shadow);
}
.detail-card dl { display: grid; grid-template-columns: max-content 1fr; gap: 8px 16px; margin: 0; }
.detail-card dt { color: var(--text-muted); font-size: 13px; }
.detail-card dd { margin: 0; word-break: break-all; }
.mono { font-family: ui-monospace, "SF Mono", Menlo, monospace; font-size: 12px; }
.log-viewer pre {
  max-height: 480px;
  overflow: auto;
  background: var(--bg);
  border-radius: var(--radius-sm);
  padding: 12px;
  font-size: 12px;
  margin: 0;
  white-space: pre-wrap;
}
.badge.status-running { background: rgba(22, 163, 74, 0.14); color: var(--success); }
.badge.status-stopped { background: rgba(100, 116, 139, 0.18); color: var(--text-muted); }
.badge.status-installing { background: rgba(217, 119, 6, 0.14); color: var(--warning); }
.badge.status-install-error { background: rgba(220, 38, 38, 0.14); color: var(--danger); }
</style>
