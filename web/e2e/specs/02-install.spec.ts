import { test, expect } from '@playwright/test'
import { shoot } from '../helpers/screenshot'

test('install dialog opens, creates a server, lands on server detail', async ({ page }, info) => {
  await page.goto('/#/catalog')
  await page.getByTestId('game-teeworlds').getByTestId('install-btn').click()
  await expect(page.getByTestId('install-dialog')).toBeVisible()
  await page.getByTestId('dialog-name').fill('e2e-tw')
  await page.getByTestId('dialog-submit').click()
  // Phase 13 routes to server detail page after create
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
