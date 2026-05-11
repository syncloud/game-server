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

ls -la ${OUT}
