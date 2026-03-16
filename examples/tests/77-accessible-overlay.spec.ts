import { expect, test } from '@playwright/test';

test.describe('77-Accessible Overlay', () => {
  test('traps focus, hides the background shell, and restores the trigger focus on close', async ({ page }) => {
    await page.goto('/77-accessible-overlay/accessible-overlay.html');

    await page.getByRole('button', { name: 'Open accessible dialog' }).click();

    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();
    await expect(page.locator('#overlay-page-shell')).toHaveAttribute('aria-hidden', 'true');
    await expect.poll(async () => page.evaluate(() => document.body.style.overflow)).toBe('hidden');
    await expect.poll(async () => page.evaluate(() => document.activeElement?.id)).toBe('confirm-accessible-dialog');

    await page.keyboard.press('Tab');
    await expect.poll(async () => page.evaluate(() => document.activeElement?.id)).toBe('close-accessible-dialog');

    await page.keyboard.press('Tab');
    await expect.poll(async () => page.evaluate(() => document.activeElement?.id)).toBe('confirm-accessible-dialog');

    await page.keyboard.press('Escape');
    await expect(dialog).toHaveCount(0);
    await expect(page.locator('#overlay-page-shell')).not.toHaveAttribute('aria-hidden', 'true');
    await expect.poll(async () => page.evaluate(() => document.body.style.overflow)).toBe('');
    await expect.poll(async () => page.evaluate(() => document.activeElement?.id)).toBe('open-accessible-overlay');
  });
});