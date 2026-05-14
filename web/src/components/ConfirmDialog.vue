<script setup>
defineProps({
  title: { type: String, required: true },
  message: { type: String, required: true },
  confirmLabel: { type: String, default: 'Confirm' },
  cancelLabel: { type: String, default: 'Cancel' },
  danger: { type: Boolean, default: false }
})
defineEmits(['cancel', 'confirm'])
</script>

<template>
  <div class="modal-backdrop" data-testid="confirm-dialog" @click.self="$emit('cancel')">
    <div class="modal">
      <header class="modal-head">
        <h2>{{ title }}</h2>
        <button class="modal-close" data-testid="confirm-close" @click="$emit('cancel')">✕</button>
      </header>
      <div class="modal-body">
        <p class="message" data-testid="confirm-message">{{ message }}</p>
      </div>
      <div class="modal-foot">
        <button class="btn ghost" data-testid="confirm-cancel" @click="$emit('cancel')">{{ cancelLabel }}</button>
        <button
          class="btn"
          :class="{ danger }"
          data-testid="confirm-ok"
          @click="$emit('confirm')"
        >{{ confirmLabel }}</button>
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
  padding: 16px;
}
.modal {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  width: min(420px, 100%);
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
.modal-body { padding: 20px; }
.message { margin: 0; color: var(--text); font-size: 14px; line-height: 1.5; }
.modal-foot {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 12px 20px;
  border-top: 1px solid var(--border);
}
.btn.danger {
  background: var(--danger);
  border-color: var(--danger);
  color: #fff;
}
.btn.danger:hover { filter: brightness(0.95); }
</style>
