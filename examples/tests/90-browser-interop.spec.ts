import { expect, test } from '@playwright/test';

test.describe('90-Browser Interop', () => {
  test('imports a lazy module on demand and renders its exports', async ({ page }) => {
    await page.goto('/90-browser-interop/browser-interop.html');

    await expect(page.getByRole('heading', { name: 'Browser Interop', exact: true })).toBeVisible({ timeout: 15000 });
    await expect(page.getByRole('button', { name: 'Load lazy module', exact: true })).toBeVisible();

    await page.getByRole('button', { name: 'Load lazy module', exact: true }).click();

    await expect(page.getByText('Imported interop-demo-module through interop.ImportModule(...).', { exact: true })).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('lazy-module:Ship the browser bridge. | default-module:Ship the browser bridge.', { exact: true })).toBeVisible();
  });
});
