#!/bin/bash -ex

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
cd ${DIR}

apt update
apt -y install wget ca-certificates

# Pinned upstream catalog sources. Bump these constants to roll the catalog
# forward; reproducible until then. parkervcp/eggs is archived (frozen),
# pelican-eggs/games is actively maintained.
PARKERVCP_SHA="${PARKERVCP_SHA:-fcfd5a3549769ade15127a7577d6d3c397e83b05}"
PELICAN_GAMES_SHA="${PELICAN_GAMES_SHA:-34331fce33c83df752d94e2a90b1e43ca6280f82}"

WORK=${DIR}/work
rm -rf ${WORK}
mkdir -p ${WORK}
cd ${WORK}

echo "fetching parkervcp/eggs @ ${PARKERVCP_SHA}"
wget -q "https://github.com/parkervcp/eggs/archive/${PARKERVCP_SHA}.tar.gz" -O parkervcp.tar.gz
tar xzf parkervcp.tar.gz
mv eggs-${PARKERVCP_SHA} parkervcp

echo "fetching pelican-eggs/games @ ${PELICAN_GAMES_SHA}"
wget -q "https://github.com/pelican-eggs/games/archive/${PELICAN_GAMES_SHA}.tar.gz" -O pelican.tar.gz
tar xzf pelican.tar.gz
mv games-${PELICAN_GAMES_SHA} pelican

cd ${DIR}/convert
CGO_ENABLED=0 go build -buildvcs=false -o ${WORK}/convert .

OUT=${DIR}/../backend/catalog/catalog.json
mkdir -p $(dirname ${OUT})
${WORK}/convert \
    --parkervcp ${WORK}/parkervcp/game_eggs \
    --pelican ${WORK}/pelican \
    --parkervcp-version ${PARKERVCP_SHA} \
    --pelican-version ${PELICAN_GAMES_SHA} \
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
