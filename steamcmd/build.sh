#!/bin/bash -ex

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
cd ${DIR}

apt update
apt -y install wget ca-certificates

OUT=${DIR}/../build/snap/steamcmd
mkdir -p ${OUT}

wget -q https://media.steampowered.com/installer/steamcmd_linux.tar.gz -O steamcmd.tar.gz
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
ls -la ${OUT}
echo "lib32 file count: $(ls ${OUT}/lib32 | wc -l), size: $(du -sh ${OUT}/lib32 | cut -f1)"
echo "lib64 file count: $(ls ${OUT}/lib64 | wc -l), size: $(du -sh ${OUT}/lib64 | cut -f1)"
