import { expect, test } from '@playwright/test';

test.describe('19-Nested Routes', () => {
  test.describe('contractual behavior', () => {
    test('renders nested dashboard and settings layouts with explicit outlets', async ({ page }) => {
      await page.goto('/19-nested-routes/nested-routes.html#/dashboard/settings/profile');

      await expect(page.getByRole('heading', { name: 'Operations workspace' })).toBeVisible();
      await expect(page.getByRole('heading', { name: 'Settings shell inside the dashboard tree' })).toBeVisible();
      await expect(page.getByRole('heading', { name: 'Profile settings' })).toBeVisible();

      await page.getByRole('link', { name: 'Team' }).click();
      await expect(page.getByRole('heading', { name: 'Operations workspace' })).toBeVisible();
      await expect(page.getByRole('heading', { name: 'Settings shell inside the dashboard tree' })).toBeVisible();
      await expect(page.getByRole('heading', { name: 'Team settings' })).toBeVisible();
      await expect(page).toHaveURL(/nested-routes\.html#\/dashboard\/settings\/team$/);
    });

    test('switches route trees while keeping the matched layout shell stable', async ({ page }) => {
      await page.goto('/19-nested-routes/nested-routes.html');

      await page.getByRole('link', { name: 'Open Dashboard' }).click();

      await expect(page.getByRole('heading', { name: 'Operations workspace' })).toBeVisible();
      await expect(page.getByRole('heading', { name: 'Overview' })).toBeVisible();

      await page.getByRole('link', { name: 'Docs' }).click();
      await expect(page.getByRole('heading', { name: 'Guide navigation' })).toBeVisible();
      await expect(page.getByRole('heading', { name: 'Getting started' })).toBeVisible();

      await page.getByRole('link', { name: 'Routing' }).click();
      await expect(page.getByRole('heading', { name: 'Guide navigation' })).toBeVisible();
      await expect(page.getByRole('heading', { name: 'Routing rules' })).toBeVisible();
      await expect(page).toHaveURL(/nested-routes\.html#\/docs\/routing$/);
    });
  });

  test.describe('aspirational flows', () => {
    test('supports repeated nested param navigation without dropping the parent shell', async ({ page }) => {
      const pageErrors: string[] = [];
      page.on('pageerror', error => pageErrors.push(error.message));

      await page.goto('/19-nested-routes/nested-routes.html');
      await page.getByRole('link', { name: 'Open Dashboard' }).click();
      await page.getByRole('link', { name: 'Report 7' }).click();

      await expect(page.getByRole('heading', { name: 'Operations workspace' })).toBeVisible();
      await expect(page.getByRole('heading', { name: 'Report #7' })).toBeVisible();

      await page.getByRole('link', { name: 'Open report #12' }).click();
      await expect(page.getByRole('heading', { name: 'Operations workspace' })).toBeVisible();
      await expect(page.getByRole('heading', { name: 'Report #12' })).toBeVisible();
      await expect(page).toHaveURL(/nested-routes\.html#\/dashboard\/reports\/12$/);

      await page.getByRole('link', { name: 'Report 7' }).click();
      await expect(page.getByRole('heading', { name: 'Operations workspace' })).toBeVisible();
      await expect(page.getByRole('heading', { name: 'Report #7' })).toBeVisible();
      await expect(pageErrors).toEqual([]);
    });

    test('allows deep-route jumps from the landing page into either layout tree', async ({ page }) => {
      await page.goto('/19-nested-routes/nested-routes.html');

      await expect(page.getByRole('heading', { name: /Dashboard shells, nested settings pages, and docs navigation/i })).toBeVisible();

      await page.getByRole('link', { name: 'Open Settings' }).click();
      await expect(page.getByRole('heading', { name: 'Operations workspace' })).toBeVisible();
      await expect(page.getByRole('heading', { name: 'Settings shell inside the dashboard tree' })).toBeVisible();
      await expect(page.getByRole('heading', { name: 'Profile settings' })).toBeVisible();

      await page.getByRole('link', { name: 'Home' }).click();
      await expect(page.getByRole('heading', { name: /Dashboard shells, nested settings pages, and docs navigation/i })).toBeVisible();

      await page.getByRole('link', { name: 'Open Docs' }).click();
      await expect(page.getByRole('heading', { name: 'Guide navigation' })).toBeVisible();
      await expect(page.getByRole('heading', { name: 'Getting started' })).toBeVisible();
    });
  });
});