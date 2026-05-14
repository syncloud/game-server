import { test, expect } from '@playwright/test'
import { shoot } from '../helpers/screenshot'

test('bottom-nav is mobile-only and clicks navigate', async ({ page, isMobile }, info) => {
  test.skip(!isMobile, 'desktop project hides bottom-nav by design')

  await page.goto('/#/catalog')
  await expect(page.getByTestId('bottom-nav')).toBeVisible()
  await expect(page.getByTestId('bottom-catalog')).toHaveClass(/active/)
  await shoot(page, info, 'mobile-catalog')

  await page.getByTestId('bottom-servers').click()
  await expect(page.getByTestId('servers-empty')).toBeVisible()
  await expect(page.getByTestId('bottom-servers')).toHaveClass(/active/)

  await page.getByTestId('bottom-settings').click()
  await expect(page.getByTestId('settings-account')).toBeVisible()
  await expect(page.getByTestId('account-name')).toBeVisible()
  await expect(page.getByTestId('settings-steam')).toBeVisible()
  await shoot(page, info, 'mobile-settings')
})
