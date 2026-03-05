import type { APIRequestContext } from '@playwright/test'
import { expect, test } from '@playwright/test'

async function getFirstServerUuid(
  request: APIRequestContext
): Promise<string | null> {
  const response = await request.get('/api/servers')
  if (!response.ok()) {
    return null
  }

  const payload = (await response.json()) as unknown
  if (!Array.isArray(payload) || payload.length === 0) {
    return null
  }

  const firstWithUuid = payload.find((item) => {
    if (!item || typeof item !== 'object') {
      return false
    }

    const candidate = item as { uuid?: unknown }
    return typeof candidate.uuid === 'string' && candidate.uuid.length > 0
  }) as { uuid: string } | undefined

  return firstWithUuid?.uuid ?? null
}

test('strona metryk: przełączanie live/history działa', async ({
  page,
  request,
}) => {
  const uuid = await getFirstServerUuid(request)
  if (!uuid) {
    test.skip()
    return
  }

  await page.goto(`/servers/${uuid}/metrics`)
  await expect(page).toHaveURL(new RegExp(`/servers/${uuid}/metrics`))

  const rangeTrigger = page.getByTestId('metrics-range-trigger')
  await expect(rangeTrigger).toContainText('1 minuta')
  await expect(page.getByTestId('metrics-live-status')).toBeVisible()

  await rangeTrigger.click()
  await page.getByRole('option', { name: /^5 minut$/ }).click()

  await expect(page.getByTestId('metrics-stat-min')).toBeVisible()
  await expect(page.getByTestId('metrics-stat-avg')).toBeVisible()
  await expect(page.getByTestId('metrics-stat-max')).toBeVisible()
  await expect(page.getByTestId('metrics-live-status')).toHaveCount(0)

  await page.getByTestId('metrics-stat-max').click()
  await expect(page.getByTestId('metrics-stat-max')).toHaveClass(/bg-zinc-700/)

  await rangeTrigger.click()
  await page.getByRole('option', { name: /^1 minuta \(Na żywo\)$/ }).click()

  await expect(page.getByTestId('metrics-stat-avg')).toHaveCount(0)
  await expect(page.getByTestId('metrics-live-status')).toBeVisible()
})

test('strona kontenerów: tryb zaznaczania wielu elementów działa', async ({
  page,
  request,
}) => {
  const uuid = await getFirstServerUuid(request)
  if (!uuid) {
    test.skip()
    return
  }

  await page.goto(`/servers/${uuid}/containers`)
  await expect(page).toHaveURL(new RegExp(`/servers/${uuid}/containers`))

  const emptyState = page.getByText('Brak kontenerów')
  if (await emptyState.isVisible().catch(() => false)) {
    await expect(emptyState).toBeVisible()
    return
  }

  await expect(page.getByRole('columnheader', { name: 'Nazwa' })).toBeVisible()

  const bulkToggle = page.getByTestId('containers-bulk-toggle')
  await bulkToggle.click()

  await expect(page.getByText(/Zaznaczono:\s*\d+/)).toBeVisible()
  await page.getByRole('checkbox', { name: /^Zaznacz wszystkie$/ }).click()
  await expect(page.getByText(/Zaznaczono:\s*\d+/)).toBeVisible()

  await page.getByRole('button', { name: 'Zakończ zaznaczanie' }).click()
  await expect(
    page.getByRole('button', { name: 'Zaznacz wiele' })
  ).toBeVisible()
})

test('ustawienia integracji: pola są renderowane zgodnie z auth_type i requires_base_url', async ({
  page,
}) => {
  await page.route('**/api/services', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([
        {
          key: 'adguardhome',
          display_name: 'AdGuard Home',
          icon: '/icons/hetzner-h.svg',
          enabled: true,
          requires_base_url: true,
          auth_type: 'basic_auth',
          endpoints: ['/services/adguardhome/stats'],
        },
        {
          key: 'tailscale',
          display_name: 'Tailscale',
          icon: '/icons/tailscale.svg',
          enabled: false,
          requires_base_url: false,
          auth_type: 'token',
          fixed_base_url: 'https://api.tailscale.com/api/v2',
          endpoints: ['/services/tailscale/stats'],
        },
      ]),
    })
  })

  await page.goto('/settings/services')
  await expect(
    page.getByRole('heading', { name: 'Integracje usług' })
  ).toBeVisible()

  await expect(page.getByTestId('service-card-adguardhome')).toBeVisible()
  await expect(page.getByTestId('service-enabled-adguardhome')).toBeVisible()
  await expect(page.getByTestId('service-base-url-adguardhome')).toBeVisible()
  await expect(page.getByTestId('service-username-adguardhome')).toBeVisible()
  await expect(page.getByTestId('service-password-adguardhome')).toBeVisible()
  await expect(page.getByTestId('service-test-adguardhome')).toBeVisible()
  await expect(page.getByTestId('service-save-adguardhome')).toBeVisible()
  await expect(page.getByAltText('AdGuard Home icon')).toBeVisible()

  await expect(page.getByTestId('service-card-tailscale')).toBeVisible()
  await expect(page.getByAltText('Tailscale icon')).toBeVisible()
  await expect(page.getByTestId('service-token-tailscale')).toHaveCount(0)

  await page.getByTestId('service-enabled-tailscale').click()
  await expect(page.getByTestId('service-fixed-url-tailscale')).toBeVisible()
  await expect(page.getByTestId('service-token-tailscale')).toBeVisible()
  await expect(page.getByTestId('service-test-tailscale')).toBeVisible()
  await expect(page.getByTestId('service-save-tailscale')).toBeVisible()
})
