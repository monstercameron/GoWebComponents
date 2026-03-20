import { test, expect } from '@playwright/test';
import { fileURLToPath } from 'node:url';

import { gotoApp } from './support/app.js';

const clientScriptPath = fileURLToPath(new URL('../../tools/livereload/scripts/livereload-client.js', import.meta.url));
const benchmarkStartKey = 'gwc:hot-reload:benchmark-start';

test.describe('GoWebComponents preserved-state hot reload benchmark', () => {
  test.beforeEach(async ({ page }) => {
    await page.addInitScript(() => {
      class MockWebSocket {
        constructor(url) {
          this.url = url;
          this.readyState = 1;
          setTimeout(() => {
            if (typeof this.onopen === 'function') {
              this.onopen();
            }
          }, 0);
        }

        send() {}

        close() {
          if (typeof this.onclose === 'function') {
            this.onclose();
          }
        }
      }

      window.WebSocket = MockWebSocket;
      window.hotReloadWasm = function() {
        throw new Error('simulated hot reload failure');
      };
    });

    await page.addInitScript({ path: clientScriptPath });
    await gotoApp(page);
  });

  test('restores a preserved state snapshot within a bounded round-trip time', async ({ page }) => {
    await page.locator('#atom-increment').click();
    await expect(page.locator('#atom-value-a')).toHaveText('AtomA: 1');

    const snapshot = await page.evaluate(() => window.GoLiveReload.exportState());
    expect(snapshot).toContain('sharedCounter');

    await page.evaluate((key) => {
      sessionStorage.setItem(key, String(Date.now()));
    }, benchmarkStartKey);

    await page.evaluate((stateSnapshot) => {
      window.GoLiveReload.triggerHotReload(stateSnapshot);
    }, snapshot);

    await expect(page.locator('#atom-value-a')).toHaveText('AtomA: 1', { timeout: 15000 });

    const roundTripMs = await page.evaluate((key) => {
      const startedAt = Number(sessionStorage.getItem(key) || '0');
      return startedAt > 0 ? Date.now() - startedAt : Number.POSITIVE_INFINITY;
    }, benchmarkStartKey);

    expect(roundTripMs).toBeLessThan(10000);
    test.info().annotations.push({
      type: 'benchmark',
      description: `hot reload snapshot round-trip: ${roundTripMs}ms`,
    });
  });
});
