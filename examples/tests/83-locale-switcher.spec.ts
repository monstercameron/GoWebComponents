import { expect, test } from '@playwright/test';

test.describe('83-Locale Switcher', () => {
  test('switches locale, direction, and formatted copy together', async ({ page }) => {
    await page.goto('/83-locale-switcher/locale-switcher.html');

    await expect(page.locator('#locale-switcher-headline')).toHaveText('Ship every market from one component tree');
    await expect(page.locator('[data-current-locale="en"]')).toBeVisible();

    await page.locator('#switch-locale-fr').click();
    await expect(page.locator('#locale-switcher-headline')).toHaveText('Lancez chaque marche depuis le meme arbre de composants');
    await expect(page.locator('[data-current-locale="fr"]')).toBeVisible();
    await expect(page.locator('#locale-switcher-date')).toHaveText('16 March 2026');

    await page.locator('#switch-locale-ar').click();
    await expect(page.locator('#locale-switcher-headline')).toHaveText('اطلق كل سوق من شجرة مكونات واحدة');
    await expect(page.locator('[dir="rtl"]')).toBeVisible();

    await page.locator('#locale-switcher-add').click();
    await expect(page.locator('#locale-switcher-cart-copy')).toContainText('3');
  });
});