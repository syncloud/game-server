<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '../api'
import GameCard from '../components/GameCard.vue'
import InstallDialog from '../components/InstallDialog.vue'

const router = useRouter()
const games = ref([])
const loading = ref(true)
const error = ref(null)
const query = ref('')
const tierFilter = ref('all')
const sourceFilter = ref('all')
const installing = ref(null)

const TIERS = [
  { id: 'all', label: 'All' },
  { id: 'supported', label: 'Supported' },
  { id: 'compatible', label: 'Compatible' },
  { id: 'experimental', label: 'Experimental' }
]

async function load () {
  loading.value = true; error.value = null
  try { games.value = await api.games() }
  catch (e) { error.value = e.message }
  finally { loading.value = false }
}

const sourceOptions = computed(() => {
  const set = new Set(games.value.map(g => g.source))
  return ['all', ...Array.from(set).sort()]
})

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  return games.value.filter(g => {
    if (tierFilter.value !== 'all' && g.tier !== tierFilter.value) return false
    if (sourceFilter.value !== 'all' && g.source !== sourceFilter.value) return false
    if (!q) return true
    return g.name.toLowerCase().includes(q) || (g.summary || '').toLowerCase().includes(q)
  })
})

const counts = computed(() => {
  const c = { supported: 0, compatible: 0, experimental: 0 }
  for (const g of games.value) if (c[g.tier] !== undefined) c[g.tier]++
  return c
})

async function createServer (payload) {
  try {
    const created = await api.createServer(payload)
    installing.value = null
    await router.push(`/servers/${created.id}`)
  } catch (e) {
    error.value = e.message
    throw e
  }
}

onMounted(load)
</script>

<template>
  <div v-if="error" class="error" data-testid="error">{{ error }}</div>

  <div class="toolbar">
    <input
      v-model="query"
      class="search"
      placeholder="Search games"
      data-testid="search"
    />
    <select v-model="sourceFilter" class="select" data-testid="source-filter">
      <option v-for="s in sourceOptions" :key="s" :value="s">{{ s === 'all' ? 'All sources' : s }}</option>
    </select>
  </div>

  <div class="tier-pills">
    <button
      v-for="t in TIERS"
      :key="t.id"
      class="pill"
      :class="{ active: tierFilter === t.id }"
      :data-testid="`tier-${t.id}`"
      @click="tierFilter = t.id"
    >
      {{ t.label }}
      <span v-if="t.id !== 'all'" class="pill-count">{{ counts[t.id] }}</span>
    </button>
  </div>

  <div v-if="loading" class="empty">Loading {{ games.length || '' }} games…</div>
  <div v-else-if="filtered.length === 0" class="empty">
    <div class="empty-title">No games match</div>
    Adjust filters or try a different search.
  </div>
  <div v-else class="grid" data-testid="game-grid">
    <GameCard
      v-for="g in filtered"
      :key="g.id"
      :game="g"
      @install="installing = g"
    />
  </div>

  <InstallDialog
    v-if="installing"
    :game="installing"
    @close="installing = null"
    @submit="createServer"
  />
</template>

<style scoped>
.select {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 10px 14px;
  color: var(--text);
  font-size: 14px;
}
.tier-pills {
  display: flex;
  gap: 8px;
  margin-bottom: 24px;
  flex-wrap: wrap;
}
.pill {
  border: 1px solid var(--border);
  background: var(--bg-elevated);
  color: var(--text-muted);
  padding: 6px 14px;
  border-radius: 999px;
  font-size: 13px;
  font-weight: 500;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.pill.active {
  background: var(--accent);
  color: var(--accent-contrast);
  border-color: var(--accent);
}
.pill-count {
  background: rgba(255,255,255,0.18);
  border-radius: 999px;
  padding: 0 6px;
  font-size: 11px;
}
.pill:not(.active) .pill-count {
  background: var(--bg);
  color: var(--text-muted);
}
</style>
