import { expect, test } from '@playwright/test';

test.describe('browser compatibility smoke', () => {
  test('hydrates prerendered markup in 71-hydrate', async ({ page }) => {
    await page.goto('/71-hydrate/hydrate.html');

    await expect(page.getByRole('heading', { name: 'ui.Hydrate', exact: true })).toBeVisible();
    await expect(page.locator('#app')).toContainText('server-rendered');

    await page.getByRole('button', { name: 'Increment counter' }).click();
    await expect(page.locator('#app')).toContainText('3');

    await page.getByRole('button', { name: 'Toggle topic' }).click();
    await expect(page.locator('#app')).toContainText('client-hydrated');
  });

  test('restores inline bootstrap in 73-ssr-bootstrap', async ({ page }) => {
    await page.goto('/73-ssr-bootstrap/ssr-bootstrap.html');

    await expect(page.getByRole('heading', { name: 'Dedicated SSR bootstrap', exact: true })).toBeVisible();
    await expect(page.locator('#client-status')).toHaveText('Hydrated from inline bootstrap payload');
    await expect(page.locator('#app')).toContainText('/bootstrap');
    await expect(page.locator('#app')).toContainText('inline bootstrap payload');
  });

  test('activates only the explicit islands in 101-static-islands', async ({ page }) => {
    await page.goto('/101-static-islands/islands.html');

    await expect(page.getByRole('heading', { name: 'Static marketing shell, two hydrated islands, and visible startup budgets.', exact: true })).toBeVisible();
    await expect(page.locator('#metric-startup-total')).not.toHaveText('Pending...');
    await expect(page.locator('#metric-newsletter-hydration')).not.toHaveText('Pending...');
    await expect(page.locator('#metric-quote-hydration')).not.toHaveText('Pending...');

    await page.getByRole('button', { name: 'Team', exact: true }).click();
    await expect(page.locator('#newsletter-selected-tier')).toHaveText('Team');
    await expect(page.locator('#metric-newsletter-interaction')).not.toHaveText('Pending...');

    await page.getByRole('button', { name: 'Next note', exact: true }).click();
    await expect(page.locator('#quote-index')).toHaveText('2 / 3');
    await expect(page.locator('#metric-quote-interaction')).not.toHaveText('Pending...');
  });
});
