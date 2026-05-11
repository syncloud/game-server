<script setup>
defineProps({ server: { type: Object, required: true } })
</script>

<template>
  <div class="card" :data-testid="`server-${server.id}`">
    <div class="card-head">
      <div class="card-icon">{{ server.name.charAt(0) }}</div>
      <div>
        <h3 class="card-title">{{ server.name }}</h3>
        <span class="badge">{{ server.status }}</span>
      </div>
    </div>
    <div class="card-foot">
      <span class="port">PORT {{ server.port }}</span>
      <button class="btn ghost">Manage</button>
    </div>
  </div>
</template>
