import { test, expect } from '@playwright/test';

test.describe('17-SSR Routing', () => {
  test('renders the SSR shell without JavaScript', async ({ browser }) => {
    const context = await browser.newContext({ javaScriptEnabled: false });
    const page = await context.newPage();

    await page.goto('/17-ssr-routing/ssr-routing.html');
    await expect(page.getByText('SSR Routing Demo', { exact: true })).toBeVisible();
    await expect(page.getByText('SSR transport and hydration', { exact: true })).toBeVisible();
    expect(await page.content()).toContain('bootstrap.json');

    await context.close();
  });

  test('hydrates and drives advanced routing flows', async ({ page }) => {
    await page.goto('/17-ssr-routing/ssr-routing.html');

    await expect(page.getByText('SSR transport and hydration', { exact: true })).toBeVisible();
    await expect(page.getByText('json-sidecar', { exact: true })).toBeVisible();

    await page.getByRole('link', { name: 'Legacy Redirect' }).click();
    await expect(page.getByText('Advanced route loaders and redirects', { exact: true })).toBeVisible({ timeout: 5000 });
    await expect(page).toHaveURL(/ssr-routing\.html#\/docs\/routing\?tab=loader$/);

    await page.getByRole('button', { name: 'Revalidate' }).click();
    await expect(page.getByText('Revision')).toBeVisible();

    await page.getByRole('link', { name: 'Search' }).click();
    await expect(page.getByText('Query-driven loader results', { exact: true })).toBeVisible();
    await page.getByRole('button', { name: 'Filter routing' }).click();
    await expect(page.getByText('Current query: routing', { exact: true })).toBeVisible({ timeout: 5000 });

    await page.getByRole('link', { name: 'Protected' }).click();
    await expect(page.getByText('Protected routes can redirect before rendering', { exact: true })).toBeVisible({ timeout: 5000 });
    await page.getByRole('button', { name: 'Grant access' }).click();
    await expect(page.getByText('Guarded content unlocked', { exact: true })).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('Morgan Reconciler', { exact: true })).toBeVisible();
  });
});