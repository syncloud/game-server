#!/bin/sh -ex

DIR=$( cd "$( dirname "$0" )" && pwd )
cd ${DIR}

BUILD_DIR=${DIR}/../build/snap/jre

${BUILD_DIR}/bin/java -version
${BUILD_DIR}/bin/java --help > /dev/null
