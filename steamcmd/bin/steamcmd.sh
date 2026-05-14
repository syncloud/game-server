#!/bin/bash
set -e

SCDIR=/snap/games/current/steamcmd
LIBS="${SCDIR}/lib32"

export HOME=/var/snap/games/current/.steam-home
mkdir -p "${HOME}"

cd "${HOME}"

exec "${SCDIR}/linux32/ld-linux.so.2" --library-path "${SCDIR}/linux32:${LIBS}" "${SCDIR}/linux32/steamcmd" "$@"
