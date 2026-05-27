#!/bin/bash
set -e

SCDIR=/snap/games/current/steamcmd
LIBS="${SCDIR}/lib32"
RUNTIME=/var/snap/games/current/.steam-runtime

export HOME=/var/snap/games/current/.steam-home
mkdir -p "${HOME}"

export LD_LIBRARY_PATH="${RUNTIME}/linux32:${LIBS}:${LD_LIBRARY_PATH:-}"

cd "${RUNTIME}"

STATUS=42
while [ "${STATUS}" -eq 42 ]; do
    set +e
    "${RUNTIME}/linux32/ld-linux.so.2" --library-path "${RUNTIME}/linux32:${LIBS}" "${RUNTIME}/linux32/steamcmd" "$@"
    STATUS=$?
    set -e
done
exit "${STATUS}"
