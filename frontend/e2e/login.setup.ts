import fs from 'node:fs'
import path from 'node:path'
import { test as setup, expect } from '@playwright/test'

const authFile = path.join(__dirname, '../playwright/.auth/user.json')

setup('authenticate as admin', async ({ page }) => {
  fs.mkdirSync(path.dirname(authFile), { recursive: true })

  await page.goto('/login')

  if (page.url().includes('/setup')) {
    throw new Error('Panel wymaga setupu użytkownika. Załóż konto admin/admin1234 przed uruchomieniem testów e2e.')
  }

  await page.getByTestId('login-username').fill('admin')
  await page.getByTestId('login-password').fill('admin1234')
  await page.getByTestId('login-submit').click()

  await expect(page).toHaveURL(/\/dashboard/)
  await page.context().storageState({ path: authFile })
})
