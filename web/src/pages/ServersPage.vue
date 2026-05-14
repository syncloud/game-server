<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api'
import ServerRow from '../components/ServerRow.vue'
import ConfirmDialog from '../components/ConfirmDialog.vue'

const router = useRouter()
const servers = ref([])
const loading = ref(true)
const error = ref(null)
const pendingDelete = ref(null)
let poll

async function load () {
  try { servers.value = await api.servers(); error.value = null }
  catch (e) { error.value = e.message }
  finally { loading.value = false }
}

async function action (id, name) {
  await api.action(id, name)
  await load()
}

async function confirmDelete () {
  const s = pendingDelete.value
  pendingDelete.value = null
  if (!s) return
  await api.deleteServer(s.id)
  await load()
}

onMounted(() => {
  load()
  poll = setInterval(load, 5000)
})
onUnmounted(() => clearInterval(poll))
</script>

<template>
  <div v-if="error" class="error" data-testid="error">{{ error }}</div>

  <div v-if="loading" class="empty">Loading…</div>
  <div v-else-if="servers.length === 0" class="empty" data-testid="servers-empty">
    <div class="empty-title">No servers yet</div>
    <p>Install a game from the <router-link to="/catalog">Catalog</router-link> to get started.</p>
  </div>
  <div v-else class="grid">
    <ServerRow
      v-for="s in servers"
      :key="s.id"
      :server="s"
      @open="router.push({ name: 'server-detail', params: { id: s.id } })"
      @start="action(s.id, 'start')"
      @stop="action(s.id, 'stop')"
      @install="action(s.id, 'install')"
      @delete="pendingDelete = s"
    />
  </div>

  <ConfirmDialog
    v-if="pendingDelete"
    title="Delete server?"
    :message="`Remove '${pendingDelete.name}' from the catalog. Installed game files under /data/games/servers/${pendingDelete.name} will stay on disk — delete them manually if you no longer want them.`"
    confirm-label="Delete"
    danger
    @cancel="pendingDelete = null"
    @confirm="confirmDelete"
  />
</template>
