import { expect, test } from '@playwright/test';

import { ensureServiceWorkerReady, exampleURL } from './support/pwa';

const offlineCacheURL = exampleURL('97-pwa-offline-cache/offline-cache.html');

test.describe('97-PWA Offline Cache', () => {
  test('renders offline cache controls and diagnostics panel', async ({ page }) => {
    await page.goto(offlineCacheURL);

    await expect(page.getByRole('heading', { name: 'PWA offline cache', exact: true })).toBeVisible({ timeout: 15000 });
    await expect(page.getByRole('button', { name: 'Warm offline cache', exact: true })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Queue offline write', exact: true })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Queue conflicting write', exact: true })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Replay queued writes', exact: true })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Replay with conflict policy', exact: true })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Schedule background replay', exact: true })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Inspect diagnostics', exact: true })).toBeVisible();
    await expect(page.locator('#offline-diagnostics-preview')).toContainText('Click Inspect diagnostics to capture a structured PWA snapshot.');
  });

  test('warms cache, cleans stale caches, supports offline navigation, and replays queued writes', async ({ page, context }) => {
    await page.goto(offlineCacheURL);

    await expect(page.getByRole('heading', { name: 'PWA offline cache', exact: true })).toBeVisible({ timeout: 15000 });

    await page.evaluate(async () => {
      await caches.open('pwa-offline-cache-demo-legacy');
    });

    await page.getByRole('button', { name: 'Warm offline cache', exact: true }).click();
    await expect(page.locator('#offline-cache-status')).toContainText(/Cached \d+ release entries|Cache warmup failed/i, { timeout: 15000 });

    await ensureServiceWorkerReady(page, '/97-pwa-offline-cache/sw.js', '/97-pwa-offline-cache/');

    const cacheNames = await page.evaluate(async () => caches.keys());
    expect(cacheNames).toContain('pwa-offline-cache-demo-v1');
    expect(cacheNames).not.toContain('pwa-offline-cache-demo-legacy');

    await page.getByRole('button', { name: 'Queue offline write', exact: true }).click();
    await expect(page.locator('#offline-queue-status')).toContainText('Queued offline write', { timeout: 10000 });

    await page.getByRole('button', { name: 'Replay queued writes', exact: true }).click();
    await expect(page.locator('#offline-replay-status')).toContainText('Replay succeeded=1', { timeout: 10000 });

    await page.getByRole('button', { name: 'Queue conflicting write', exact: true }).click();
    await expect(page.locator('#offline-conflict-status')).toContainText('Queued conflict demo write', { timeout: 10000 });

    await page.getByRole('button', { name: 'Replay with conflict policy', exact: true }).click();
    await expect(page.locator('#offline-conflict-status')).toContainText('Resolved 1 conflict', { timeout: 10000 });

    await page.getByRole('button', { name: 'Replay with conflict policy', exact: true }).click();
    await expect(page.locator('#offline-conflict-status')).toContainText('Conflict-aware replay succeeded=1', { timeout: 10000 });

    await page.getByRole('button', { name: 'Schedule background replay', exact: true }).click();
    await expect(page.locator('#offline-background-sync-status')).toContainText(/registered with tag|unavailable/i, { timeout: 10000 });

    await page.getByRole('button', { name: 'Inspect diagnostics', exact: true }).click();
    await expect(page.locator('#offline-diagnostics-preview')).toContainText('queued writes: 0', { timeout: 10000 });
    await expect(page.locator('#offline-diagnostics-preview')).toContainText(/background sync available: (true|false)/i, { timeout: 10000 });

    await context.setOffline(true);
    const offlinePage = await context.newPage();
    await offlinePage.goto(exampleURL('97-pwa-offline-cache/missing-route'), { waitUntil: 'domcontentloaded' });
    await expect(offlinePage.getByRole('heading', { name: 'Network unavailable', exact: true })).toBeVisible({ timeout: 15000 });
    await offlinePage.close();
    await context.setOffline(false);
  });
});