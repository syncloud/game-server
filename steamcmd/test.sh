#!/bin/sh -ex

DIR=$( cd "$( dirname "$0" )" && pwd )
cd ${DIR}

BUILD_DIR=${DIR}/../build/snap

# Wrapper now reads from a pre-bootstrapped /snap (RO) and writes only
# to $HOME under $SNAP_DATA. So all the binary test needs is a games
# user and a writable HOME pointed at via HOME_OVERRIDE — no /snap or
# /var/snap symlinks, no chown dance.
id games >/dev/null 2>&1 || adduser --system --group --no-create-home games

HOME_DIR=/tmp/games-steam-home
rm -rf ${HOME_DIR}
install -d -o games -g games -m 0755 ${HOME_DIR}

mkdir -p /snap/games
ln -sfn ${BUILD_DIR} /snap/games/current

# Run as the same user the snap declares (apps.backend.user = games).
# +quit (an alias for +exit) is the lightest path through steamcmd that
# still exercises the wrapper's HOME setup, library path, and ld-linux
# invocation against the pre-bootstrapped install.
runuser -u games -- env HOME_OVERRIDE=${HOME_DIR} \
    /snap/games/current/bin/steamcmd.sh +quit

# 64-bit loader: nothing in the snap invokes it directly (per-game start
# commands do via wrapAmd64). Sanity-check it can at least link its own
# libs against this distro's filesystem layout.
${BUILD_DIR}/steamcmd/lib64/ld-linux-x86-64.so.2 --version
