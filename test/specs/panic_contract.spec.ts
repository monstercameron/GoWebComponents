import { test, expect } from '@playwright/test';

function collectBrowserErrors(page) {
  const consoleMessages = [];
  const consoleEntries = [];
  const pageErrors = [];
  const pendingConsoleEntries = [];

  page.on('console', async msg => {
    const pending = (async () => {
      const args = [];
      for (const handle of msg.args()) {
        try {
          args.push(await handle.jsonValue());
        } catch {
          args.push(null);
        }
      }
      consoleEntries.push({ type: msg.type(), text: msg.text(), args });
      consoleMessages.push(`${msg.type()}: ${msg.text()}`);
    })();
    pendingConsoleEntries.push(pending);
  });

  page.on('pageerror', error => {
    pageErrors.push(error.message);
  });

  return {
    consoleMessages,
    consoleEntries,
    pageErrors,
    flushConsole: async () => {
      await Promise.allSettled(pendingConsoleEntries.splice(0));
    }
  };
}

test.describe('Wrapped panic contract', () => {
  test('fatal render crash prints wrapped panic metadata', async ({ page }) => {
    const { consoleMessages, consoleEntries, pageErrors, flushConsole } = collectBrowserErrors(page);

    await page.goto('/?crash=render');
    await page.waitForTimeout(1500);
    await flushConsole();

    const errorOutput = [...consoleMessages, ...pageErrors].join('\n');
    const wrappedOccurrences = (errorOutput.match(/\[GWC-RUNTIME-PANIC-RENDER\] uncaught render panic in HelloWorld/g) || []).length;

    expect(errorOutput).toContain('intentional render crash for Playwright logging');
    expect(errorOutput).toContain('GWC-RUNTIME-PANIC-RENDER');
    expect(errorOutput).toContain('where:');
    expect(errorOutput).toContain('path: HelloWorld');
    expect(errorOutput).toContain('runtime:');
    expect(errorOutput).toContain('app:');
    expect(errorOutput).toContain('framework: GWC');
    expect(errorOutput).toContain('platform: GOLANG');
    expect(errorOutput).toContain('next: Match code GWC-RUNTIME-PANIC-RENDER in automation');
    expect(errorOutput).not.toContain('panic: intentional render crash for Playwright logging');
    expect(errorOutput).not.toContain('panic: (runtime.reportedPanic)');
    expect(wrappedOccurrences).toBe(1);

    const structuredEntry = consoleEntries.find(entry => entry.text.includes('[GWC structured panic]'));
    expect(structuredEntry).toBeTruthy();
    expect(structuredEntry.text).toContain('"code":"GWC-RUNTIME-PANIC-RENDER"');
    expect(structuredEntry.text).toContain('"phase":"render"');
    expect(structuredEntry.text).toContain('"subject":"HelloWorld"');
    expect(structuredEntry.text).toContain('"path":"HelloWorld"');
  });

  test('boundary-owned render crash stays non-fatal', async ({ page }) => {
    const { consoleMessages, pageErrors, flushConsole } = collectBrowserErrors(page);

    await page.goto('/?crash=boundary-render');
    await page.waitForSelector('#boundary-fallback', { state: 'visible', timeout: 30000 });
    await flushConsole();

    await expect(page.locator('#boundary-fallback-title')).toHaveText('Boundary fallback rendered');
    await expect(page.locator('#boundary-fallback-error')).toContainText('intentional boundary render crash for Playwright logging');

    const errorOutput = [...consoleMessages, ...pageErrors].join('\n');

    expect(errorOutput).not.toContain('GWC-RUNTIME-PANIC-RENDER');
    expect(errorOutput).not.toContain('runtime: no boundary handled the panic');
  });
});