#!/bin/bash -ex

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && cd .. && pwd )
cd ${DIR}/web/e2e

npm ci
# Default user/password mirror the syncloud-lib integration test fixture
# (created by the platform during activate_custom).
PLAYWRIGHT_DOMAIN=${PLAYWRIGHT_DOMAIN:-bookworm.com} \
PLAYWRIGHT_USER=${PLAYWRIGHT_USER:-user} \
PLAYWRIGHT_PASSWORD=${PLAYWRIGHT_PASSWORD:-Password1} \
npx playwright test --project="${1:-desktop}"
