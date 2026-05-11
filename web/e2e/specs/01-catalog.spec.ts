import { test, expect } from '@playwright/test'
import { shoot } from '../helpers/screenshot'

test('catalog renders with brand and game grid', async ({ page }, info) => {
  await page.goto('/')
  await expect(page.getByTestId('brand')).toBeVisible()
  await expect(page.getByTestId('game-grid')).toBeVisible()
  await expect(page.getByTestId('game-teeworlds')).toBeVisible()
  await expect(page.getByTestId('game-cs2')).toBeVisible()
  await shoot(page, info, 'catalog')
})

test('search filters games', async ({ page }, info) => {
  await page.goto('/')
  await page.getByTestId('search').fill('teeworlds')
  await expect(page.getByTestId('game-teeworlds')).toBeVisible()
  await expect(page.getByTestId('game-cs2')).not.toBeVisible()
  await shoot(page, info, 'search')
})

test('servers tab is empty by default', async ({ page }, info) => {
  await page.goto('/')
  await page.getByTestId('tab-servers').click()
  await expect(page.getByTestId('servers-empty')).toBeVisible()
  await shoot(page, info, 'servers-empty')
})
