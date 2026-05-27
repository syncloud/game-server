#!/bin/sh -ex

DIR=$( cd "$( dirname "$0" )" && pwd )
cd ${DIR}

BUILD_DIR=${DIR}/../build/snap

${BUILD_DIR}/bin/cli --help
${BUILD_DIR}/meta/hooks/install --help > /dev/null 2>&1 || true
${BUILD_DIR}/meta/hooks/configure --help > /dev/null 2>&1 || true
${BUILD_DIR}/meta/hooks/pre-refresh --help > /dev/null 2>&1 || true
${BUILD_DIR}/meta/hooks/post-refresh --help > /dev/null 2>&1 || true

${BUILD_DIR}/bin/backend 2>&1 | head -5 || true
