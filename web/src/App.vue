<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import ThemeToggle from './components/ThemeToggle.vue'

const route = useRoute()
const isActive = (name) => computed(() => route.name === name || (name === 'servers' && route.name === 'server-detail'))
</script>

<template>
  <div class="page">
    <header class="header">
      <div class="header-inner">
        <router-link to="/catalog" class="brand" data-testid="brand">
          <div class="brand-icon">G</div>
          <span>Game Server</span>
        </router-link>
        <div class="spacer" />
        <nav class="tabs">
          <router-link to="/catalog" class="tab" data-testid="tab-catalog" :class="{ active: isActive('catalog').value }">Catalog</router-link>
          <router-link to="/servers" class="tab" data-testid="tab-servers" :class="{ active: isActive('servers').value }">My Servers</router-link>
          <router-link to="/settings" class="tab" data-testid="tab-settings" :class="{ active: isActive('settings').value }">Settings</router-link>
        </nav>
        <ThemeToggle />
      </div>
    </header>
    <main>
      <router-view />
    </main>
  </div>
</template>
