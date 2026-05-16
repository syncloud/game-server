import { test, expect } from '@playwright/test'
import { shoot } from '../helpers/screenshot'

test.use({ video: 'on' })

test('install dialog opens, creates a server, lands on server detail', async ({ page }, info) => {
  const name = `e2e-tw-${Date.now()}`

  await page.goto('/#/catalog')
  const teeCard = page.getByTestId('game-teeworlds')
  await teeCard.scrollIntoViewIfNeeded()
  await teeCard.getByTestId('install-btn').click()
  await expect(page.getByTestId('install-dialog')).toBeVisible()
  await page.getByTestId('dialog-name').fill(name)
  await page.getByTestId('dialog-port').fill('8313')
  await page.getByTestId('dialog-submit').click()
  await expect(page.getByTestId('detail-name')).toHaveText(name)
  await expect(page.getByTestId('detail-status')).toBeVisible()
  await shoot(page, info, 'server-detail')

  await page.getByTestId('subtab-logs').click()
  await expect(page.getByTestId('detail-logs')).toBeVisible()

  await page.getByTestId('subtab-query').click()
  await expect(page.getByTestId('detail-query')).toBeVisible()

  await page.getByTestId('action-delete').click()
  await expect(page.getByTestId('confirm-dialog')).toBeVisible()
  await page.getByTestId('confirm-ok').click()
  await expect(page.getByTestId('servers-empty')).toBeVisible()
})
