import { test, expect } from '@playwright/test';
import { gotoApp } from './support/app';
import { addLivereloadClient } from './support/livereload';

test.describe('GoWebComponents hot reload failure recovery', () => {
  test.beforeEach(async ({ page }) => {
    await page.addInitScript(() => {
      class MockWebSocket {
        constructor(url) {
          this.url = url;
          this.readyState = 1;
          window.__mockWebSockets = window.__mockWebSockets || [];
          window.__mockWebSockets.push(this);
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
      window.__mockWebSockets = [];
      window.hotReloadWasm = function() {
        throw new Error('simulated hot reload failure');
      };
    });

    await addLivereloadClient(page);
    await gotoApp(page);
  });

  test('falls back to a full reload and restores state when hot reload throws', async ({ page }) => {
    await expect(page.locator('#atom-value-a')).toHaveText('AtomA: 0');
    await expect(page.locator('#atom-value-b')).toHaveText('AtomB: 0');

    await page.locator('#atom-increment').click();
    await expect(page.locator('#atom-value-a')).toHaveText('AtomA: 1');
    await expect(page.locator('#atom-value-b')).toHaveText('AtomB: 1');

    await page.evaluate(() => {
      if (!window.GoLiveReload || typeof window.GoLiveReload.triggerHotReload !== 'function') {
        throw new Error('hot reload trigger not initialized');
      }
      window.GoLiveReload.triggerHotReload();
    });

    await expect(page.locator('#atom-value-a')).toHaveText('AtomA: 1', { timeout: 15000 });
    await expect(page.locator('#atom-value-b')).toHaveText('AtomB: 1', { timeout: 15000 });

    const storedState = await page.evaluate(() => sessionStorage.getItem('gwc:livereload:state'));
    expect(storedState).toContain('sharedCounter');
  });
});
