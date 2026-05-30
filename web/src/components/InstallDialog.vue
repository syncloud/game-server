<script setup>
import { ref, computed } from 'vue'

const props = defineProps({ game: { type: Object, required: true } })
const emit = defineEmits(['close', 'submit'])

const port = ref(props.game.defaultPort)
const error = ref(null)

const valid = computed(() => port.value > 0)

async function submit () {
  error.value = null
  try {
    await emit('submit', {
      gameId: props.game.id,
      port: Number(port.value)
    })
  } catch (e) {
    error.value = e.message
  }
}
</script>

<template>
  <div class="modal-backdrop" data-testid="install-dialog" @click.self="emit('close')">
    <div class="modal">
      <header class="modal-head">
        <h2>Install {{ game.name }}</h2>
        <button class="modal-close" data-testid="dialog-close" @click="emit('close')">✕</button>
      </header>
      <div class="modal-body">
        <label>
          Port
          <input v-model.number="port" type="number" data-testid="dialog-port" />
        </label>
        <p v-if="game.installRecipe?.method === 'steam'" class="hint">
          Steam dedicated server (app {{ game.installRecipe.steamAppId }}). Anonymous login will be used.
        </p>
        <p v-else-if="game.installRecipe?.method === 'downloadExtract'" class="hint">
          Direct download: {{ game.installRecipe.url }}
        </p>
        <p v-else class="hint">
          No install recipe for this game.
        </p>
        <p v-if="error" class="error">{{ error }}</p>
      </div>
      <div class="modal-foot">
        <button class="btn ghost" @click="emit('close')">Cancel</button>
        <button class="btn" :disabled="!valid" data-testid="dialog-submit" @click="submit">Install</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(15, 23, 42, 0.55);
  display: grid;
  place-items: center;
  z-index: 100;
}

.modal {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  width: min(440px, 92vw);
  box-shadow: var(--shadow);
  overflow: hidden;
}

.modal-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border);
}

.modal-head h2 { margin: 0; font-size: 18px; font-weight: 600; }

.modal-close {
  border: 0;
  background: transparent;
  color: var(--text-muted);
  font-size: 18px;
  padding: 4px 8px;
  border-radius: 6px;
}

.modal-close:hover { background: var(--bg); color: var(--text); }

.modal-body {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.modal-body label {
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 13px;
  color: var(--text-muted);
  font-weight: 500;
}

.modal-body input {
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 8px 12px;
  color: var(--text);
  font-size: 14px;
}

.modal-body input:focus {
  outline: none;
  border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-soft);
}

.hint {
  font-size: 12px;
  color: var(--text-muted);
  background: var(--bg);
  border-radius: 8px;
  padding: 8px 10px;
  margin: 0;
}

.modal-foot {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 12px 20px;
  border-top: 1px solid var(--border);
}
</style>
