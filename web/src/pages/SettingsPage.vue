<script setup>
import { ref, onMounted } from 'vue'
import { api } from '../api'

const sources = ref({})
const error = ref(null)
const steam = ref({ username: '', password: '', guardCode: '', state: 'idle', message: '', linked: false, linkedUsername: '' })

async function load () {
  try { sources.value = await api.catalogSources() }
  catch (e) { error.value = e.message }
  try {
    const s = await fetch('/api/v1/steam/status').then(r => r.json())
    steam.value.linked = !!s.linked
    steam.value.linkedUsername = s.username || ''
  } catch (_) { /* */ }
}

async function steamLogin () {
  steam.value.state = 'submitting'
  steam.value.message = ''
  try {
    const r = await fetch('/api/v1/steam/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        username: steam.value.username,
        password: steam.value.password,
        guardCode: steam.value.guardCode
      })
    })
    if (!r.ok) {
      const b = await r.json().catch(() => ({}))
      steam.value.state = 'error'
      steam.value.message = b.error || `http ${r.status}`
      return
    }
    const b = await r.json()
    if (b.needsGuard) {
      steam.value.state = 'needs-guard'
      steam.value.message = b.prompt || 'Steam Guard code required — check your email or mobile app'
      return
    }
    steam.value.state = 'ok'
    steam.value.message = 'Linked as ' + (b.username || steam.value.username)
    steam.value.password = ''
    steam.value.guardCode = ''
  } catch (e) {
    steam.value.state = 'error'
    steam.value.message = e.message
  }
}

onMounted(load)
</script>

<template>
  <h1 class="page-title">Settings</h1>

  <section class="detail-card" data-testid="settings-steam">
    <h2>Steam account</h2>
    <p class="muted">
      Needed only for games that require a paid Steam account (Rust, Squad, Arma 3, …).
      Anonymous-friendly games (CS2, TF2, Valheim, Project Zomboid, ARK) install without this.
    </p>
    <p class="muted">
      We don't store your Steam password — after a successful login we keep only the
      session token (sentry file). You may need to enter a Steam Guard code from your
      email or mobile app on first login.
    </p>
    <p v-if="steam.linked" class="form-msg ok" data-testid="steam-linked">
      Connected as <strong>{{ steam.linkedUsername }}</strong>. Re-submit below to change.
    </p>
    <form @submit.prevent="steamLogin">
      <label>
        Username
        <input v-model="steam.username" data-testid="steam-username" autocomplete="username" />
      </label>
      <label>
        Password
        <input v-model="steam.password" type="password" data-testid="steam-password" autocomplete="current-password" />
      </label>
      <label v-if="steam.state === 'needs-guard' || steam.guardCode">
        Steam Guard code
        <input v-model="steam.guardCode" data-testid="steam-guard" />
      </label>
      <button class="btn" :disabled="steam.state === 'submitting'" data-testid="steam-submit" type="submit">
        {{ steam.state === 'submitting' ? 'Logging in…' : 'Connect Steam' }}
      </button>
      <p v-if="steam.message" :class="['form-msg', steam.state]" data-testid="steam-message">{{ steam.message }}</p>
    </form>
  </section>

  <section class="detail-card" data-testid="settings-sources">
    <h2>Catalog sources</h2>
    <p class="muted">Pinned upstream versions vendored at this snap's build time. Bump to refresh the catalog.</p>
    <dl v-if="Object.keys(sources).length">
      <template v-for="(ver, name) in sources" :key="name">
        <dt>{{ name }}</dt>
        <dd class="mono">{{ ver }}</dd>
      </template>
    </dl>
    <p v-else class="muted">No source info available.</p>
  </section>

  <section class="detail-card" data-testid="settings-about">
    <h2>About</h2>
    <p class="muted">
      <a href="https://github.com/syncloud/game-server" target="_blank">syncloud/game-server</a>
      — Syncloud panel for SteamCMD and Pelican-egg game servers. WIP, see issue
      <a href="https://github.com/syncloud/platform/issues/35" target="_blank">platform#35</a>.
    </p>
  </section>
</template>

<style scoped>
.page-title { font-size: 24px; margin: 0 0 16px; }
.detail-card { margin-bottom: 16px; }
.detail-card h2 { margin: 0 0 12px; font-size: 18px; }
.detail-card form { display: flex; flex-direction: column; gap: 12px; max-width: 360px; }
.detail-card form label { display: flex; flex-direction: column; gap: 4px; font-size: 13px; color: var(--text-muted); font-weight: 500; }
.detail-card form input {
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 8px 12px;
  color: var(--text);
  font-size: 14px;
}
.detail-card form input:focus {
  outline: none; border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-soft);
}
.detail-card form .btn { align-self: flex-start; }
.form-msg.ok { color: var(--success); }
.form-msg.error { color: var(--danger); }
.form-msg.needs-guard { color: var(--warning); }
.detail-card dl { display: grid; grid-template-columns: max-content 1fr; gap: 8px 16px; margin: 0; }
.detail-card dt { color: var(--text-muted); font-size: 13px; }
.detail-card dd { margin: 0; word-break: break-all; }
.mono { font-family: ui-monospace, "SF Mono", Menlo, monospace; font-size: 12px; }
</style>
