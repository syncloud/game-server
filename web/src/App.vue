<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import ThemeToggle from './components/ThemeToggle.vue'
import catalogIcon from './assets/icons/catalog.svg'
import serversIcon from './assets/icons/servers.svg'
import settingsIcon from './assets/icons/settings.svg'
import syncloudLogo from './assets/syncloud-logo.svg'

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
          <img class="brand-logo" :src="syncloudLogo" alt="Syncloud" />
          <span class="brand-name">Syncloud Game Hub</span>
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
        <span class="bottom-icon" :style="{ '--icon-url': `url(${catalogIcon})` }" aria-hidden="true" />
        <span class="bottom-label">Catalog</span>
      </router-link>
      <router-link to="/servers" class="bottom-tab" :class="{ active: isActive('servers').value }" data-testid="bottom-servers">
        <span class="bottom-icon" :style="{ '--icon-url': `url(${serversIcon})` }" aria-hidden="true" />
        <span class="bottom-label">Servers</span>
      </router-link>
      <router-link to="/settings" class="bottom-tab" :class="{ active: isActive('settings').value }" data-testid="bottom-settings">
        <span class="bottom-icon" :style="{ '--icon-url': `url(${settingsIcon})` }" aria-hidden="true" />
        <span class="bottom-label">Settings</span>
      </router-link>
    </nav>
  </div>
</template>
