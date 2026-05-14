import { test, expect } from '@playwright/test'
import { shoot } from '../helpers/screenshot'

test('install dialog opens, creates a server, lands on server detail', async ({ page }, info) => {
  // Unique name per attempt — retries+UNIQUE constraint would otherwise
  // poison the retry when the first attempt half-succeeds.
  const name = `e2e-tw-${Date.now()}`

  await page.goto('/#/catalog')
  // On mobile the fixed bottom-bar overlaps cards near the viewport
  // bottom. scroll-margin-bottom on .card handles the natural anchor
  // scroll; this explicit scroll covers the case where the card is
  // partly above the viewport too.
  const teeCard = page.getByTestId('game-teeworlds')
  await teeCard.scrollIntoViewIfNeeded()
  await teeCard.getByTestId('install-btn').click()
  await expect(page.getByTestId('install-dialog')).toBeVisible()
  await page.getByTestId('dialog-name').fill(name)
  // Some egg variants don't have a SERVER_PORT variable so the catalog
  // defaultPort comes through as 0 — fill explicitly to enable submit.
  await page.getByTestId('dialog-port').fill('8313')
  await page.getByTestId('dialog-submit').click()
  // Hash-router doesn't fire 'load' on navigation; assert the detail
  // element directly instead of waitForURL.
  await expect(page.getByTestId('detail-name')).toHaveText(name)
  await expect(page.getByTestId('detail-status')).toBeVisible()
  await shoot(page, info, 'server-detail')

  await page.getByTestId('subtab-logs').click()
  await expect(page.getByTestId('detail-logs')).toBeVisible()

  await page.getByTestId('subtab-query').click()
  await expect(page.getByTestId('detail-query')).toBeVisible()

  // confirm() prompt — auto-accept
  page.once('dialog', d => d.accept())
  await page.getByTestId('action-delete').click()
  await expect(page.getByTestId('servers-empty')).toBeVisible()
})
