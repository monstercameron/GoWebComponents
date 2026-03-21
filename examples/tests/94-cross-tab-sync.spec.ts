import { expect, test } from '@playwright/test';

import { openExamplePair } from './support/pwa';

test.describe('94-Cross-Tab Sync', () => {
  test('propagates theme, logout, cache invalidation, and draft updates across tabs', async ({ browser }) => {
    const { context, pageA, pageB } = await openExamplePair(browser, '94-cross-tab-sync/cross-tab-sync.html');

    try {
      await expect(pageA.getByRole('heading', { name: 'Cross-Tab Sync', exact: true })).toBeVisible({ timeout: 15000 });
      await expect(pageB.getByRole('heading', { name: 'Cross-Tab Sync', exact: true })).toBeVisible({ timeout: 15000 });

      await pageA.getByRole('button', { name: 'Theme: dark', exact: true }).click();
      await expect(pageB.getByText('Received theme "dark"', { exact: false })).toBeVisible({ timeout: 10000 });

      await pageA.getByRole('button', { name: 'Broadcast logout', exact: true }).click();
      await expect(pageB.getByText('Received auth signal "signed-out"', { exact: false })).toBeVisible({ timeout: 10000 });

      await pageA.getByRole('button', { name: 'Invalidate cache', exact: true }).click();
      await expect(pageB.getByText('Received cache invalidation for products:list rev', { exact: false })).toBeVisible({ timeout: 10000 });

      await pageA.locator('#cross-tab-draft').fill('PWA coordination survives a second tab.');
      await pageA.getByRole('button', { name: 'Broadcast draft', exact: true }).click();
      await expect(pageB.locator('#cross-tab-draft')).toHaveValue('PWA coordination survives a second tab.', { timeout: 10000 });
    } finally {
      await context.close();
    }
  });
});