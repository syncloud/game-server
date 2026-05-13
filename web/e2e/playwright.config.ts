import { defineConfig, devices } from '@playwright/test'

const domain = process.env.PLAYWRIGHT_DOMAIN || 'bookworm.com'
const baseURL = `https://games.${domain}`
const storageState = '.auth/state.json'

export default defineConfig({
  testDir: './specs',
  timeout: 60_000,
  expect: { timeout: 10_000 },
  workers: 1,
  retries: process.env.CI ? 1 : 0,
  reporter: [['html', { open: 'never' }]],
  globalSetup: './global-setup.ts',
  use: {
    baseURL,
    ignoreHTTPSErrors: true,
    storageState,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure'
  },
  projects: [
    { name: 'desktop', use: { ...devices['Desktop Chrome'], baseURL, ignoreHTTPSErrors: true, storageState } },
    { name: 'mobile', use: { ...devices['Pixel 7'], baseURL, ignoreHTTPSErrors: true, storageState } }
  ]
})
