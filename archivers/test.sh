#!/bin/bash -ex

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
BIN=${DIR}/../build/snap/archivers/bin

${BIN}/tar.sh --version | head -1
${BIN}/unzip.sh -v | head -1
${BIN}/bzip2.sh --version 2>&1 | head -1
${BIN}/xz.sh --version | head -1
${BIN}/gzip.sh --version | head -1

echo "archivers smoke test ok"
