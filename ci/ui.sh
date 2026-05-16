#!/bin/bash -ex

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && cd .. && pwd )
cd ${DIR}/web/e2e

npm ci
PROJECT="${1:-desktop}"
set +e
PLAYWRIGHT_DOMAIN=${PLAYWRIGHT_DOMAIN:-bookworm.com} \
PLAYWRIGHT_USER=${PLAYWRIGHT_USER:-user} \
PLAYWRIGHT_PASSWORD=${PLAYWRIGHT_PASSWORD:-Password1} \
npx playwright test --project="${PROJECT}"
EXIT=$?
set -e

ART=${DIR}/artifact
SHOTS=${ART}/screenshots-${PROJECT}
VIDEOS=${ART}/videos-${PROJECT}
mkdir -p ${SHOTS} ${VIDEOS}

if [ -d test-results ]; then
    for d in test-results/*/; do
        name=$(basename "$d")
        i=0
        for img in "$d"*.png; do
            [ -f "$img" ] || continue
            suffix=$([ $i -eq 0 ] && echo "" || echo "-$i")
            cp "$img" "${SHOTS}/${name}${suffix}.png"
            i=$((i+1))
        done
        [ -f "${d}video.webm" ] && cp "${d}video.webm" "${VIDEOS}/${name}.webm"
    done
fi

exit ${EXIT}
