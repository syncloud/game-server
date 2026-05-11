#!/bin/bash -ex

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && cd .. && pwd )
cd ${DIR}/web/e2e

npm ci
npx playwright install --with-deps chromium
PLAYWRIGHT_DOMAIN=${PLAYWRIGHT_DOMAIN:-bookworm.com} npx playwright test --project="${1:-desktop}"
