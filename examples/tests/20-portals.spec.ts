import { expect, test } from '@playwright/test';

test.describe('20-Portals', () => {
  test('renders modal content into the portal root and cleans it up on close', async ({ page }) => {
    await page.goto('/20-portals/portals.html');

    await page.getByRole('button', { name: 'Open Modal' }).click();

    const portalRoot = page.locator('#portal-root');
    await expect(portalRoot.locator('#modal-surface')).toBeVisible();
    await expect(portalRoot.getByText('Portal modal', { exact: true })).toBeVisible();
    await expect(page.locator('#app #modal-surface')).toHaveCount(0);

    await portalRoot.getByRole('button', { name: 'Confirm Modal' }).click();
    await expect(page.getByText('Confirmed modal actions: 1', { exact: true })).toBeVisible();
    await expect(portalRoot.locator('#modal-surface')).toHaveCount(0);
  });

  test('renders tooltip and popover overlays into the shared portal target', async ({ page }) => {
    await page.goto('/20-portals/portals.html');

    await page.getByRole('button', { name: 'Toggle Tooltip' }).click();
    await page.getByRole('button', { name: 'Toggle Popover' }).click();

    const portalRoot = page.locator('#portal-root');
    await expect(portalRoot.locator('#tooltip-surface')).toBeVisible();
    await expect(portalRoot.locator('#popover-surface')).toBeVisible();
    await expect(page.locator('#app #tooltip-surface')).toHaveCount(0);
    await expect(page.locator('#app #popover-surface')).toHaveCount(0);

    await page.getByRole('button', { name: 'Toggle Tooltip' }).click();
    await expect(portalRoot.locator('#tooltip-surface')).toHaveCount(0);
    await expect(portalRoot.locator('#popover-surface')).toBeVisible();

    await portalRoot.getByRole('button', { name: 'Dismiss' }).evaluate(button => {
      (button as HTMLButtonElement).click();
    });
    await expect(portalRoot.locator('#popover-surface')).toHaveCount(0);
  });
});