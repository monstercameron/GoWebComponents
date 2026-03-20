import { expect, test } from '@playwright/test';

test.describe('96-Code Splitting Demo', () => {
  test('keeps route shells stable while lazy panels resolve', async ({ page }) => {
    await page.goto('/96-code-splitting/code-splitting.html');

    await expect(page.getByRole('heading', { name: 'Keep route shells stable while lazy panels resolve underneath them.', exact: true })).toBeVisible();

    await page.getByRole('button', { name: 'Open Catalog', exact: true }).click();
    await expect(page.getByRole('heading', { name: 'Catalog route family', exact: true })).toBeVisible();
    await expect(page.getByText('Resolving Catalog overview panel...', { exact: true })).toBeVisible();
    await expect(page.getByText('Catalog overview panel Version 1 resolves after the route shell is already on screen.', { exact: true })).toBeVisible({ timeout: 10000 });

    await page.getByRole('button', { name: 'Reload panel', exact: true }).click();
    await expect(page.getByText('Resolving Catalog overview panel...', { exact: true })).toBeVisible();
    await expect(page.getByText('Catalog overview panel Version 2 resolves after the route shell is already on screen.', { exact: true })).toBeVisible({ timeout: 10000 });

    await page.getByRole('button', { name: 'Operations', exact: true }).first().click();
    await expect(page.getByRole('heading', { name: 'Operations route family', exact: true })).toBeVisible();
    await expect(page.getByText('Operations queue panel Version 1 resolves after the route shell is already on screen.', { exact: true })).toBeVisible({ timeout: 10000 });
  });
});
