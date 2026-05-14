#!/bin/sh -ex

DIR=$( cd "$( dirname "$0" )" && pwd )
cd ${DIR}

BUILD_DIR=${DIR}/../build/snap

DATA_DIR=/tmp/games-snap-data
rm -rf ${DATA_DIR}
mkdir -p ${DATA_DIR}

mkdir -p /snap/games /var/snap/games
ln -sfn ${BUILD_DIR} /snap/games/current
ln -sfn ${DATA_DIR}  /var/snap/games/current

/snap/games/current/bin/steamcmd.sh +quit

${BUILD_DIR}/steamcmd/lib64/ld-linux-x86-64.so.2 --version
