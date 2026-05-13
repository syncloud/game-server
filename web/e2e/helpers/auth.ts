import { Page } from '@playwright/test'

/**
 * Drive the Authelia login form once and let the browser keep the
 * session cookie. Authelia redirects /any-protected-url to
 * https://auth.<domain>/?rd=<encoded>, where the form has standard
 * input[name=username|password] fields.
 */
export async function loginViaAuthelia (
  page: Page,
  baseURL: string,
  username: string,
  password: string
) {
  await page.goto(baseURL)
  // We expect a redirect to authelia. Wait for the username field.
  await page.waitForSelector('input[name="username"]', { timeout: 15_000 })
  await page.fill('input[name="username"]', username)
  await page.fill('input[name="password"]', password)
  await Promise.all([
    page.waitForURL((url) => !url.toString().includes('/?rd='), { timeout: 30_000 }),
    page.click('button[type="submit"]')
  ])
  // We should now be back on the app domain. Wait for the brand to render.
  await page.waitForSelector('[data-testid="brand"]', { timeout: 15_000 })
}
