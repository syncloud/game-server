import { Page } from '@playwright/test'

/**
 * Drive the Authelia login form once and let the browser keep the
 * session cookie. Authelia redirects /any-protected-url to
 * https://auth.<domain>/?rd=<encoded>. The login form's exact element
 * ids/names depend on Authelia version — we try a few selectors.
 */
export async function loginViaAuthelia (
  page: Page,
  baseURL: string,
  username: string,
  password: string
) {
  await page.goto(baseURL)

  // The app no longer bounces unauthenticated visits via nginx; the SPA
  // mounts, fetches /api/v1/me, gets 401, and only THEN does
  // window.location.assign('/auth/login'), which the backend redirects to
  // Authelia. Wait for the URL to leave the app host before scanning for
  // the username field.
  try {
    await page.waitForURL((url) => {
      const h = new URL(url.toString()).host
      return h.startsWith('auth.')
    }, { timeout: 15_000 })
  } catch (_) {
    // already on auth host, or app didn't redirect — fall through and let
    // the selector loop produce a useful error.
  }

  const usernameSelectors = [
    'input[name="username"]',
    'input#username-textfield',
    'input[autocomplete="username"]',
    'input[type="text"]'
  ]
  const passwordSelectors = [
    'input[name="password"]',
    'input#password-textfield',
    'input[autocomplete="current-password"]',
    'input[type="password"]'
  ]
  const submitSelectors = [
    'button#sign-in-button',
    'button[type="submit"]',
    'button:has-text("Sign in")',
    'button:has-text("Login")'
  ]

  const found = async (selectors: string[]): Promise<string> => {
    for (const sel of selectors) {
      const el = page.locator(sel).first()
      try {
        await el.waitFor({ state: 'visible', timeout: 5_000 })
        return sel
      } catch (_) { /* try next */ }
    }
    const url = page.url()
    const title = await page.title().catch(() => '?')
    throw new Error(`no selector matched on ${url} (title="${title}"): ${selectors.join(', ')}`)
  }

  const userSel = await found(usernameSelectors)
  await page.fill(userSel, username)
  const passSel = await found(passwordSelectors)
  await page.fill(passSel, password)
  const submitSel = await found(submitSelectors)
  await Promise.all([
    page.waitForURL((url) => !url.toString().includes('/?rd='), { timeout: 30_000 }),
    page.click(submitSel)
  ])
  await page.waitForSelector('[data-testid="brand"]', { timeout: 15_000 })
}
