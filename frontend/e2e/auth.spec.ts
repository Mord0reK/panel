import { test, expect } from '@playwright/test'

// Middleware blokuje niezalogowanych
test('niezalogowany użytkownik → redirect do /login', async ({ page }) => {
  await page.goto('/dashboard')
  await expect(page).toHaveURL(/\/login/)
})

// Strona główna → redirect do /dashboard → middleware → /login
test('strona główna → redirect do /login gdy brak sesji', async ({ page }) => {
  await page.goto('/')
  await expect(page).toHaveURL(/\/login/)
})

// /login wyświetla formularz logowania
test('strona /login wyświetla formularz', async ({ page }) => {
  await page.goto('/login')
  // Sprawdź że jesteśmy na /login (lub /setup jeśli setup_required)
  const url = page.url()
  if (url.includes('/setup')) {
    // Backend wymaga setupu — sprawdź formularz setup
    await expect(page.getByTestId('setup-username')).toBeVisible()
    await expect(page.getByTestId('setup-password')).toBeVisible()
    await expect(page.getByTestId('setup-submit')).toBeVisible()
  } else {
    await expect(page.getByTestId('login-username')).toBeVisible()
    await expect(page.getByTestId('login-password')).toBeVisible()
    await expect(page.getByTestId('login-submit')).toBeVisible()
  }
})

// Błędne dane logowania → komunikat błędu
test('błędne dane logowania → wyświetla błąd', async ({ page }) => {
  await page.goto('/login')

  // Jeśli setup_required, pomijamy ten test
  if (page.url().includes('/setup')) {
    test.skip()
    return
  }

  await page.getByTestId('login-username').fill('zly_uzytkownik')
  await page.getByTestId('login-password').fill('zle_haslo_9999')
  await page.getByTestId('login-submit').click()

  await expect(page.getByTestId('login-error')).toBeVisible({ timeout: 10_000 })
})

// Puste pola → HTML5 validation, nie wysyłamy
test('puste pola nie przechodzą walidacji HTML5', async ({ page }) => {
  await page.goto('/login')

  if (page.url().includes('/setup')) {
    test.skip()
    return
  }

  await page.getByTestId('login-submit').click()
  // Nie powinniśmy dostać błędu z API (formularz nie wysłany)
  await expect(page.getByTestId('login-error')).not.toBeVisible()
})

// Formularz setup — walidacja minimalnej długości
test('setup: hasło < 8 znaków → błąd walidacji', async ({ page }) => {
  await page.goto('/setup')

  if (page.url().includes('/login')) {
    test.skip()
    return
  }

  await page.getByTestId('setup-username').fill('admin')
  await page.getByTestId('setup-password').fill('short')
  await page.getByTestId('setup-submit').click()

  await expect(page.getByTestId('setup-error')).toBeVisible({ timeout: 5_000 })
})
