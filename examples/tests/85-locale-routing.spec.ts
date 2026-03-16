import { expect, test } from '@playwright/test';

test.describe('85-Locale Routing', () => {
  test('resolves locale prefixes and loader content from the current route', async ({ page }) => {
    await page.goto('/85-locale-routing/locale-routing.html');

    await expect(page.locator('#locale-routing-locale')).toHaveText('en');
    await expect(page.locator('#locale-routing-base')).toHaveText('/pricing');
    await expect(page.locator('#locale-routing-path')).toHaveText('/pricing');

    await page.locator('#locale-route-fr').click();
    await expect(page.locator('#locale-routing-locale')).toHaveText('fr');
    await expect(page.locator('#locale-routing-path')).toHaveText('/fr/pricing');
    await expect(page.locator('#locale-routing-title')).toHaveText('Route tarifaire avec prefixe de langue');

    await page.locator('#locale-route-ar').click();
    await expect(page.locator('#locale-routing-locale')).toHaveText('ar');
    await expect(page.locator('[dir="rtl"]')).toBeVisible();
    await expect(page.locator('#locale-routing-path')).toHaveText('/ar/pricing');
  });
});