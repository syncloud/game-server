#!/bin/bash -e

SCDIR=/snap/game-server/current/steamcmd
LD="${SCDIR}/lib32/ld-linux.so.2"
LIBS="${SCDIR}/lib32"

# steamcmd writes self-update cache, dumps, config to $HOME — point it at a
# writable spot under $SNAP_DATA so it survives across runs and doesn't try
# to write to a non-writable /.
export HOME="${HOME_OVERRIDE:-/var/snap/game-server/current/.steam-home}"
mkdir -p "${HOME}"

# steamcmd looks up its bootstrap (steamcmd_linux files, package/, etc.)
# relative to PWD. Stay in SCDIR — not HOME, not the install dir.
cd "${SCDIR}"

# Mask the bundled libs into the linker search path then exec the i386
# binary directly via our bundled ld-linux. SSL_CERT_FILE so libcurl can
# verify https://steamcontent.com etc. through the host's CA bundle.
export LD_LIBRARY_PATH="${LIBS}:${LD_LIBRARY_PATH:-}"
export SSL_CERT_FILE="${SSL_CERT_FILE:-/etc/ssl/certs/ca-certificates.crt}"
exec "${LD}" --library-path "${LIBS}" "${SCDIR}/linux32/steamcmd" "$@"
