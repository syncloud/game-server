#!/bin/bash
set -e

SCDIR=/snap/games/current/steamcmd
LIBS="${SCDIR}/lib32"

export HOME=/var/snap/games/current/.steam-home
mkdir -p "${HOME}"

export LD_LIBRARY_PATH="${SCDIR}/linux32:${LIBS}:${LD_LIBRARY_PATH:-}"
export SSL_CERT_FILE="${SSL_CERT_FILE:-/etc/ssl/certs/ca-certificates.crt}"

cd "${HOME}"

exec "${SCDIR}/linux32/ld-linux.so.2" --library-path "${SCDIR}/linux32:${LIBS}" "${SCDIR}/linux32/steamcmd" "$@"
