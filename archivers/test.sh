#!/bin/bash -ex

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
BUILD_DIR=${DIR}/../build/snap/archivers
LD=${BUILD_DIR}/lib/ld-linux-x86-64.so.2
LIB=${BUILD_DIR}/lib

run() {
    "${LD}" --library-path "${LIB}" "${BUILD_DIR}/bin/$1" "${@:2}"
}

run tar --version | head -1
run unzip -v | head -1
run bzip2 --version 2>&1 | head -1
run xz --version | head -1
run gzip --version | head -1

echo "archivers smoke test ok"
