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

SRC=/snap/games/current/steamcmd
RUNTIME=/var/snap/games/current/.steam-runtime
mkdir -p ${RUNTIME}
for f in ${SRC}/*; do
    name=$(basename "$f")
    case "$name" in lib32|lib64) continue ;; esac
    cp -r "$f" ${RUNTIME}/
done
cp -f ${SRC}/lib32/ld-linux.so.2 ${RUNTIME}/linux32/ld-linux.so.2

/snap/games/current/bin/steamcmd.sh +quit

${BUILD_DIR}/steamcmd/lib64/ld-linux-x86-64.so.2 --version
