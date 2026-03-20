import { test, expect } from '@playwright/test';
import { fileURLToPath } from 'node:url';

import { gotoApp } from './support/app.js';

const clientScriptPath = fileURLToPath(new URL('../../tools/livereload/scripts/livereload-client.js', import.meta.url));

test.describe('GoWebComponents hot reload snapshot transport', () => {
  test.beforeEach(async ({ page }) => {
    await page.addInitScript(() => {
      class MockWebSocket {
        constructor(url) {
          this.url = url;
          this.readyState = 1;
          this.sentMessages = [];
          window.__mockWebSockets = window.__mockWebSockets || [];
          window.__mockWebSockets.push(this);
          setTimeout(() => {
            if (typeof this.onopen === 'function') {
              this.onopen();
            }
          }, 0);
        }

        send(data) {
          this.sentMessages.push(data);
        }

        close() {
          if (typeof this.onclose === 'function') {
            this.onclose();
          }
        }
      }

      window.WebSocket = MockWebSocket;
      window.__mockWebSockets = [];
      window.hotReloadWasm = function() {
        throw new Error('simulated hot reload failure');
      };
    });

    await page.addInitScript({ path: clientScriptPath });
    await gotoApp(page);
  });

  test('requests a snapshot and reuses the server-provided payload on hot reload', async ({ page }) => {
    await expect(page.locator('#atom-value-a')).toHaveText('AtomA: 0');
    await expect(page.locator('#atom-value-b')).toHaveText('AtomB: 0');

    await page.locator('#atom-increment').click();
    await expect(page.locator('#atom-value-a')).toHaveText('AtomA: 1');
    await expect(page.locator('#atom-value-b')).toHaveText('AtomB: 1');

    const exportedSnapshot = await page.evaluate(() => window.GoLiveReload.exportState());
    expect(exportedSnapshot).toContain('sharedCounter');

    await page.evaluate((stateSnapshot) => {
      if (!window.GoLiveReload || typeof window.GoLiveReload.triggerHotReload !== 'function') {
        throw new Error('hot reload trigger not initialized');
      }
      window.GoLiveReload.triggerHotReload(stateSnapshot);
    }, exportedSnapshot);

    await expect(page.locator('#atom-value-a')).toHaveText('AtomA: 1', { timeout: 15000 });
    await expect(page.locator('#atom-value-b')).toHaveText('AtomB: 1', { timeout: 15000 });
  });
});
