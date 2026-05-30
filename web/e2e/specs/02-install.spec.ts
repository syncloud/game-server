import { test, expect } from '@playwright/test'
import { shoot } from '../helpers/screenshot'

test.use({ video: 'on' })

test('install keys a server to the game, shows connect address, one-per-game', async ({ page }, info) => {
  await page.goto('/#/catalog')
  const teeCard = page.getByTestId('game-teeworlds')
  await teeCard.scrollIntoViewIfNeeded()
  await teeCard.getByTestId('install-btn').click()
  await expect(page.getByTestId('install-dialog')).toBeVisible()
  await page.getByTestId('dialog-port').fill('8313')
  await page.getByTestId('dialog-submit').click()

  // lands on detail, titled by the game name (no per-instance name field anymore)
  await expect(page.getByTestId('detail-name')).toContainText('Teeworlds')
  await expect(page.getByTestId('detail-status')).toBeVisible()
  await expect(page.getByTestId('detail-connect')).toContainText(':8313')
  await shoot(page, info, 'server-detail')

  await page.getByTestId('subtab-logs').click()
  await expect(page.getByTestId('detail-logs')).toBeVisible()
  await page.getByTestId('subtab-query').click()
  await expect(page.getByTestId('detail-query')).toBeVisible()

  // servers list shows the connect address + copy affordance
  await page.goto('/#/servers')
  await expect(page.getByTestId('server-connect')).toContainText(':8313')
  await expect(page.getByTestId('server-copy')).toBeVisible()
  await shoot(page, info, 'servers-list')

  // one server per game: the catalog now offers "Installed", not a second install
  await page.goto('/#/catalog')
  const teeAgain = page.getByTestId('game-teeworlds')
  await teeAgain.scrollIntoViewIfNeeded()
  await expect(teeAgain.getByTestId('open-btn')).toBeVisible()
  await expect(teeAgain.getByTestId('install-btn')).toHaveCount(0)

  // open the installed server and delete it
  await teeAgain.getByTestId('open-btn').click()
  await expect(page.getByTestId('detail-name')).toContainText('Teeworlds')
  await page.getByTestId('action-delete').click()
  await expect(page.getByTestId('confirm-dialog')).toBeVisible()
  await page.getByTestId('confirm-ok').click()
  await expect(page.getByTestId('servers-empty')).toBeVisible()

  // catalog offers install again after removal
  await page.goto('/#/catalog')
  await page.getByTestId('game-teeworlds').scrollIntoViewIfNeeded()
  await expect(page.getByTestId('game-teeworlds').getByTestId('install-btn')).toBeVisible()
})
