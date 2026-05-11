#!/bin/bash -ex

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
cd ${DIR}

OUT=${DIR}/../build/snap/web/dist
mkdir -p ${OUT}

npm ci
npm run build

cp -r dist/. ${OUT}
