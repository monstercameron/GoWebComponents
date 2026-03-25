import { expect, test, type Page } from '@playwright/test';

test.describe('100-AI-Chat-Wizard manual smoke', () => {
  test.skip(!process.env.PLAYWRIGHT_MANUAL_SMOKE, 'Manual smoke suite is opt-in.');

  async function loginAndLoad(page: Page): Promise<void> {
    await page.goto('/login');
    await expect(page.getByRole('heading', { name: 'Welcome back' })).toBeVisible({ timeout: 60_000 });
    await page.getByLabel('Email').fill('demo@example.com');
    await page.getByLabel('Password').fill('password123');
    await page.getByRole('button', { name: 'Log in' }).click();
    await expect(page.getByRole('button', { name: 'New chat' })).toBeVisible({ timeout: 60_000 });
  }

  test('manual smoke: login, inspect seeded history, and open settings', async ({ page }) => {
    await loginAndLoad(page);

    await expect(page.getByText('Explore the Go WASM UI experiment')).toBeVisible();
    await expect(page.getByRole('button', { name: /WebAssembly and Go/i }).first()).toBeVisible();
    await expect(page.getByRole('button', { name: /Golang Goroutines Explained/i }).first()).toBeVisible();

    await page.getByRole('button', { name: /Golang Goroutines Explained/i }).first().click();
    await expect(page.getByText('How do goroutines work in Go?')).toBeVisible();
    await expect(page.getByText(/Goroutines are lightweight threads/i)).toBeVisible();

    await page.getByRole('button', { name: /Edit settings/i }).click();
    await expect(page.locator('#name-input')).toBeVisible();
    await expect(page.locator('[data-current-locale]')).toBeVisible();
  });
});