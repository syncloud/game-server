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
    # Copy everything from the bundled steamcmd dir except lib32 (which stays
    # in /snap — it doesn't mutate). Picks up linux32/, public/, steamcmd.sh,
    # and any future tarball additions automatically.
    for f in "${SCDIR}"/*; do
        name=$(basename "$f")
        case "$name" in
            lib32|lib64) continue ;;
        esac
        cp -r "$f" "${RUNTIME}/"
    done
fi

export HOME="${HOME_OVERRIDE:-/var/snap/game-server/current/.steam-home}"
mkdir -p "${HOME}"

cd "${RUNTIME}"

export LD_LIBRARY_PATH="${LIBS}:${LD_LIBRARY_PATH:-}"
export SSL_CERT_FILE="${SSL_CERT_FILE:-/etc/ssl/certs/ca-certificates.crt}"
exec "${LD}" --library-path "${LIBS}" "${RUNTIME}/linux32/steamcmd" "$@"
