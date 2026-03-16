import { test, expect } from '@playwright/test';

test.describe('16-Devtools', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/16-devtools/devtools.html');
    await expect(page.getByText('Inspect runtime state without the OMI shell')).toBeVisible();
  });

  test('shows profiling hotspots and diagnostics in the panel', async ({ page }) => {
    await expect(page.getByText('Standalone Devtools')).toBeVisible();
    await expect(page.getByText('Profiling', { exact: true })).toBeVisible();
    await expect(page.getByText('Hot branches', { exact: true })).toBeVisible();
    await expect(page.getByText(/missing key on one or more sibling elements/i)).toBeVisible({ timeout: 10000 });

    await page.getByRole('button', { name: 'Increment counter' }).click();
    await expect(page.getByText('Counter', { exact: true })).toBeVisible();

    await page.getByRole('button', { name: 'Hide devtools' }).click();
    await expect(page.getByRole('button', { name: 'Hide devtools' })).not.toBeVisible();
  });
});