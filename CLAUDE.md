# CI

http://ci.syncloud.org:8080/syncloud/game-server (via 192.168.1.101:8080).

Check builds via API:
```
curl -s "http://192.168.1.101:8080/api/repos/syncloud/game-server/builds?limit=5"
```

Check which step failed:
```
curl -s "http://192.168.1.101:8080/api/repos/syncloud/game-server/builds/{n}" | python3 -c "import json,sys; d=json.load(sys.stdin); [print(s['name'], s['status']) for st in d['stages'] for s in st['steps']]"
```

Tail a specific step (stage_number/step_number, both 1-indexed):
```
curl -s "http://192.168.1.101:8080/api/repos/syncloud/game-server/builds/{n}/logs/1/{step}" | python3 -c "import json,sys; [print(l.get('out','').rstrip()) for l in json.load(sys.stdin)]"
```

Artifacts at `http://ci.syncloud.org:8081/files/game-server/{build}-amd64/`.

# Status

WIP on `wip` branch. Issue: syncloud/platform#35.

Phases shipped:
- 0: snap skeleton, stub catalog API, store-styled Vue UI
- 1: server CRUD + SQLite (modernc.org/sqlite)
- 2: process management (start/stop/restart, SIGTERM+10s+SIGKILL, process-group)
- 3: SteamCMD vendor (amd64 only — 32-bit binary, anonymous + user/pass logins)
- 4: Pelican egg consumer (best-effort non-docker, parses parkervcp/eggs format)
- 5: in-memory ring-buffer log endpoint (500 lines / server)
- 6: A2S_INFO Source-engine UDP server query (with challenge handshake)
- 7: OIDC client registration via `platformClient.RegisterOIDCClient`
- 8: Playwright e2e (desktop + mobile)
- 9: docs

# Architecture

- `cli/` — Go cobra: install / configure / pre-refresh / post-refresh / cli (storage-change, access-change, backup-pre-stop, restore-pre-start, restore-post-start)
- `backend/` — Go HTTP server on `unix:/var/snap/game-server/current/backend.sock`
  - `db/` — SQLite open + schema
  - `server/` — Store with CRUD on servers
  - `runner/` — exec.Cmd-based process management with ring-buffer log capture
  - `installer/` — SteamCMD + Pelican egg install logic
  - `query/` — A2S_INFO UDP query
- `web/` — Vue 3 + Vite. Plain CSS theme cribbed from `../store/web` (light/dark, no Element Plus). `web/e2e/` is a self-contained Playwright TS package.
- `config/` — `nginx.conf` + Authelia forward-auth templates (`{{ .AuthLocalSocket }}` + `{{ .AuthUrl }}` rendered by `config.Generate`)
- `nginx/` — vendored nginx (build.sh copies the entire FS from the nginx docker image)
- `steamcmd/` — `build.sh` downloads steamcmd_linux.tar.gz
- `test/` — pytest integration tests using `syncloud-lib`

# Constraints

- **amd64 only.** `.drone.jsonnet` lists only amd64. SteamCMD ships x86_64.
- **SteamCMD is 32-bit i386.** Snap will need 32-bit glibc bundled — first SteamCMD-based test installs may fail until that's added. Tracked as a follow-up.
- **Pelican eggs that need Docker won't work.** Backend runs install scripts directly. Best-effort for bash/native eggs (Teeworlds, Minetest work).
- **Auth = Authelia forward-auth + OIDC registration.** OIDC client is registered at configure time (for future use); active enforcement is still nginx `auth_request` against Authelia local socket.

# Integration test fixture

Use **teeworlds** for any test that needs a real install — smallest server in parkervcp/eggs (~10MB binary, headless, A2S-queryable). Egg: `https://raw.githubusercontent.com/parkervcp/eggs/master/game_eggs/teeworlds/egg-teeworlds.json`. Anonymous-friendly Steam apps (cs2, tf2, gmod, valheim, zomboid, ark) work for Steam-path tests.

# Storage layout (on device)

```
/snap/game-server/current/           # read-only, refreshed by snap install
  steamcmd/                          # bundled SteamCMD bootstrap (~5MB)
  web/dist/                          # SPA assets
  nginx/                             # vendored nginx
  bin/                               # service wrappers + Go binaries

/var/snap/game-server/current/       # writable, $SNAP_DATA
  database.db                        # SQLite catalog of installed servers
  backend.sock                       # nginx -> backend
  servers/<name>/                    # per-server install + state
  config/                            # rendered Authelia includes
  oidc.secret                        # OIDC client_secret (set if registered)

/var/snap/game-server/common/        # shared across revisions, $SNAP_COMMON
  web.socket                         # platform -> nginx
  installed                          # marker file
```

Per platform contract, `web.socket` must live at `$SNAP_COMMON/web.socket` — do not move it. Everything else lives under `$SNAP_DATA` so install/refresh rollback works.
