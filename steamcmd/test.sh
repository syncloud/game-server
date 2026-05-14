#!/bin/sh -ex

DIR=$( cd "$( dirname "$0" )" && pwd )
cd ${DIR}

BUILD_DIR=${DIR}/../build/snap

# Create games user upfront — both the wrapper-as-games invocation
# below AND any chown the wrapper attempts need it to exist.
id games >/dev/null 2>&1 || adduser --system --group --no-create-home games

# Writable scratch for what the snap would expose as $SNAP_DATA. Use a
# known path (not mktemp) so the parent dir is world-traversable; mktemp
# defaults to mode 700 which blocks the games user from cd-ing through.
SCRATCH=/tmp/games-snap-data
rm -rf ${SCRATCH}
install -d -o games -g games -m 0755 ${SCRATCH}

mkdir -p /snap/games /var/snap/games
ln -sfn ${BUILD_DIR} /snap/games/current
ln -sfn ${SCRATCH}   /var/snap/games/current

# On a real device the backend runs the wrapper as user `games` (snap.yaml
# apps.backend.user = games). Match that here so steamcmd's STEAMROOT
# discovery sees a games-owned runtime tree from the start, and the
# wrapper's root-only chown branch is skipped (which is what prod does).
runuser -u games -- /snap/games/current/bin/steamcmd.sh +exit

# 64-bit loader: nothing in the snap invokes it directly (per-game start
# commands do via wrapAmd64). Sanity-check it can at least link its own
# libs against this distro's filesystem layout.
${BUILD_DIR}/steamcmd/lib64/ld-linux-x86-64.so.2 --version
