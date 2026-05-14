#!/bin/sh -ex

DIR=$( cd "$( dirname "$0" )" && pwd )
cd ${DIR}

BUILD_DIR=${DIR}/../build/snap

# The wrapper at bin/steamcmd.sh hardcodes /snap/games/current/* and
# /var/snap/games/current/.steam-runtime — same paths the snap exposes
# on a real device. Symlink them to the CI build output + a writable
# scratch dir so we can exercise the actual wrapper instead of
# reconstructing the loader/library-path invocation by hand. If the
# wrapper drifts (lib order, runtime-dir layout, chown timing) the
# test catches it.
SCRATCH=$(mktemp -d)
mkdir -p /snap/games /var/snap/games
ln -sfn ${BUILD_DIR} /snap/games/current
ln -sfn ${SCRATCH}   /var/snap/games/current

# Wrapper chowns the runtime dir to games:games when run as root; create
# the user so set -e doesn't kill the script on a missing group.
id games >/dev/null 2>&1 || adduser --system --group --no-create-home games

# +exit short-circuits steamcmd after the loader hands off — proves the
# wrapper's path math, the lib32 closure, the linux32 self-update copy
# and the final ld-linux invocation all work end to end.
/snap/games/current/bin/steamcmd.sh +exit

# 64-bit loader: nothing in the snap invokes it directly (per-game start
# commands do via wrapAmd64). Sanity-check it can at least link its own
# libs against this distro's filesystem layout.
${BUILD_DIR}/steamcmd/lib64/ld-linux-x86-64.so.2 --version
