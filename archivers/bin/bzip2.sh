#!/bin/sh
DIR="$(dirname "$(readlink -f "$0")")"
exec "${DIR}/../lib/ld-linux-x86-64.so.2" \
    --library-path "${DIR}/../lib" \
    "${DIR}/bzip2" "$@"
