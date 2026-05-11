<script setup>
import { ref, onMounted, watch } from 'vue'

const KEY = 'syncloud-game-server-theme'
const theme = ref('light')

function apply (t) {
  document.documentElement.setAttribute('data-theme', t)
}

onMounted(() => {
  const stored = localStorage.getItem(KEY)
  if (stored === 'light' || stored === 'dark') {
    theme.value = stored
  } else if (window.matchMedia('(prefers-color-scheme: dark)').matches) {
    theme.value = 'dark'
  }
  apply(theme.value)
})

watch(theme, t => {
  apply(t)
  localStorage.setItem(KEY, t)
})

function toggle () {
  theme.value = theme.value === 'dark' ? 'light' : 'dark'
}
</script>

<template>
  <button class="theme-toggle" data-testid="theme-toggle" @click="toggle">
    {{ theme === 'dark' ? '☀' : '☾' }}
  </button>
</template>
