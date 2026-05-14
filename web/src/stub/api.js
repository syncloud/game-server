import { createServer, Response } from 'miragejs'

const games = [
  {
    id: 'teeworlds',
    name: 'Teeworlds',
    source: 'pelican',
    upstreamRef: 'game_eggs/teeworlds/egg-teeworlds.json',
    summary: 'Fast-paced 2D online shooter. Smallest dedicated server in the catalog (~10 MB).',
    defaultPort: 8303,
    protocols: ['udp'],
    tier: 'compatible'
  },
  {
    id: 'hlds-cs',
    name: 'Counter-Strike 1.6',
    source: 'steam',
    upstreamRef: 'curated/hlds-cs',
    summary: 'Classic GoldSrc CS 1.6 dedicated server (HLDS). Bundled mod = cstrike.',
    defaultPort: 27015,
    protocols: ['udp'],
    tier: 'verified',
    steamAppId: 90
  },
  {
    id: 'cs2',
    name: 'Counter-Strike 2',
    source: 'steam',
    upstreamRef: 'curated/cs2',
    summary: 'Source 2 successor to CS:GO. Anonymous-friendly install, ~30 GB.',
    defaultPort: 27015,
    protocols: ['udp'],
    tier: 'verified',
    steamAppId: 730
  },
  {
    id: 'valheim',
    name: 'Valheim',
    source: 'steam',
    upstreamRef: 'curated/valheim',
    summary: 'Co-op Viking survival. Dedicated server bundled by the publisher.',
    defaultPort: 2456,
    protocols: ['udp'],
    tier: 'verified',
    steamAppId: 896660
  },
  {
    id: 'paper',
    name: 'Paper',
    source: 'pelican',
    upstreamRef: 'game_eggs/minecraft/java/paper/egg-paper.json',
    summary: 'High-performance Minecraft Java server (Paper). Plugins compatible with Spigot.',
    defaultPort: 25565,
    protocols: ['tcp'],
    tier: 'compatible'
  },
  {
    id: 'factorio',
    name: 'Factorio',
    source: 'steam',
    upstreamRef: 'curated/factorio',
    summary: 'Build automated factories. Multiplayer dedicated server.',
    defaultPort: 34197,
    protocols: ['udp'],
    tier: 'compatible',
    steamAppId: 427520
  },
  {
    id: 'mindustry',
    name: 'Mindustry',
    source: 'pelican',
    upstreamRef: 'mindustry/egg-mindustry.json',
    summary: 'Tower-defense factory game. Headless server bundles a single JAR.',
    defaultPort: 6567,
    protocols: ['tcp'],
    tier: 'compatible'
  },
  {
    id: 'arma3',
    name: 'Arma 3',
    source: 'steam',
    upstreamRef: 'curated/arma3',
    summary: 'Tactical military sandbox. Needs a paid Steam account on the host.',
    defaultPort: 2302,
    protocols: ['udp'],
    tier: 'experimental',
    steamAppId: 233780
  }
]

const sources = {
  'parkervcp/eggs': 'fcfd5a3549769ade15127a7577d6d3c397e83b05',
  'pelican-eggs/games': '34331fce33c83df752d94e2a90b1e43ca6280f82'
}

const initialServers = [
  {
    id: 1,
    name: 'my-tw',
    gameId: 'teeworlds',
    port: 8303,
    status: 'running',
    installDir: '/data/games/servers/my-tw',
    startCmd: 'cd /data/games/servers/my-tw && ./teeworlds_srv "sv_port 8303"'
  },
  {
    id: 2,
    name: 'mc-paper',
    gameId: 'paper',
    port: 25565,
    status: 'stopped',
    installDir: '/data/games/servers/mc-paper',
    startCmd: 'cd /data/games/servers/mc-paper && java -jar paper.jar nogui'
  }
]

function buildLog (server) {
  return [
    `[stub] runner: starting server[${server.id}] ${server.name}`,
    `[stub] runner: workDir=${server.installDir}`,
    `[stub] server[${server.id}] stdout: bound on port ${server.port}`,
    `[stub] server[${server.id}] stdout: ready, waiting for clients...`
  ]
}

export function mock () {
  let nextId = initialServers.length + 1
  const servers = [...initialServers]

  createServer({
    routes () {
      this.get('/api/v1/health', () => ({ status: 'ok' }))
      this.get('/api/v1/me', () => ({ sub: 'devstub', name: 'Dev Stub', email: 'dev@stub.local' }))
      this.get('/api/v1/games', () => games)
      this.get('/api/v1/catalog/sources', () => sources)
      this.get('/api/v1/steam/status', () => ({ linked: false, username: '' }))
      this.post('/api/v1/steam/login', (_, request) => {
        const body = JSON.parse(request.requestBody || '{}')
        if (!body.guardCode) {
          return { needsGuard: true, prompt: 'Steam Guard code (stub: any 5 chars accepted)' }
        }
        return { linked: true, username: body.username }
      })

      this.get('/api/v1/servers', () => servers)

      this.get('/api/v1/servers/:id', (_, request) => {
        const s = servers.find(x => x.id === Number(request.params.id))
        return s || new Response(404, {}, { error: 'not found' })
      })

      this.post('/api/v1/servers', (_, request) => {
        const body = JSON.parse(request.requestBody || '{}')
        const g = games.find(x => x.id === body.gameId)
        if (!g) return new Response(400, {}, { error: 'unknown game' })
        const s = {
          id: nextId++,
          name: body.name,
          gameId: body.gameId,
          port: body.port || g.defaultPort,
          status: 'stopped',
          installDir: `/data/games/servers/${body.name}`,
          startCmd: '(stub) not started yet'
        }
        servers.push(s)
        return new Response(201, {}, s)
      })

      this.delete('/api/v1/servers/:id', (_, request) => {
        const idx = servers.findIndex(x => x.id === Number(request.params.id))
        if (idx === -1) return new Response(404, {}, { error: 'not found' })
        servers.splice(idx, 1)
        return new Response(204)
      })

      // /servers/:id/{install,start,stop,restart} — flip status, return server
      const action = (next) => (_, request) => {
        const s = servers.find(x => x.id === Number(request.params.id))
        if (!s) return new Response(404, {}, { error: 'not found' })
        s.status = next
        return s
      }
      this.post('/api/v1/servers/:id/install', action('stopped'))
      this.post('/api/v1/servers/:id/start', action('running'))
      this.post('/api/v1/servers/:id/stop', action('stopped'))
      this.post('/api/v1/servers/:id/restart', action('running'))

      this.get('/api/v1/servers/:id/logs', (_, request) => {
        const s = servers.find(x => x.id === Number(request.params.id))
        if (!s) return new Response(404, {}, { error: 'not found' })
        return { lines: buildLog(s) }
      })

      this.get('/api/v1/servers/:id/query', (_, request) => {
        const s = servers.find(x => x.id === Number(request.params.id))
        if (!s) return new Response(404, {}, { error: 'not found' })
        if (s.status !== 'running') return new Response(502, {}, { error: 'server not running' })
        return {
          name: `${s.name} (stub)`,
          map: 'dm1',
          players: 3,
          maxPlayers: 16,
          gameId: s.gameId,
          ping: 8
        }
      })

      this.passthrough()
    }
  })
}
