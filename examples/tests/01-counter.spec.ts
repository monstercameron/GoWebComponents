import { test, expect } from '@playwright/test';

test.describe('01-Counter', () => {
  test('should increment counter', async ({ page }) => {
    await page.goto('/01-counter/counter.html');
    
    // Wait for WASM to load and render - look for the initial count "0"
    // We use a more specific locator to avoid matching other "0"s if any
    await expect(page.getByText('0', { exact: true })).toBeVisible();
    await expect(page.getByText('Current Count')).toBeVisible();
    
    // Click increment (+)
    await page.getByRole('button').nth(2).click();
    
    // Verify count increased
    await expect(page.getByText('1', { exact: true })).toBeVisible();
    
    // Click again
    await page.getByRole('button').nth(2).click();
    await expect(page.getByText('2', { exact: true })).toBeVisible();

    // Click decrement (-)
    await page.getByRole('button').nth(0).click();
    await expect(page.getByText('1', { exact: true })).toBeVisible();

    // Click Reset
    await page.getByRole('button', { name: 'Reset' }).click();
    await expect(page.getByText('0', { exact: true })).toBeVisible();
  });
});
