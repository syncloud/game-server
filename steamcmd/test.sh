#!/bin/sh -ex

DIR=$( cd "$( dirname "$0" )" && pwd )
cd ${DIR}

BUILD_DIR=${DIR}/../build/snap

id games >/dev/null 2>&1 || adduser --system --group --no-create-home games

HOME_DIR=/tmp/games-steam-home
rm -rf ${HOME_DIR}
install -d -o games -g games -m 0755 ${HOME_DIR}

mkdir -p /snap/games
ln -sfn ${BUILD_DIR} /snap/games/current

runuser -u games -- env HOME_OVERRIDE=${HOME_DIR} \
    /snap/games/current/bin/steamcmd.sh +quit

${BUILD_DIR}/steamcmd/lib64/ld-linux-x86-64.so.2 --version
