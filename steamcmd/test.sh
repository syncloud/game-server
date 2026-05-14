#!/bin/sh -ex

DIR=$( cd "$( dirname "$0" )" && pwd )
cd ${DIR}

BUILD_DIR=${DIR}/../build/snap

id games >/dev/null 2>&1 || adduser --system --group --no-create-home games

DATA_DIR=/tmp/games-snap-data
rm -rf ${DATA_DIR}
install -d -o games -g games -m 0755 ${DATA_DIR}

mkdir -p /snap/games /var/snap/games
ln -sfn ${BUILD_DIR} /snap/games/current
ln -sfn ${DATA_DIR}  /var/snap/games/current

runuser -u games -- /snap/games/current/bin/steamcmd.sh +quit

${BUILD_DIR}/steamcmd/lib64/ld-linux-x86-64.so.2 --version
