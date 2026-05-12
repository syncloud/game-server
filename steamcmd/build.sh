#!/bin/bash -ex

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
cd ${DIR}

apt update
apt -y install wget ca-certificates dpkg-dev

OUT=${DIR}/../build/snap/steamcmd
mkdir -p ${OUT}

wget -q https://media.steampowered.com/installer/steamcmd_linux.tar.gz -O steamcmd.tar.gz
tar xf steamcmd.tar.gz -C ${OUT}
rm steamcmd.tar.gz

# 32-bit runtime libs for steamcmd (and any 32-bit HLDS / SrcDS binaries).
# steamcmd itself is i386; many older dedicated servers are too. We bundle a
# minimal lib32 set so the snap doesn't depend on host multiarch.
dpkg --add-architecture i386
apt update
mkdir -p /tmp/i386 ${OUT}/lib32

cd /tmp/i386
apt download \
    libc6:i386 \
    libstdc++6:i386 \
    libgcc-s1:i386 \
    zlib1g:i386 \
    libcurl4:i386 \
    libtinfo6:i386 \
    libncurses6:i386
for deb in *.deb; do
    dpkg -x "$deb" /tmp/i386/extract
done

# i386 dynamic linker
find /tmp/i386/extract -name 'ld-linux.so.2' -exec cp -v {} ${OUT}/lib32/ \;
# everything from /lib/i386-linux-gnu and /usr/lib/i386-linux-gnu
if [ -d /tmp/i386/extract/lib/i386-linux-gnu ]; then
    cp -P /tmp/i386/extract/lib/i386-linux-gnu/* ${OUT}/lib32/
fi
if [ -d /tmp/i386/extract/usr/lib/i386-linux-gnu ]; then
    cp -P /tmp/i386/extract/usr/lib/i386-linux-gnu/* ${OUT}/lib32/
fi

cd ${DIR}
ls -la ${OUT}
ls ${OUT}/lib32 | head -30
echo "lib32 size:"
du -sh ${OUT}/lib32
