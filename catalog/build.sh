#!/bin/bash -ex

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
cd ${DIR}

apt update
apt -y install wget ca-certificates

PARKERVCP_SHA="${PARKERVCP_SHA:-fcfd5a3549769ade15127a7577d6d3c397e83b05}"
PELICAN_GAMES_SHA="${PELICAN_GAMES_SHA:-34331fce33c83df752d94e2a90b1e43ca6280f82}"
LINUXGSM_SHA="${LINUXGSM_SHA:-d05992d7d2deb88ed0a0c9df5ffc423995947d11}"  # v26.1.0

WORK=${DIR}/work
rm -rf ${WORK}
mkdir -p ${WORK}
cd ${WORK}

echo "fetching parkervcp/eggs @ ${PARKERVCP_SHA}"
mkdir -p parkervcp
wget -q "https://github.com/parkervcp/eggs/archive/${PARKERVCP_SHA}.tar.gz" -O parkervcp.tar.gz
tar -xzf parkervcp.tar.gz -C parkervcp --strip-components=1

echo "fetching pelican-eggs/games @ ${PELICAN_GAMES_SHA}"
mkdir -p pelican
wget -q "https://github.com/pelican-eggs/games/archive/${PELICAN_GAMES_SHA}.tar.gz" -O pelican.tar.gz
tar -xzf pelican.tar.gz -C pelican --strip-components=1

echo "fetching GameServerManagers/LinuxGSM @ ${LINUXGSM_SHA}"
mkdir -p linuxgsm
wget -q "https://github.com/GameServerManagers/LinuxGSM/archive/${LINUXGSM_SHA}.tar.gz" -O linuxgsm.tar.gz
tar -xzf linuxgsm.tar.gz -C linuxgsm --strip-components=1

cd ${DIR}/convert
CGO_ENABLED=0 go build -buildvcs=false -o ${WORK}/convert .

OUT=${DIR}/../backend/catalog/catalog.json
mkdir -p $(dirname ${OUT})
${WORK}/convert \
    --parkervcp ${WORK}/parkervcp/game_eggs \
    --pelican ${WORK}/pelican \
    --linuxgsm ${WORK}/linuxgsm \
    --parkervcp-version ${PARKERVCP_SHA} \
    --pelican-version ${PELICAN_GAMES_SHA} \
    --linuxgsm-version ${LINUXGSM_SHA} \
    --out ${OUT}

echo "catalog stats:"
wc -c ${OUT}
python3 -c "
import json
with open('${OUT}') as f: c=json.load(f)
tiers={}
for g in c['games']:
    tiers[g['tier']] = tiers.get(g['tier'], 0) + 1
print('total:', len(c['games']))
print('by tier:', tiers)
print('versions:', c['sources'])
"
