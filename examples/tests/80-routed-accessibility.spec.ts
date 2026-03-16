import { expect, test } from '@playwright/test';

test.describe('80-Routed Accessibility', () => {
  test('announces route changes and moves focus to the active page heading', async ({ page }) => {
    await page.goto('/80-routed-accessibility/routed-accessibility.html');

    await expect(page.getByRole('heading', { name: 'Overview accessibility' })).toBeVisible();
    await expect(page.locator('[aria-live="polite"]').first()).toContainText('Loaded Overview accessibility');

    await page.getByRole('link', { name: 'Settings' }).click();
    await expect(page.getByRole('heading', { name: 'Settings accessibility' })).toBeVisible();
    await expect(page).toHaveURL(/routed-accessibility\.html#\/accessibility\/settings$/);
    await expect(page.locator('[aria-live="polite"]').first()).toContainText('Loaded Settings accessibility');
    await expect.poll(async () => page.evaluate(() => document.activeElement?.id)).toBe('route-page-heading');

    await page.getByRole('link', { name: 'Reports' }).click();
    await expect(page.getByRole('heading', { name: 'Reports accessibility' })).toBeVisible();
    await expect(page.locator('[aria-live="polite"]').first()).toContainText('Loaded Reports accessibility');
  });
});