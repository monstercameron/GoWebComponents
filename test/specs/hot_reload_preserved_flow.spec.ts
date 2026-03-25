import { test, expect } from '@playwright/test';
import { gotoApp } from './support/app';
import { addLivereloadClient } from './support/livereload';

test.describe('GoWebComponents preserved-state hot reload flows', () => {
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

    await addLivereloadClient(page);
    await gotoApp(page);
  });

  test('keeps shared atoms, cleanup state, local input state, and DOM nodes after reload', async ({ page }) => {
    const inputHandle = await page.locator('#test-input').elementHandle();
    const formHandle = await page.locator('#form-input').elementHandle();

    await expect(page.locator('#memo-value')).toHaveText('Memo Value: 1');
    await expect(page.locator('#memo-runs')).toHaveText('Memo Runs: 1');

    await page.locator('#test-input').fill('Preserve me');
    await expect(page.locator('#input-value')).toHaveText('Input: Preserve me');

    await page.locator('#form-input').fill('submitted value');
    await page.locator('#test-form button[type="submit"]').click();
    await expect(page.locator('#submit-value')).toHaveText('Submitted: submitted value');

    await page.locator('#atom-increment').click();
    await page.locator('#atom-increment').click();
    await expect(page.locator('#atom-value-a')).toHaveText('AtomA: 2');
    await expect(page.locator('#atom-value-b')).toHaveText('AtomB: 2');

    await page.locator('#toggle-child-btn').click();
    await expect(page.locator('#cleanup-status')).toHaveText('mounted');
    await page.locator('#toggle-child-btn').click();
    await expect(page.locator('#cleanup-status')).toHaveText('cleaned');

    const snapshot = await page.evaluate(() => {
      if (!window.GoLiveReload || typeof window.GoLiveReload.exportState !== 'function') {
        throw new Error('snapshot bridge not initialized');
      }
      return window.GoLiveReload.exportState();
    });
    expect(snapshot).toContain('sharedCounter');

    await page.evaluate((stateSnapshot) => {
      if (!window.GoLiveReload || typeof window.GoLiveReload.triggerHotReload !== 'function') {
        throw new Error('hot reload trigger not initialized');
      }
      window.GoLiveReload.triggerHotReload(stateSnapshot);
    }, snapshot);

    await expect(page.locator('#atom-value-a')).toHaveText('AtomA: 2', { timeout: 15000 });
    await expect(page.locator('#atom-value-b')).toHaveText('AtomB: 2', { timeout: 15000 });
    await expect(page.locator('#cleanup-status')).toHaveText('cleaned', { timeout: 15000 });

    await expect(page.locator('#input-value')).toHaveText('Input: Preserve me', { timeout: 15000 });
    await expect(page.locator('#submit-value')).toHaveText('Submitted: submitted value', { timeout: 15000 });
    await expect(page.locator('#memo-value')).toHaveText('Memo Value: 1', { timeout: 15000 });
    await expect(page.locator('#memo-runs')).toHaveText('Memo Runs: 1', { timeout: 15000 });
    await expect(page.locator('#test-input')).toHaveValue('Preserve me');
    await expect(page.locator('#form-input')).toHaveValue('submitted value');

    const memoRuns = await page.evaluate(() => window.__memoReloadRuns);
    expect(memoRuns).toBe(1);

    const sameInputNode = await page.evaluate((node) => node === document.querySelector('#test-input'), inputHandle);
    const sameFormNode = await page.evaluate((node) => node === document.querySelector('#form-input'), formHandle);
    expect(sameInputNode).toBe(true);
    expect(sameFormNode).toBe(true);
  });
});
