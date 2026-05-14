#!/bin/sh -ex

DIR=$( cd "$( dirname "$0" )" && pwd )
cd ${DIR}

BUILD_DIR=${DIR}/../build/snap/jre

# Resolves all transitive libs against the running distro's libc. Eclipse
# Temurin 17 supports glibc >= 2.17 → both bookworm (2.36) and buster (2.28).
${BUILD_DIR}/bin/java -version
${BUILD_DIR}/bin/java --help > /dev/null
