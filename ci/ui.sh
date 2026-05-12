#!/bin/bash -ex

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && cd .. && pwd )
cd ${DIR}/web/e2e

npm ci
npx playwright install chromium
PLAYWRIGHT_DOMAIN=${PLAYWRIGHT_DOMAIN:-bookworm.com} \
PLAYWRIGHT_USER=${PLAYWRIGHT_USER:-syncloud} \
PLAYWRIGHT_PASSWORD=${PLAYWRIGHT_PASSWORD:-syncloud} \
npx playwright test --project="${1:-desktop}"
