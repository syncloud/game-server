#!/bin/bash -e

SCDIR=/snap/game-server/current/steamcmd
LD="${SCDIR}/lib32/ld-linux.so.2"
LIBS="${SCDIR}/lib32"

# steamcmd writes self-update cache, dumps, config to $HOME — point it at a
# writable spot under $SNAP_DATA so it survives across runs and doesn't try
# to write to a non-writable /.
export HOME="${HOME_OVERRIDE:-/var/snap/game-server/current/.steam-home}"
mkdir -p "${HOME}"
cd "${HOME}"
export LD_LIBRARY_PATH="${LIBS}:${LD_LIBRARY_PATH:-}"
exec "${LD}" --library-path "${LIBS}" "${SCDIR}/linux32/steamcmd" "$@"
