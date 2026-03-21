import { expect, test } from '@playwright/test';

test.describe('99-Plugin Host', () => {
  test('registers plugins and evaluates contributed hooks', async ({ page }) => {
    await page.goto('/99-plugin-host/plugin-host.html');

    await expect(page.getByRole('heading', { name: 'Plugin Host', exact: true })).toBeVisible({ timeout: 15000 });
    await expect(page.locator('#plugin-host-summary')).toContainText('explicit capabilities');
    await expect(page.locator('#plugin-head-preview')).toContainText('data-gwc-router-managed');

    await page.getByRole('button', { name: 'Evaluate admin route', exact: true }).click();
    await expect(page.locator('#plugin-route-result')).toContainText('redirect to /signin');

    await page.getByRole('button', { name: 'Decorate inventory request', exact: true }).click();
    await expect(page.locator('#plugin-cache-key')).toHaveText('plugins:inventory:list');

    await page.getByRole('button', { name: 'Validate invalid order', exact: true }).click();
    await expect(page.locator('#plugin-validation-result')).toContainText('quantity: Quantity must be greater than zero.');

    await page.getByRole('button', { name: 'Validate ready order', exact: true }).click();
    await expect(page.locator('#plugin-validation-result')).toContainText('No validation issues.');
  });
});