#!/bin/bash
set -e

# /snap/ is squashfs (read-only); steamcmd self-updates `package/` and writes
# state next to its binary, so we copy the bundle to a writable runtime dir
# under $SNAP_DATA on first run. The lib32/ stays in /snap (doesn't change).
SCDIR=/snap/game-server/current/steamcmd
LIBS="${SCDIR}/lib32"

RUNTIME=/var/snap/game-server/current/.steam-runtime
if [ ! -x "${RUNTIME}/linux32/steamcmd" ]; then
    mkdir -p "${RUNTIME}"
    # Copy everything from the bundled steamcmd dir except lib32/lib64.
    for f in "${SCDIR}"/*; do
        name=$(basename "$f")
        case "$name" in
            lib32|lib64) continue ;;
        esac
        cp -r "$f" "${RUNTIME}/"
    done
    if [ "$(id -u)" = "0" ]; then
        chown -R game-server:game-server "${RUNTIME}"
        chown -R game-server:game-server "${HOME_OVERRIDE:-/var/snap/game-server/current/.steam-home}" 2>/dev/null || true
    fi
fi

# Put a copy of ld-linux into the writable runtime dir alongside steamcmd.
# Why we invoke it from THERE (not from /snap/.../lib32/): the loader's
# path becomes /proc/self/exe, which steamcmd reads to derive STEAMROOT.
# If STEAMROOT lands in /snap (read-only squashfs) Steam fails its disk-
# space check and prints the misleading "needs to be online" / "needs 100MB"
# cascade. With ld-linux running from $RUNTIME, STEAMROOT is writable.
cp -f "${LIBS}/ld-linux.so.2" "${RUNTIME}/linux32/ld-linux.so.2"

export HOME="${HOME_OVERRIDE:-/var/snap/game-server/current/.steam-home}"
mkdir -p "${HOME}"

cd "${RUNTIME}"

# Library order: steamcmd ships its OWN linux32/libstdc++.so.6 (2013-era
# 32-bit ABI it was built against) — must win over our bookworm bundle.
# Mirror the official steamcmd.sh: prepend linux32 to LD_LIBRARY_PATH.
export LD_LIBRARY_PATH="${RUNTIME}/linux32:${LIBS}:${LD_LIBRARY_PATH:-}"
export SSL_CERT_FILE="${SSL_CERT_FILE:-/etc/ssl/certs/ca-certificates.crt}"

exec "${RUNTIME}/linux32/ld-linux.so.2" --library-path "${RUNTIME}/linux32:${LIBS}" "${RUNTIME}/linux32/steamcmd" "$@"
