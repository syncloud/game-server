#!/bin/bash -ex

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )

cd ${DIR}/aggregate
CGO_ENABLED=0 go build -buildvcs=false -o ${DIR}/aggregate.bin .

OUT=${DIR}/../backend/catalog/catalog.json
${DIR}/aggregate.bin --root ${DIR} --out ${OUT}
rm ${DIR}/aggregate.bin

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
"
