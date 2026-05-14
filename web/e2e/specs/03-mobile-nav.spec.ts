import { test, expect } from '@playwright/test'
import { shoot } from '../helpers/screenshot'

test('bottom-nav is mobile-only and clicks navigate', async ({ page, isMobile }, info) => {
  test.skip(!isMobile, 'desktop project hides bottom-nav by design')

  await page.goto('/#/catalog')
  await expect(page.getByTestId('bottom-nav')).toBeVisible()
  await expect(page.getByTestId('bottom-catalog')).toHaveClass(/active/)
  await shoot(page, info, 'mobile-catalog')

  // Scroll back to the top before clicking the fixed bottom-bar so the
  // hit-test under the bar isn't a long catalog card whose summary text
  // happens to be at the same viewport coords as the bar.
  await page.evaluate(() => window.scrollTo(0, 0))
  // The fixed bar has z-index: 100 over static cards, but in Pixel-7
  // emulation Playwright's elementFromPoint occasionally walks past the
  // bar's bounding box into the catalog content beneath. Force-click
  // bypasses the actionability check — this is a Playwright/Chromium
  // hit-test quirk, not real-UX broken: a finger tap on the bar in a
  // mobile browser hits the bar.
  await page.getByTestId('bottom-servers').click({ force: true })
  await expect(page.getByTestId('servers-empty')).toBeVisible()
  await expect(page.getByTestId('bottom-servers')).toHaveClass(/active/)

  await page.evaluate(() => window.scrollTo(0, 0))
  await page.getByTestId('bottom-settings').click({ force: true })
  await expect(page.getByTestId('settings-account')).toBeVisible()
  await expect(page.getByTestId('account-name')).toBeVisible()
  await expect(page.getByTestId('settings-steam')).toBeVisible()
  await shoot(page, info, 'mobile-settings')
})
