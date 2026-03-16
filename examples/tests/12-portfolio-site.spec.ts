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

  test('supports primary calls to action and external links', async ({ page }) => {
    await page.goto('/12-portfolio-site/portfolio.html');

    await expect(page.getByRole('button', { name: /See GoWebComponents in Action/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /Get In Touch/i })).toBeVisible();
    await expect(page.getByRole('button', { name: /View Project/i }).first()).toBeVisible();

    await expect(page.getByRole('link', { name: 'GitHub' }).first()).toHaveAttribute('href', /monstercameron\/GoWebComponents/);
    await expect(page.getByRole('link', { name: 'Connect' })).toHaveAttribute('href', /linkedin\.com\/in\/earl-cameron/i);
  });
});