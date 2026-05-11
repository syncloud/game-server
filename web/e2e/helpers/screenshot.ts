import { Page, TestInfo } from '@playwright/test'

export async function shoot (page: Page, info: TestInfo, name: string) {
  const buf = await page.screenshot({ fullPage: true })
  await info.attach(name, { body: buf, contentType: 'image/png' })
}
