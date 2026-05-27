#!/bin/bash -ex

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
cd ${DIR}

apt update
apt -y install wget ca-certificates

OUT=${DIR}/../build/snap/steamcmd
BIN_OUT=${DIR}/../build/snap/bin
mkdir -p ${OUT} ${BIN_OUT}

install -m 0755 ${DIR}/bin/steamcmd.sh ${BIN_OUT}/steamcmd.sh

wget -q https://media.steampowered.com/installer/steamcmd_linux.tar.gz -O steamcmd.tar.gz
tar xf steamcmd.tar.gz -C ${OUT}
rm steamcmd.tar.gz

dpkg --add-architecture i386
apt update
apt -y install \
    libc6:i386 libstdc++6:i386 libgcc-s1:i386 \
    zlib1g:i386 libcurl4:i386 libssl3:i386 \
    libtinfo6:i386 libncurses6:i386 \
    libsdl2-2.0-0:i386 libgl1:i386 \
    curl:i386

mkdir -p ${OUT}/lib32
cp /usr/bin/curl ${OUT}/lib32/curl.i386 || cp /usr/bin/curl.i386 ${OUT}/lib32/curl.i386 || true
[ -f /lib/ld-linux.so.2 ] && cp /lib/ld-linux.so.2 ${OUT}/lib32/
for src in /lib/i386-linux-gnu /usr/lib/i386-linux-gnu; do
    if [ -d "$src" ]; then
        find "$src" -maxdepth 1 \( -type f -o -type l \) -name '*.so*' -exec cp -P {} ${OUT}/lib32/ \;
    fi
done

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

cd ${DIR}

ls -la ${OUT}
echo "lib32 file count: $(ls ${OUT}/lib32 | wc -l), size: $(du -sh ${OUT}/lib32 | cut -f1)"
echo "lib64 file count: $(ls ${OUT}/lib64 | wc -l), size: $(du -sh ${OUT}/lib64 | cut -f1)"
