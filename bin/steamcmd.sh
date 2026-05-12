#!/bin/bash -e

# /snap/ is squashfs (read-only); steamcmd self-updates `package/` and writes
# state next to its binary, so we copy the bundle to a writable runtime dir
# under $SNAP_DATA on first run. The lib32/ stays in /snap (doesn't change).
SCDIR=/snap/game-server/current/steamcmd
LIBS="${SCDIR}/lib32"
LD="${LIBS}/ld-linux.so.2"

RUNTIME=/var/snap/game-server/current/.steam-runtime
if [ ! -x "${RUNTIME}/linux32/steamcmd" ]; then
    mkdir -p "${RUNTIME}"
    # Copy everything from the bundled steamcmd dir except lib32/lib64 (which
    # stay in /snap — they don't mutate). Picks up linux32/, public/,
    # steamcmd.sh and any future tarball additions automatically.
    for f in "${SCDIR}"/*; do
        name=$(basename "$f")
        case "$name" in
            lib32|lib64) continue ;;
        esac
        cp -r "$f" "${RUNTIME}/"
    done
    # If we happen to be running as root (install hook, manual SSH diag, etc.)
    # ensure the runtime + game-server's HOME end up owned by game-server,
    # otherwise the backend service (which runs as game-server) can't write
    # back into them and steamcmd dies with permission errors / exit 1.
    if [ "$(id -u)" = "0" ]; then
        chown -R game-server:game-server "${RUNTIME}"
        chown -R game-server:game-server "${HOME_OVERRIDE:-/var/snap/game-server/current/.steam-home}" 2>/dev/null || true
    fi
fi

export HOME="${HOME_OVERRIDE:-/var/snap/game-server/current/.steam-home}"
mkdir -p "${HOME}"

cd "${RUNTIME}"

# Library order matters: steamcmd ships its OWN linux32/libstdc++.so.6
# (2013-era 32-bit libstdc++ it was built against). The official
# steamcmd.sh prepends linux32/ to LD_LIBRARY_PATH so Steam's libstdc++
# wins over system libs. Mirror that — fall back to our bookworm lib32
# only for libs Steam doesn't ship (libc, libssl, libcurl, nss_*, ...).
export LD_LIBRARY_PATH="${RUNTIME}/linux32:${LIBS}:${LD_LIBRARY_PATH:-}"
export SSL_CERT_FILE="${SSL_CERT_FILE:-/etc/ssl/certs/ca-certificates.crt}"
exec "${LD}" --library-path "${RUNTIME}/linux32:${LIBS}" "${RUNTIME}/linux32/steamcmd" "$@"
