import { test, expect } from '@playwright/test'
import { shoot } from '../helpers/screenshot'

test('install dialog opens and creates a server', async ({ page }, info) => {
  await page.goto('/')
  await page.getByTestId('game-teeworlds').getByTestId('install-btn').click()
  await expect(page.getByTestId('install-dialog')).toBeVisible()
  await page.getByTestId('dialog-name').fill('e2e-tw')
  await page.getByTestId('dialog-submit').click()
  await expect(page.getByTestId('install-dialog')).not.toBeVisible()
  await expect(page.locator('[data-testid^="server-"]')).toHaveCount(1)
  await shoot(page, info, 'server-created')

  await page.locator('[data-testid^="server-"]').getByTestId('server-delete').click()
  await expect(page.getByTestId('servers-empty')).toBeVisible()
})
