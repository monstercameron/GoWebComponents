import { expect, test } from '@playwright/test';

test.describe('84-SSR I18n Bootstrap', () => {
  test('hydrates locale and messages from the bootstrap payload', async ({ page }) => {
    await page.goto('/84-ssr-i18n-bootstrap/ssr-i18n-bootstrap.html');

    await expect(page.locator('#bootstrap-locale-headline')).toHaveText('Bonjour Cam');
    await expect(page.locator('#bootstrap-locale-value')).toHaveText('fr');
    await expect(page.locator('#bootstrap-message-count')).toHaveText('6');
    await expect(page.locator('#bootstrap-client-status')).toHaveText('Hydrated locale and messages from ui.SSRBootstrap.I18n');

    await page.locator('#bootstrap-switch-en').click();
    await expect(page.locator('#bootstrap-locale-headline')).toHaveText('Hello Cam');
  });
});