#!/bin/bash -ex

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
DATA=${DIR}/../backend/catalog/data

rm -rf ${DATA}
mkdir -p ${DATA}
for src in parkervcp pelican-eggs linuxgsm; do
    if compgen -G "${DIR}/${src}/*.json" > /dev/null; then
        mkdir -p ${DATA}/${src}
        cp ${DIR}/${src}/*.json ${DATA}/${src}/
    fi
done

echo "catalog data populated under ${DATA}:"
find ${DATA} -name '*.json' | wc -l
