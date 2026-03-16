import { test, expect } from '@playwright/test';

test.describe('12-Portfolio Site', () => {
  test('renders the main website sections', async ({ page }) => {
    await page.goto('/12-portfolio-site/portfolio.html');

    await expect(page.getByRole('heading', { name: 'Earl Cameron', exact: true }).first()).toBeVisible({ timeout: 15000 });
    await expect(page.getByRole('heading', { name: 'Recent Projects', exact: true })).toBeVisible();
    await expect(page.getByRole('heading', { name: "Let's Connect", exact: true })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'GoWebComponents', exact: true }).first()).toBeVisible();
    await expect(page.getByRole('heading', { name: 'gRPC Tunnel', exact: true })).toBeVisible();
  });

  test('renders the docs route and returns to the main site', async ({ page }) => {
    await page.goto('/12-portfolio-site/portfolio.html');
    await expect(page.getByRole('heading', { name: 'Earl Cameron', exact: true }).first()).toBeVisible({ timeout: 15000 });
    await page.evaluate(() => {
      window.location.hash = '/docs';
    });

    await expect(page.getByRole('heading', { name: /GoWebComponents API Documentation/i })).toBeVisible({ timeout: 15000 });
    await expect(page.getByRole('heading', { name: /Table of Contents/i })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Core Types', exact: true }).first()).toBeVisible();

    await page.getByRole('button', { name: /Back to App/i }).click();
    await expect(page.getByRole('heading', { name: 'Earl Cameron', exact: true }).first()).toBeVisible();
    await expect(page).toHaveURL(/portfolio\.html#\/?$/);
  });

  test('renders the 404 route and recovers into docs and home flows', async ({ page }) => {
    await page.goto('/12-portfolio-site/portfolio.html');
    await expect(page.getByRole('heading', { name: 'Earl Cameron', exact: true }).first()).toBeVisible({ timeout: 15000 });
    await page.evaluate(() => {
      window.location.hash = '/missing-route';
    });

    await expect(page.getByRole('heading', { name: 'Page Not Found', exact: true })).toBeVisible({ timeout: 15000 });
    await expect(page.getByRole('button', { name: /View Docs/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /Go Home/i })).toBeVisible();

    await page.getByRole('button', { name: /View Docs/i }).click();
    await expect(page.getByRole('heading', { name: /GoWebComponents API Documentation/i })).toBeVisible();
    await expect(page).toHaveURL(/portfolio\.html#\/docs$/);

    await page.evaluate(() => {
      window.location.hash = '/missing-route';
    });
    await expect(page.getByRole('heading', { name: 'Page Not Found', exact: true })).toBeVisible({ timeout: 15000 });
    await page.getByRole('button', { name: /Go Home/i }).click();
    await expect(page.getByRole('heading', { name: 'Earl Cameron', exact: true }).first()).toBeVisible();
    await expect(page).toHaveURL(/portfolio\.html#\/?$/);
  });

  test('supports primary calls to action and external links', async ({ page }) => {
    await page.goto('/12-portfolio-site/portfolio.html');

    await expect(page.getByRole('button', { name: /See GoWebComponents in Action/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /Get In Touch/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /View Project/i }).first()).toBeVisible();

    await expect(page.getByRole('link', { name: 'GitHub' }).first()).toHaveAttribute('href', /monstercameron\/GoWebComponents/);
    await expect(page.getByRole('link', { name: 'Connect' })).toHaveAttribute('href', /linkedin\.com\/in\/earl-cameron/i);
  });
});