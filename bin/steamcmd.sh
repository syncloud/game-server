#!/bin/bash -e

SCDIR=/snap/game-server/current/steamcmd
LD="${SCDIR}/lib32/ld-linux.so.2"
LIBS="${SCDIR}/lib32"

cd "${SCDIR}"
export LD_LIBRARY_PATH="${LIBS}:${LD_LIBRARY_PATH:-}"
exec "${LD}" --library-path "${LIBS}" "${SCDIR}/linux32/steamcmd" "$@"
