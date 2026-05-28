#!/bin/bash -ex

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
BUILD_DIR=${DIR}/../build/snap/archivers

export LD_LIBRARY_PATH=${BUILD_DIR}/lib
export PATH=${BUILD_DIR}/bin:${PATH}

# --version smoke test: proves the bundled binary loads on this platform
# image without relying on the host's glibc/libstdc++ state.
# Extraction itself is exercised by the bedrock + sauerbraten e2e
# install tests in test/test.py.
tar --version | head -1
unzip -v | head -1
bzip2 --version 2>&1 | head -1
xz --version | head -1
gzip --version | head -1

echo "archivers smoke test ok"
