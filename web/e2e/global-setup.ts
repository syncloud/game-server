import { chromium, FullConfig } from '@playwright/test'
import { loginViaAuthelia } from './helpers/auth'

export default async function globalSetup (config: FullConfig) {
  const { baseURL, storageState } = config.projects[0].use as any
  const username = process.env.PLAYWRIGHT_USER || 'user'
  const password = process.env.PLAYWRIGHT_PASSWORD || 'Password1'

  const browser = await chromium.launch()
  const context = await browser.newContext({ ignoreHTTPSErrors: true })
  const page = await context.newPage()
  try {
    await loginViaAuthelia(page, baseURL, username, password)
    await context.storageState({ path: storageState })
  } finally {
    await browser.close()
  }
}
