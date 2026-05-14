#!/bin/bash -ex

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
cd ${DIR}

apt update
apt -y install wget ca-certificates

OUT=${DIR}/../build/snap/steamcmd
BIN_OUT=${DIR}/../build/snap/bin
mkdir -p ${OUT} ${BIN_OUT}

# Wrapper that the runner uses to invoke steamcmd — owned by this
# component since it knows about the lib32/lib64 layout produced below.
install -m 0755 ${DIR}/bin/steamcmd.sh ${BIN_OUT}/steamcmd.sh

wget -q https://media.steampowered.com/installer/steamcmd_linux.tar.gz -O steamcmd.tar.gz

# Pin the bootstrap tarball: snap refresh is the only thing that should
# change steamcmd's version in prod, so the build refuses to silently
# pick up a republished tarball. Pass STEAMCMD_SHA256 in CI; leave it
# unset for ad-hoc local builds and the script just prints the actual
# hash so you can bump the constant.
ACTUAL_SHA=$(sha256sum steamcmd.tar.gz | awk '{print $1}')
echo "steamcmd_linux.tar.gz sha256: ${ACTUAL_SHA}"
if [ -n "${STEAMCMD_SHA256:-}" ] && [ "${STEAMCMD_SHA256}" != "${ACTUAL_SHA}" ]; then
    echo "ERROR: tarball sha256 mismatch — expected ${STEAMCMD_SHA256}" >&2
    exit 1
fi

tar xf steamcmd.tar.gz -C ${OUT}
rm steamcmd.tar.gz

# 32-bit runtime: steamcmd is i386, and so are HLDS, CS 1.6, TF2, CS:GO
# legacy, Gmod, L4D2. apt resolves the transitive closure for us; bulk-copy
# the resulting /lib/i386-linux-gnu and /usr/lib/i386-linux-gnu.
dpkg --add-architecture i386
apt update
apt -y install \
    libc6:i386 libstdc++6:i386 libgcc-s1:i386 \
    zlib1g:i386 libcurl4:i386 libssl3:i386 \
    libtinfo6:i386 libncurses6:i386 \
    libsdl2-2.0-0:i386 libgl1:i386 \
    curl:i386

mkdir -p ${OUT}/lib32
# Also bundle the i386 curl binary so we can independently verify 32-bit
# HTTPS works from inside the snap (diagnostic only).
cp /usr/bin/curl ${OUT}/lib32/curl.i386 || cp /usr/bin/curl.i386 ${OUT}/lib32/curl.i386 || true
[ -f /lib/ld-linux.so.2 ] && cp /lib/ld-linux.so.2 ${OUT}/lib32/
for src in /lib/i386-linux-gnu /usr/lib/i386-linux-gnu; do
    if [ -d "$src" ]; then
        find "$src" -maxdepth 1 \( -type f -o -type l \) -name '*.so*' -exec cp -P {} ${OUT}/lib32/ \;
    fi
done

# 64-bit runtime: CS2, Valheim, Rust, Project Zomboid, Factorio, Minecraft
# Java. Same pattern. Steam games copy-deploy their engine libs but expect
# host libc/libstdc++/libgcc/libpthread/libGL for both archs. Bundling these
# makes the snap independent of host multiarch state.
apt -y install \
    libc6:amd64 libstdc++6:amd64 libgcc-s1:amd64 \
    zlib1g:amd64 libcurl4:amd64 libssl3:amd64 \
    libtinfo6:amd64 libncurses6:amd64 \
    libsdl2-2.0-0:amd64 libgl1:amd64

mkdir -p ${OUT}/lib64
[ -f /lib64/ld-linux-x86-64.so.2 ] && cp /lib64/ld-linux-x86-64.so.2 ${OUT}/lib64/
for src in /lib/x86_64-linux-gnu /usr/lib/x86_64-linux-gnu; do
    if [ -d "$src" ]; then
        find "$src" -maxdepth 1 \( -type f -o -type l \) -name '*.so*' -exec cp -P {} ${OUT}/lib64/ \;
    fi
done

# Patch steamcmd's ELF interpreter to point at our bundled i386 ld-linux
# in /snap (the path is stable once the snap is installed). Why this
# matters: steamcmd derives STEAMROOT from /proc/self/exe and chdirs
# there. If we wrap with `exec ld-linux --library-path X binary`,
# /proc/self/exe reports the ld-linux path; otherwise it's the binary
# path. We want the binary path so STEAMROOT is the writable RUNTIME copy.
# NOTE: we tried patching the binary's PT_INTERP at build time so the
# kernel would load our bundled ld-linux directly — that fails with
# kernel ENOENT on PT_INTERP load in the snap mount namespace, even
# though the file is verifiably present. Sidestep by invoking ld-linux
# explicitly in bin/steamcmd.sh instead. Leave the binary's interpreter
# at its default.

cd ${DIR}

# Pre-bootstrap. Out of the box Valve's tarball is a ~3MB shim that
# downloads the full ~30MB client on first launch — that's where the
# "Steam needs to be online" cascade happened, and it's what forced the
# runtime-copy gymnastics in the wrapper. Run +quit here, in the build
# container with network and writable cwd, so the snap ships a fully
# baked client. At runtime the install is read-only squashfs — Valve
# does silently re-check for updates on startup but with our bundled
# version current and the install RO, the check is a no-op (or fails
# its write and steamcmd falls back to the current install). snap
# refresh is the only thing that re-rolls the bundled version.
BOOT_HOME=$(mktemp -d)
HOME=${BOOT_HOME} ${OUT}/linux32/steamcmd +quit
rm -rf ${BOOT_HOME}

ls -la ${OUT}
echo "lib32 file count: $(ls ${OUT}/lib32 | wc -l), size: $(du -sh ${OUT}/lib32 | cut -f1)"
echo "lib64 file count: $(ls ${OUT}/lib64 | wc -l), size: $(du -sh ${OUT}/lib64 | cut -f1)"
echo "steamcmd post-bootstrap size: $(du -sh ${OUT} | cut -f1)"
