async function jsonOrThrow (res) {
  if (!res.ok) {
    let msg = `http ${res.status}`
    try { msg = (await res.json()).error || msg } catch (_) {  }
    throw new Error(msg)
  }
  if (res.status === 204) return null
  return res.json()
}

export const api = {
  health: () => fetch('/api/v1/health').then(jsonOrThrow),
  games: () => fetch('/api/v1/games').then(jsonOrThrow),
  catalogSources: () => fetch('/api/v1/catalog/sources').then(jsonOrThrow),
  servers: () => fetch('/api/v1/servers').then(jsonOrThrow),
  server: (id) => fetch(`/api/v1/servers/${id}`).then(jsonOrThrow),
  createServer: (body) => fetch('/api/v1/servers', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body)
  }).then(jsonOrThrow),
  deleteServer: (id) => fetch(`/api/v1/servers/${id}`, { method: 'DELETE' }).then(jsonOrThrow),
  action: (id, name) => fetch(`/api/v1/servers/${id}/${name}`, { method: 'POST' }).then(jsonOrThrow),
  logs: (id) => fetch(`/api/v1/servers/${id}/logs`).then(jsonOrThrow),
  query: (id) => fetch(`/api/v1/servers/${id}/query`).then(jsonOrThrow)
}
