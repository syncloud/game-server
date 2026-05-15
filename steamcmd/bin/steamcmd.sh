#!/bin/bash
set -e

SCDIR=/snap/games/current/steamcmd
LIBS="${SCDIR}/lib32"
RUNTIME=/var/snap/games/current/.steam-runtime

if [ ! -x "${RUNTIME}/linux32/steamcmd" ]; then
    mkdir -p "${RUNTIME}"
    for f in "${SCDIR}"/*; do
        name=$(basename "$f")
        case "$name" in
            lib32|lib64) continue ;;
        esac
        cp -r "$f" "${RUNTIME}/"
    done
fi

cp -f "${LIBS}/ld-linux.so.2" "${RUNTIME}/linux32/ld-linux.so.2"

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
