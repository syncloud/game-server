#!/bin/bash
set -e

# steamcmd's install dir is pre-bootstrapped at build time (see
# steamcmd/build.sh) and lives read-only on the squashfs at /snap.
# Only the user-state surface — Steam/ home dir, sentry, app manifests —
# is writable, under $SNAP_DATA. snap refresh is the only thing that
# rolls the steamcmd version forward.
SCDIR=/snap/games/current/steamcmd
LIBS="${SCDIR}/lib32"

export HOME="${HOME_OVERRIDE:-/var/snap/games/current/.steam-home}"
mkdir -p "${HOME}"

# Library order: steamcmd ships its OWN linux32/libstdc++.so.6 (2013-era
# 32-bit ABI it was built against) — must win over our bookworm bundle.
# Mirror the official steamcmd.sh: prepend linux32 to LD_LIBRARY_PATH.
export LD_LIBRARY_PATH="${SCDIR}/linux32:${LIBS}:${LD_LIBRARY_PATH:-}"
export SSL_CERT_FILE="${SSL_CERT_FILE:-/etc/ssl/certs/ca-certificates.crt}"

# cwd has to be writable so steamcmd can drop transient logs without
# tripping over the RO install dir.
cd "${HOME}"

exec "${SCDIR}/linux32/ld-linux.so.2" --library-path "${SCDIR}/linux32:${LIBS}" "${SCDIR}/linux32/steamcmd" "$@"
