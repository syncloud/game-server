#!/bin/bash -ex

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && cd .. && pwd )
cd ${DIR}/web/e2e

npm ci
# Default user/password mirror the syncloud-lib integration test fixture
# (created by the platform during activate_custom).
PROJECT="${1:-desktop}"
set +e
PLAYWRIGHT_DOMAIN=${PLAYWRIGHT_DOMAIN:-bookworm.com} \
PLAYWRIGHT_USER=${PLAYWRIGHT_USER:-user} \
PLAYWRIGHT_PASSWORD=${PLAYWRIGHT_PASSWORD:-Password1} \
npx playwright test --project="${PROJECT}"
EXIT=$?
set -e

# Upload report + traces + screenshots alongside the snap artifact so we
# can debug failures from the artifact server without re-running CI.
ART=${DIR}/artifact
mkdir -p ${ART}
[ -d playwright-report ] && cp -r playwright-report ${ART}/playwright-report-${PROJECT}
[ -d test-results ] && cp -r test-results ${ART}/test-results-${PROJECT}

exit ${EXIT}
