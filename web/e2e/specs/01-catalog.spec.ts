import { test, expect } from '@playwright/test'
import { shoot } from '../helpers/screenshot'

test('catalog renders with brand and game grid', async ({ page }, info) => {
  await page.goto('/#/catalog')
  await expect(page.getByTestId('brand')).toBeVisible()
  await expect(page.getByTestId('game-grid')).toBeVisible()
  // teeworlds is our pinned-fixture egg; cs2 is hand-curated steam; hlds-cs is CI fixture
  await expect(page.getByTestId('game-teeworlds')).toBeVisible()
  await expect(page.getByTestId('game-cs2')).toBeVisible()
  await shoot(page, info, 'catalog')
})

test('search filters games', async ({ page }, info) => {
  await page.goto('/#/catalog')
  await page.getByTestId('search').fill('teeworlds')
  await expect(page.getByTestId('game-teeworlds')).toBeVisible()
  await expect(page.getByTestId('game-cs2')).not.toBeVisible()
  await shoot(page, info, 'search')
})

test('tier filter pills', async ({ page }, info) => {
  await page.goto('/#/catalog')
  await expect(page.getByTestId('tier-all')).toBeVisible()
  await expect(page.getByTestId('tier-supported')).toBeVisible()
  await expect(page.getByTestId('tier-compatible')).toBeVisible()
  await expect(page.getByTestId('tier-experimental')).toBeVisible()
  // pick supported — hlds-cs is hand-curated as supported.
  await page.getByTestId('tier-supported').click()
  await expect(page.getByTestId('game-hlds-cs')).toBeVisible()
  await shoot(page, info, 'tier-supported')
  // pick compatible — teeworlds (from the egg catalog) lands here.
  await page.getByTestId('tier-compatible').click()
  await expect(page.getByTestId('game-teeworlds')).toBeVisible()
})

test('servers tab empty by default', async ({ page }, info) => {
  await page.goto('/#/servers')
  await expect(page.getByTestId('servers-empty')).toBeVisible()
  await shoot(page, info, 'servers-empty')
})

test('settings page shows catalog sources and Steam form', async ({ page }, info) => {
  await page.goto('/#/settings')
  await expect(page.getByTestId('settings-steam')).toBeVisible()
  await expect(page.getByTestId('settings-sources')).toBeVisible()
  await expect(page.getByTestId('steam-username')).toBeVisible()
  await expect(page.getByTestId('steam-password')).toBeVisible()
  await shoot(page, info, 'settings')
})
