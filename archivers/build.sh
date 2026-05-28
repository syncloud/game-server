#!/bin/bash -ex

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
cd ${DIR}

apt update
apt -y install tar unzip bzip2 xz-utils gzip file

OUT=${DIR}/../build/snap/archivers
rm -rf ${OUT}
mkdir -p ${OUT}/bin ${OUT}/lib

for bin in tar unzip bzip2 xz gzip; do
    src=$(command -v ${bin})
    if [ -z "${src}" ]; then
        echo "missing required binary: ${bin}" >&2
        exit 1
    fi
    cp -L "${src}" ${OUT}/bin/${bin}
done

cp ${DIR}/bin/*.sh ${OUT}/bin/
chmod +x ${OUT}/bin/*.sh

for bin in ${OUT}/bin/*; do
    ldd "${bin}" 2>/dev/null \
        | awk '/=> \//{print $3} /ld-linux/{print $1}' \
        | while read lib; do
            [ -f "${lib}" ] || continue
            cp -L "${lib}" "${OUT}/lib/$(basename ${lib})" 2>/dev/null || true
        done
done

echo "archivers bundle:"
ls -la ${OUT}/bin ${OUT}/lib
du -sh ${OUT}
