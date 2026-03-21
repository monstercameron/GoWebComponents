import { expect, test } from '@playwright/test';

import { openExamplePair } from './support/pwa';

test.describe('97-PWA Multi-Client Coordination', () => {
  test('coordinates two wasm tabs while keeping PWA shell state inspectable', async ({ browser }) => {
    const { context, pageA, pageB } = await openExamplePair(browser, '97-pwa-multi-client/pwa-multi-client.html');

    try {
      await expect(pageA.getByRole('heading', { name: 'PWA multi-client coordination', exact: true })).toBeVisible({ timeout: 15000 });
      await expect(pageB.getByRole('heading', { name: 'PWA multi-client coordination', exact: true })).toBeVisible({ timeout: 15000 });

      await expect(pageA.locator('#multi-client-pwa-last-sent')).toContainText(/hello|failed/i, { timeout: 10000 });
      await expect(pageB.locator('#multi-client-pwa-last-sent')).toContainText(/hello|failed/i, { timeout: 10000 });

      await pageA.getByRole('button', { name: 'Announce peer', exact: true }).click();
      await expect(pageB.locator('#multi-client-pwa-last-received')).toContainText('Peer hello', { timeout: 10000 });

      await pageB.getByRole('button', { name: 'Announce peer', exact: true }).click();
      await expect(pageA.locator('#multi-client-pwa-last-received')).toContainText('Peer hello', { timeout: 10000 });

      await pageA.getByRole('button', { name: 'Broadcast sync event', exact: true }).click();
      await expect(pageB.locator('#multi-client-pwa-last-received')).toContainText('Received sync event', { timeout: 10000 });

      await pageA.getByRole('button', { name: 'Warm offline shell', exact: true }).click();
      await expect(pageA.locator('#multi-client-pwa-cache-status')).toContainText(/Cached \d+ entries|Shell warmup failed/i, { timeout: 15000 });

      await pageA.getByRole('button', { name: 'Broadcast cache invalidation', exact: true }).click();
      await expect(pageB.locator('#multi-client-pwa-last-received')).toContainText('Received cache invalidation', { timeout: 10000 });

      await pageA.getByRole('button', { name: 'Inspect diagnostics', exact: true }).click();
      await expect(pageA.locator('#multi-client-pwa-diagnostics-preview')).toContainText('manifest valid: true', { timeout: 10000 });
      await expect(pageA.locator('#multi-client-pwa-diagnostics-preview')).toContainText(/cache entries: \d+/, { timeout: 10000 });
    } finally {
      await context.close();
    }
  });
});