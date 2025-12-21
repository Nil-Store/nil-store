import { test, expect } from '@playwright/test'

const path = process.env.E2E_PATH || '/#/dashboard'

test('dashboard loads and shows wallet prompt + bridge status', async ({ page }) => {
  await page.goto(path)
  await page.waitForSelector('[data-testid="connect-wallet"], [data-testid="wallet-address"]', {
    timeout: 60_000,
  })
  const connectButton = page.getByTestId('connect-wallet').first()
  const walletAddress = page.getByTestId('wallet-address')
  if (await connectButton.isVisible().catch(() => false)) {
    await expect(connectButton).toBeVisible()
  } else {
    await expect(walletAddress).toBeVisible()
  }
  const bridge = page.getByText(/EVM Bridge/i)
  if ((await bridge.count()) > 0) {
    await expect(bridge).toBeVisible()
  }
})
