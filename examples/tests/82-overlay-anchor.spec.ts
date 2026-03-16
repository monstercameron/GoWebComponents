import { expect, test } from '@playwright/test';

test.describe('82-Overlay Anchor', () => {
  test('keeps tooltip layering stable across selector and explicit portal targets', async ({ page }) => {
    await page.goto('/82-overlay-anchor/overlay-anchor.html');

    await page.getByRole('button', { name: 'Open anchored menu' }).click();

    const selectorRoot = page.locator('#overlay-anchor-root');
    const explicitRoot = page.locator('#overlay-anchor-explicit-root');
    await expect(selectorRoot.locator('#overlay-anchor-menu')).toBeVisible();

    await page.locator('#toggle-overlay-tooltip').evaluate(button => {
      (button as HTMLButtonElement).click();
    });
    await expect(selectorRoot.locator('#overlay-anchor-tooltip')).toBeVisible();
    await expect
      .poll(async () => page.locator('#overlay-anchor-tooltip').getAttribute('data-overlay-depth'))
      .toBe('1');
    await expect
      .poll(async () => page.locator('#overlay-anchor-menu').getAttribute('data-overlay-depth'))
      .toBe('0');

    await page.locator('#use-explicit-overlay-target').evaluate(button => {
      (button as HTMLButtonElement).click();
    });
    await page.locator('#toggle-overlay-tooltip').evaluate(button => {
      (button as HTMLButtonElement).click();
    });
    await page.locator('#toggle-overlay-tooltip').evaluate(button => {
      (button as HTMLButtonElement).click();
    });

    await expect(selectorRoot.locator('#overlay-anchor-tooltip')).toHaveCount(0);
    await expect(explicitRoot.locator('#overlay-anchor-tooltip')).toBeVisible();
    await expect(selectorRoot.locator('#overlay-anchor-menu')).toBeVisible();

    await page.keyboard.press('Escape');
    await expect(selectorRoot.locator('#overlay-anchor-menu')).toHaveCount(0);
    await expect(explicitRoot.locator('#overlay-anchor-tooltip')).toHaveCount(0);
  });
});