<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import ThemeToggle from './components/ThemeToggle.vue'

const route = useRoute()
const menuOpen = ref(false)
const user = ref(null)
const isActive = (name) => computed(() => route.name === name || (name === 'servers' && route.name === 'server-detail'))
function go () { menuOpen.value = false }

// On first load, ask backend who we are. 401 means no session — start the
// OIDC code flow by redirecting to /auth/login. Backend will bounce to
// Authelia and back via /auth/callback, then back to /.
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
        <router-link to="/catalog" class="brand" data-testid="brand" @click="go">
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
        <button class="hamburger mobile-only" :aria-expanded="menuOpen" aria-label="Menu" data-testid="hamburger" @click="menuOpen = !menuOpen">
          <span /><span /><span />
        </button>
      </div>
      <transition name="slide">
        <nav v-if="menuOpen" class="mobile-menu mobile-only">
          <router-link to="/catalog" class="mobile-link" data-testid="mobile-tab-catalog" :class="{ active: isActive('catalog').value }" @click="go">Catalog</router-link>
          <router-link to="/servers" class="mobile-link" data-testid="mobile-tab-servers" :class="{ active: isActive('servers').value }" @click="go">My Servers</router-link>
          <router-link to="/settings" class="mobile-link" data-testid="mobile-tab-settings" :class="{ active: isActive('settings').value }" @click="go">Settings</router-link>
          <div v-if="user" class="mobile-link-row" data-testid="mobile-user">
            <span class="mobile-user-name">{{ user.name || user.sub }}</span>
            <button class="logout-btn" @click="logout">Logout</button>
          </div>
          <div class="mobile-link-row">
            <ThemeToggle />
          </div>
        </nav>
      </transition>
    </header>
    <main>
      <router-view />
    </main>
    <nav class="bottom-bar mobile-only">
      <router-link to="/catalog" class="bottom-tab" :class="{ active: isActive('catalog').value }" data-testid="bottom-catalog">
        <span class="bottom-icon">🎮</span>
        <span class="bottom-label">Catalog</span>
      </router-link>
      <router-link to="/servers" class="bottom-tab" :class="{ active: isActive('servers').value }" data-testid="bottom-servers">
        <span class="bottom-icon">⚙</span>
        <span class="bottom-label">Servers</span>
      </router-link>
      <router-link to="/settings" class="bottom-tab" :class="{ active: isActive('settings').value }" data-testid="bottom-settings">
        <span class="bottom-icon">☰</span>
        <span class="bottom-label">Settings</span>
      </router-link>
    </nav>
  </div>
</template>
