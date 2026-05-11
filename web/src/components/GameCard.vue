<script setup>
defineProps({ game: { type: Object, required: true } })
</script>

<template>
  <article class="card" :data-testid="`game-${game.id}`">
    <div class="card-head">
      <div class="card-icon">{{ game.name.charAt(0) }}</div>
      <div>
        <h3 class="card-title">{{ game.name }}</h3>
        <span :class="['badge', game.source === 'egg' ? 'egg' : '']">{{ game.source }}</span>
      </div>
    </div>
    <p class="card-summary">{{ game.summary }}</p>
    <div class="card-foot">
      <span class="port">{{ game.protocols.join('/').toUpperCase() }} {{ game.defaultPort }}</span>
      <button class="btn" data-testid="install-btn" disabled>Install</button>
    </div>
  </article>
</template>
