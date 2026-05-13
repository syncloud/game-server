import { test, expect } from '@playwright/test'
import { shoot } from '../helpers/screenshot'

test('install dialog opens, creates a server, lands on server detail', async ({ page }, info) => {
  await page.goto('/#/catalog')
  await page.getByTestId('game-teeworlds').getByTestId('install-btn').click()
  await expect(page.getByTestId('install-dialog')).toBeVisible()
  await page.getByTestId('dialog-name').fill('e2e-tw')
  // Some egg variants don't have a SERVER_PORT variable so the catalog
  // defaultPort comes through as 0 — fill explicitly to enable submit.
  await page.getByTestId('dialog-port').fill('8313')
  await page.getByTestId('dialog-submit').click()
  // Phase 13 routes to server detail page after create. Wait for URL
  // change first so a timeout here clearly says "navigation didn't happen"
  // rather than "element not found".
  await page.waitForURL(/#\/servers\/\d+$/, { timeout: 20_000 })
  await expect(page.getByTestId('detail-name')).toHaveText('e2e-tw')
  await expect(page.getByTestId('detail-status')).toBeVisible()
  await shoot(page, info, 'server-detail')

  await page.getByTestId('subtab-logs').click()
  await expect(page.getByTestId('detail-logs')).toBeVisible()

  await page.getByTestId('subtab-query').click()
  await expect(page.getByTestId('detail-query')).toBeVisible()

  await page.getByTestId('action-delete').click()
  await page.waitForURL(/#\/servers$/)
})
