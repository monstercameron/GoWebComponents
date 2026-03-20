import { test, expect } from '@playwright/test';
import { fileURLToPath } from 'node:url';

import { gotoApp } from './support/app.js';

const clientScriptPath = fileURLToPath(new URL('../../tools/livereload/scripts/livereload-client.js', import.meta.url));

test.describe('GoWebComponents hot reload state bridge', () => {
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
    });

    await page.addInitScript({ path: clientScriptPath });
    await gotoApp(page);
  });

  test('restores shared atom state after a reload', async ({ page }) => {
    await expect(page.locator('#atom-value-a')).toHaveText('AtomA: 0');
    await expect(page.locator('#atom-value-b')).toHaveText('AtomB: 0');

    await page.locator('#atom-increment').click();
    await expect(page.locator('#atom-value-a')).toHaveText('AtomA: 1');
    await expect(page.locator('#atom-value-b')).toHaveText('AtomB: 1');

    const payload = await page.evaluate(() => window.GoLiveReload.exportState());
    expect(payload).toContain('sharedCounter');

    await page.evaluate((statePayload) => {
      window.GoLiveReload.storeState(statePayload);
    }, payload);

    const storedState = await page.evaluate(() => sessionStorage.getItem('gwc:livereload:state'));
    expect(storedState).toContain('sharedCounter');

    await page.reload();

    await expect(page.locator('#atom-value-a')).toHaveText('AtomA: 1', { timeout: 15000 });
    await expect(page.locator('#atom-value-b')).toHaveText('AtomB: 1', { timeout: 15000 });
  });
});
