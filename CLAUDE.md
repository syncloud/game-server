# CI

http://ci.syncloud.org:8080/syncloud/games (via 192.168.1.101:8080).

Check builds via API:
```
curl -s "http://192.168.1.101:8080/api/repos/syncloud/games/builds?limit=5"
```

Check which step failed:
```
curl -s "http://192.168.1.101:8080/api/repos/syncloud/games/builds/{n}" | python3 -c "import json,sys; d=json.load(sys.stdin); [print(s['name'], s['status']) for st in d['stages'] for s in st['steps']]"
```

Tail a specific step (stage_number/step_number, both 1-indexed):
```
curl -s "http://192.168.1.101:8080/api/repos/syncloud/games/builds/{n}/logs/1/{step}" | python3 -c "import json,sys; [print(l.get('out','').rstrip()) for l in json.load(sys.stdin)]"
```

Artifacts at `http://ci.syncloud.org:8081/files/games/{build}-amd64/`.

# Status

WIP on `wip` branch. Issue: syncloud/platform#35.

Last green: build #26 (lib64 bundle landed), build #27 has steamcmd diagnostics.

Phases shipped:
- 0: snap skeleton, stub catalog API, store-styled Vue UI
- 1: server CRUD + SQLite (modernc.org/sqlite)
- 2: process management (start/stop/restart, SIGTERM+10s+SIGKILL, process-group)
- 3: SteamCMD vendor (amd64 only — 32-bit binary, anonymous + user/pass logins)
- 3c: amd64 lib bundle for 64-bit games (CS2/Valheim/Rust/Factorio) — snap is now host-lib-independent for both archs
- 4: Pelican egg consumer (best-effort non-docker, parses parkervcp/eggs format)
- 4b: real Teeworlds install + start + UDP-bound probe — proves the integration test really plays a game in CI
- 5: in-memory ring-buffer log endpoint (500 lines / server)
- 6: A2S_INFO Source-engine UDP server query (with challenge handshake)
- 7: OIDC client registration via `platformClient.RegisterOIDCClient`
- 8: Playwright e2e (desktop + mobile)
- 9: docs

In progress / open:
- 3b: real HLDS (CS 1.6, appid 90, 250MB) install via SteamCMD — xfailed.
  steamcmd bootstrap fails with "Steam needs to be online to update" + empty
  Steam/logs/. lib32 + writable runtime dir aren't enough. Diagnostics added
  in test_steamcmd_diagnostics — read CI build log to root-cause.
- 7b: own session middleware (replace nginx forward-auth with OIDC session
  cookie). OIDC client is registered at configure but enforcement still via
  nginx authelia_authrequest.
- 8b: Playwright login helper. Specs + config in repo but CI step disabled
  until the helper drives the authelia login form.

# Architecture

- `cli/` — Go cobra: install / configure / pre-refresh / post-refresh / cli (storage-change, access-change, backup-pre-stop, restore-pre-start, restore-post-start)
- `backend/` — Go HTTP server on `unix:/var/snap/games/current/backend.sock`
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
/snap/games/current/                # read-only squashfs
  steamcmd/                               # bundled steamcmd_linux.tar.gz contents
    linux32/steamcmd                      # i386 bootstrap binary
    steamcmd.sh, steam.sh                 # original wrapper (unused)
    lib32/                                # ~40MB. i386 base runtime:
                                          #   ld-linux.so.2
                                          #   libc.so.6, libstdc++.so.6, libgcc_s.so.1
                                          #   libcurl.so.4, libssl.so.3, libsdl2, libgl1, ...
    lib64/                                # ~25MB. amd64 base runtime:
                                          #   ld-linux-x86-64.so.2
                                          #   same lib set, amd64 variants
  web/dist/                               # Vue SPA build
  nginx/                                  # vendored nginx (etc/, opt/, usr/, lib/)
  bin/
    cli                                   # Go cobra hooks binary
    backend                               # Go HTTP backend (static)
    service.backend.sh, service.nginx.sh  # systemd wrappers
    steamcmd.sh                           # wrapper that invokes linux32/steamcmd
                                          #   via lib32/ld-linux.so.2 + lib32

/var/snap/games/current/            # writable, $SNAP_DATA
  database.db                             # SQLite catalog of installed servers
  backend.sock                            # nginx -> backend
  servers/<name>/                         # per-server install (game files)
  config/                                 # rendered Authelia includes
  oidc.secret                             # OIDC client_secret
  nginx/                                  # nginx state (temp_path etc.)
  .steam-home/                            # steamcmd's $HOME — Steam/logs/, .steam/
  .steam-runtime/                         # steamcmd's writable copy
                                          # (linux32 + public + package + steamcmd.sh)

/var/snap/games/common/             # shared across revisions, $SNAP_COMMON
  web.socket                              # platform -> nginx (PLATFORM CONTRACT)
  installed                               # marker file
```

`web.socket` must stay at `$SNAP_COMMON/web.socket` (platform contract). Everything else under `$SNAP_DATA` so refresh rollback works.

# Game launch wrappers

`installer.steamStartCmd` builds the startCmd for each Steam game using two helpers in `backend/installer/installer.go`:

- `wrapI386(binary, extraPaths, args)` — for HLDS, TF2, GMod (32-bit SrcDS):
  ```
  LD_LIBRARY_PATH=lib32[:extra] lib32/ld-linux.so.2 --library-path lib32[:extra] $bin $args
  ```
- `wrapAmd64(binary, extraPaths, args)` — for CS2, Valheim, Rust (64-bit native):
  ```
  LD_LIBRARY_PATH=lib64[:extra] lib64/ld-linux-x86-64.so.2 --library-path lib64[:extra] $bin $args
  ```

`extraPaths` typically includes `$installDir` and `$installDir/<mod>` so game-private engine libs (`libtier0_s.so`, `libsteam_api.so`, etc.) resolve.

This pattern means the snap never depends on host glibc/libstdc++/libGL state — same snap should run on any Linux host the platform supports.
