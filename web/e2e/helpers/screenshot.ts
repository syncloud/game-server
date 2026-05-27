import { Page, TestInfo } from '@playwright/test'

export async function shoot (page: Page, info: TestInfo, name: string) {
  await page.screenshot({ path: info.outputPath(`${name}.png`) })
}
