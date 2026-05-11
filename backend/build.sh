#!/bin/bash -ex

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
cd ${DIR}

OUT=${DIR}/../build/snap/bin
mkdir -p ${OUT}
CGO_ENABLED=0 go build -buildvcs=false -o ${OUT}/backend .
