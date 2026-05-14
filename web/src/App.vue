<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import ThemeToggle from './components/ThemeToggle.vue'

const route = useRoute()
const user = ref(null)
const isActive = (name) => computed(() => route.name === name || (name === 'servers' && route.name === 'server-detail'))

onMounted(async () => {
  try {
    const r = await fetch('/api/v1/me')
    if (r.status === 401) {
      window.location.assign('/auth/login')
      return
    }
    if (r.ok) user.value = await r.json()
  } catch (_) { /* ignore */ }
})

function logout () { window.location.assign('/auth/logout') }
</script>

<template>
  <div class="page">
    <header class="header">
      <div class="header-inner">
        <router-link to="/catalog" class="brand" data-testid="brand">
          <div class="brand-icon">G</div>
          <span class="brand-name">Game Server</span>
        </router-link>
        <div class="spacer" />
        <nav class="tabs desktop-only">
          <router-link to="/catalog" class="tab" data-testid="tab-catalog" :class="{ active: isActive('catalog').value }">Catalog</router-link>
          <router-link to="/servers" class="tab" data-testid="tab-servers" :class="{ active: isActive('servers').value }">My Servers</router-link>
          <router-link to="/settings" class="tab" data-testid="tab-settings" :class="{ active: isActive('settings').value }">Settings</router-link>
        </nav>
        <div v-if="user" class="user-chip desktop-only" data-testid="user-chip" :title="user.email || ''">
          {{ user.name || user.sub }}
          <button class="logout-btn" data-testid="logout" @click="logout">Logout</button>
        </div>
        <ThemeToggle class="desktop-only" />
      </div>
    </header>
    <main>
      <router-view />
    </main>
    <nav class="bottom-bar mobile-only" data-testid="bottom-nav">
      <router-link to="/catalog" class="bottom-tab" :class="{ active: isActive('catalog').value }" data-testid="bottom-catalog">
        <svg class="bottom-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <rect x="3" y="6" width="18" height="12" rx="3" />
          <path d="M7 12h3M8.5 10.5v3" />
          <circle cx="15" cy="11" r="1" />
          <circle cx="17" cy="13" r="1" />
        </svg>
        <span class="bottom-label">Catalog</span>
      </router-link>
      <router-link to="/servers" class="bottom-tab" :class="{ active: isActive('servers').value }" data-testid="bottom-servers">
        <svg class="bottom-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <rect x="3" y="4" width="18" height="6" rx="2" />
          <rect x="3" y="14" width="18" height="6" rx="2" />
          <circle cx="7" cy="7" r="0.8" fill="currentColor" />
          <circle cx="7" cy="17" r="0.8" fill="currentColor" />
        </svg>
        <span class="bottom-label">Servers</span>
      </router-link>
      <router-link to="/settings" class="bottom-tab" :class="{ active: isActive('settings').value }" data-testid="bottom-settings">
        <svg class="bottom-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          <circle cx="12" cy="12" r="3" />
          <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 1 1-4 0v-.09a1.65 1.65 0 0 0-1-1.51 1.65 1.65 0 0 0-1.82.33l-.06.06A2 2 0 1 1 4.21 16.96l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 1 1 0-4h.09a1.65 1.65 0 0 0 1.51-1 1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33h0a1.65 1.65 0 0 0 1-1.51V3a2 2 0 1 1 4 0v.09a1.65 1.65 0 0 0 1 1.51h0a1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82v0a1.65 1.65 0 0 0 1.51 1H21a2 2 0 1 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z" />
        </svg>
        <span class="bottom-label">Settings</span>
      </router-link>
    </nav>
  </div>
</template>
