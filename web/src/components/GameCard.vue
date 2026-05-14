<script setup>
defineProps({ game: { type: Object, required: true } })
defineEmits(['install'])
</script>

<template>
  <article class="card" :data-testid="`game-${game.id}`">
    <div class="card-head">
      <div class="card-icon">{{ game.name.charAt(0) }}</div>
      <div>
        <h3 class="card-title">{{ game.name }}</h3>
        <span :class="['badge', game.source === 'steam' ? '' : 'egg']">{{ game.source }}</span>
        <span :class="['badge', 'tier', `tier-${game.tier || 'unknown'}`]">{{ game.tier || 'unknown' }}</span>
      </div>
    </div>
    <p class="card-summary">{{ game.summary }}</p>
    <div class="card-foot">
      <span class="port">{{ (game.protocols || []).join('/').toUpperCase() }} {{ game.defaultPort }}</span>
      <button class="btn" data-testid="install-btn" @click="$emit('install', game)">Install</button>
    </div>
  </article>
</template>

<style scoped>
.badge.tier {
  margin-left: 6px;
  text-transform: uppercase;
  font-size: 10px;
  letter-spacing: 0.05em;
}
.tier-verified { background: rgba(22, 163, 74, 0.14); color: var(--success); }
.tier-compatible { background: rgba(217, 119, 6, 0.14); color: var(--warning); }
.tier-experimental { background: rgba(220, 38, 38, 0.14); color: var(--danger); }
.tier-unknown { background: var(--accent-soft); color: var(--text-muted); }
</style>
