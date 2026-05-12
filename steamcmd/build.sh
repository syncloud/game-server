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

# 32-bit runtime libs. steamcmd is i386 and so are most older HLDS / SrcDS
# binaries. We install the i386 packages system-wide so apt pulls all
# transitive deps (libssl3, libnghttp2, libidn2, libsdl2, libgl1 etc.),
# then bulk-copy /lib/i386-linux-gnu/* + /usr/lib/i386-linux-gnu/* into the
# snap. Roughly 35-40 MB, covers HTTPS-to-Steam without a per-dep apt list.
dpkg --add-architecture i386
apt update
apt -y install \
    libc6:i386 \
    libstdc++6:i386 \
    libgcc-s1:i386 \
    zlib1g:i386 \
    libcurl4:i386 \
    libtinfo6:i386 \
    libncurses6:i386 \
    libsdl2-2.0-0:i386 \
    libgl1:i386 \
    libssl3:i386

mkdir -p ${OUT}/lib32

[ -f /lib/ld-linux.so.2 ] && cp /lib/ld-linux.so.2 ${OUT}/lib32/
for src in /lib/i386-linux-gnu /usr/lib/i386-linux-gnu; do
    if [ -d "$src" ]; then
        find "$src" -maxdepth 1 \( -type f -o -type l \) -name '*.so*' -exec cp -P {} ${OUT}/lib32/ \;
    fi
done

cd ${DIR}
ls -la ${OUT}
echo "lib32 file count: $(ls ${OUT}/lib32 | wc -l)"
echo "lib32 size:"
du -sh ${OUT}/lib32
