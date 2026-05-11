<script setup>
import { ref, computed, onMounted } from 'vue'
import GameCard from './components/GameCard.vue'
import ServerRow from './components/ServerRow.vue'
import InstallDialog from './components/InstallDialog.vue'
import ThemeToggle from './components/ThemeToggle.vue'

const tab = ref('catalog')
const games = ref([])
const servers = ref([])
const loading = ref(true)
const error = ref(null)
const query = ref('')
const installing = ref(null)

async function loadAll () {
  loading.value = true
  error.value = null
  try {
    const [g, s] = await Promise.all([
      fetch('/api/v1/games').then(r => r.json()),
      fetch('/api/v1/servers').then(r => r.json())
    ])
    games.value = g
    servers.value = s
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function createServer (payload) {
  const r = await fetch('/api/v1/servers', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  })
  if (!r.ok) throw new Error((await r.json()).error || 'create failed')
  installing.value = null
  tab.value = 'servers'
  await loadAll()
}

async function deleteServer (id) {
  await fetch(`/api/v1/servers/${id}`, { method: 'DELETE' })
  await loadAll()
}

const filteredGames = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return games.value
  return games.value.filter(g =>
    g.name.toLowerCase().includes(q) ||
    (g.summary || '').toLowerCase().includes(q)
  )
})

onMounted(loadAll)
</script>

<template>
  <div class="page">
    <header class="header">
      <div class="header-inner">
        <div class="brand" data-testid="brand">
          <div class="brand-icon">G</div>
          <span>Game Server</span>
        </div>
        <div class="spacer" />
        <nav class="tabs">
          <button
            class="tab"
            :class="{ active: tab === 'catalog' }"
            data-testid="tab-catalog"
            @click="tab = 'catalog'"
          >Catalog</button>
          <button
            class="tab"
            :class="{ active: tab === 'servers' }"
            data-testid="tab-servers"
            @click="tab = 'servers'"
          >My Servers <span v-if="servers.length" class="count">{{ servers.length }}</span></button>
        </nav>
        <ThemeToggle />
      </div>
    </header>

    <main>
      <div v-if="error" class="error" data-testid="error">{{ error }}</div>

      <section v-if="tab === 'catalog'" data-testid="catalog">
        <div class="toolbar">
          <input
            v-model="query"
            class="search"
            placeholder="Search games"
            data-testid="search"
          />
        </div>
        <div v-if="loading" class="empty">Loading…</div>
        <div v-else-if="filteredGames.length === 0" class="empty">
          <div class="empty-title">No games match</div>
          Try a different search.
        </div>
        <div v-else class="grid" data-testid="game-grid">
          <GameCard
            v-for="g in filteredGames"
            :key="g.id"
            :game="g"
            @install="installing = g"
          />
        </div>
      </section>

      <section v-else data-testid="servers">
        <div v-if="loading" class="empty">Loading…</div>
        <div v-else-if="servers.length === 0" class="empty" data-testid="servers-empty">
          <div class="empty-title">No servers yet</div>
          Install a game from the Catalog to get started.
        </div>
        <div v-else class="grid">
          <ServerRow
            v-for="s in servers"
            :key="s.id"
            :server="s"
            @delete="deleteServer(s.id)"
            @refresh="loadAll"
          />
        </div>
      </section>
    </main>

    <InstallDialog
      v-if="installing"
      :game="installing"
      @close="installing = null"
      @submit="createServer"
    />
  </div>
</template>

<style scoped>
.count {
  display: inline-block;
  margin-left: 4px;
  padding: 1px 7px;
  background: var(--accent);
  color: var(--accent-contrast);
  border-radius: 999px;
  font-size: 11px;
  font-weight: 600;
}
</style>
