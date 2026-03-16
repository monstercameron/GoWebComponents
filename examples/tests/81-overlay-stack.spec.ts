import { expect, test } from '@playwright/test';

test.describe('81-Overlay Stack', () => {
  test('routes escape and focus through the topmost eligible layers', async ({ page }) => {
    await page.goto('/81-overlay-stack/overlay-stack.html');

    await page.getByRole('button', { name: 'Open release board' }).click();
    await expect(page.locator('#overlay-stack-root #overlay-stack-parent-dialog')).toBeVisible();
    await expect.poll(async () => page.evaluate(() => document.body.style.overflow)).toBe('hidden');

    await page.locator('#open-overlay-popover').click();
    await expect(page.locator('#overlay-stack-popover')).toBeVisible();

    await page.keyboard.press('Escape');
    await expect(page.locator('#overlay-stack-popover')).toHaveCount(0);
    await expect(page.locator('#overlay-stack-parent-dialog')).toBeVisible();
    await expect.poll(async () => page.evaluate(() => document.body.style.overflow)).toBe('hidden');

    await page.locator('#open-nested-overlay-dialog').click();
    await expect(page.locator('#overlay-stack-nested-dialog')).toBeVisible();
    await expect.poll(async () => page.evaluate(() => document.activeElement?.id)).toBe('confirm-nested-overlay-dialog');

    await page.keyboard.press('Escape');
    await expect(page.locator('#overlay-stack-nested-dialog')).toHaveCount(0);
    await expect(page.locator('#overlay-stack-parent-dialog')).toBeVisible();
    await expect.poll(async () => page.evaluate(() => document.activeElement?.id)).toBe('open-nested-overlay-dialog');
    await expect
      .poll(async () => page.locator('#overlay-stack-parent-dialog').getAttribute('data-overlay-handles-escape'))
      .toBe('true');
    await expect.poll(async () => page.evaluate(() => document.body.style.overflow)).toBe('hidden');

    await page.keyboard.press('Escape');
    await expect(page.locator('#overlay-stack-parent-dialog')).toHaveCount(0);
    await expect.poll(async () => page.evaluate(() => document.body.style.overflow)).toBe('');
    await expect.poll(async () => page.evaluate(() => document.activeElement?.id)).toBe('open-overlay-stack-dialog');
  });
});