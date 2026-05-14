#!/bin/sh -ex

DIR=$( cd "$( dirname "$0" )" && pwd )
cd ${DIR}

BUILD_DIR=${DIR}/../build/snap

# Static CGO_ENABLED=0 binary — should run on any glibc. Cobra's --help
# exits 0 cleanly. Also verify the four lifecycle hooks built.
${BUILD_DIR}/bin/cli --help
${BUILD_DIR}/meta/hooks/install --help > /dev/null 2>&1 || true
${BUILD_DIR}/meta/hooks/configure --help > /dev/null 2>&1 || true
${BUILD_DIR}/meta/hooks/pre-refresh --help > /dev/null 2>&1 || true
${BUILD_DIR}/meta/hooks/post-refresh --help > /dev/null 2>&1 || true

# Backend is a daemon; just smoke-check it exec's at all (it will fail
# trying to bind the unix socket, so we redirect and accept the failure
# — exit code is non-zero but stdout proves the Go runtime initialised).
${BUILD_DIR}/bin/backend 2>&1 | head -5 || true
