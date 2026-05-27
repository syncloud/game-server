#!/bin/bash -ex

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
cd ${DIR}

apt update
apt -y install wget ca-certificates

JRE_VERSION="${JRE_VERSION:-17.0.13+11}"
JRE_VERSION_ENC=$(echo ${JRE_VERSION} | sed 's/+/%2B/')
JRE_VERSION_FILE=$(echo ${JRE_VERSION} | sed 's/+/_/')

URL="https://github.com/adoptium/temurin17-binaries/releases/download/jdk-${JRE_VERSION_ENC}/OpenJDK17U-jre_x64_linux_hotspot_${JRE_VERSION_FILE}.tar.gz"

OUT=${DIR}/../build/snap/jre
rm -rf ${OUT}
mkdir -p ${OUT}

wget -q "${URL}" -O /tmp/jre.tar.gz
tar -xzf /tmp/jre.tar.gz -C ${OUT} --strip-components=1
rm /tmp/jre.tar.gz

ls -la ${OUT}
echo "java version:"
${OUT}/bin/java -version
echo "jre size:"
du -sh ${OUT}
