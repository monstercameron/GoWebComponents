import { test, expect } from '@playwright/test';

test.describe('18-SSR Server Routing', () => {
  test('serves request-time SSR HTML on direct route loads', async ({ page }) => {
    const bootstrapResponsePromise = page.waitForResponse(resp => resp.url().includes('/_gwc/bootstrap') && resp.status() === 200);
    await page.goto('/docs/routing?tab=loader');

    await expect(page.getByText('Server SSR Demo', { exact: true })).toBeVisible();
    await expect(page.getByText('Advanced routing over real URLs', { exact: true })).toBeVisible();
    await expect(page.getByText('Current tab: loader', { exact: true })).toBeVisible();

    const bootstrapResponse = await bootstrapResponsePromise;
    await expect(page).toHaveURL(/\/docs\/routing\?tab=loader$/);
    expect(bootstrapResponse.url()).toContain('path=%2Fdocs%2Frouting');
  });

  test('follows protected and redirect flows over real URLs', async ({ page }) => {
    await page.goto('/secure');
    await expect(page).toHaveURL(/\/signin\?from=secure$/);
    await expect(page.getByText('Protected routes can redirect before rendering', { exact: true })).toBeVisible();

    await page.getByRole('link', { name: 'Grant access' }).click();
    await expect(page).toHaveURL(/\/secure\?auth=true&role=maintainer$/);
    await expect(page.getByText('Guarded content unlocked', { exact: true })).toBeVisible();

    await page.goto('/legacy');
    await expect(page).toHaveURL(/\/docs\/routing\?tab=loader$/);
    await expect(page.getByText('Advanced routing over real URLs', { exact: true })).toBeVisible();
  });
});