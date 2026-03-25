const fs = require('node:fs/promises');
const path = require('node:path');
const { chromium } = require('@playwright/test');

async function collectBrowserTimings(page) {
  return page.evaluate(() => {
    const navigation = performance.getEntriesByType('navigation')[0];
    const wasmResources = performance
      .getEntriesByType('resource')
      .filter((entry) => entry instanceof PerformanceResourceTiming)
      .filter((entry) => entry.name.endsWith('.wasm'))
      .map((entry) => ({
        name: entry.name,
        transferSize: entry.transferSize,
        encodedBodySize: entry.encodedBodySize,
        decodedBodySize: entry.decodedBodySize,
        responseEnd: entry.responseEnd,
        duration: entry.duration,
      }));

    return {
      navigation: navigation
        ? {
            domContentLoadedEventEnd: navigation.domContentLoadedEventEnd,
            loadEventEnd: navigation.loadEventEnd,
            responseEnd: navigation.responseEnd,
            domInteractive: navigation.domInteractive,
          }
        : null,
      wasmResources,
      startup: window.__gwcStartupProbe || null,
    };
  });
}

async function main() {
  const [url, outPath, timeoutArg] = process.argv.slice(2);
  if (!url || !outPath) {
    throw new Error('usage: node release-startup-probe.cjs <url> <out-path> [timeout-ms]');
  }
  const timeoutMs = Number.parseInt(timeoutArg || '30000', 10) || 30000;

  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  const startedAt = Date.now();
  try {
    const response = await page.goto(url, { waitUntil: 'domcontentloaded', timeout: timeoutMs });
    if (!response) {
      throw new Error('startup probe navigation returned no response');
    }
    if (!response.ok()) {
      throw new Error(`startup probe navigation failed: ${response.status()} ${response.statusText()}`);
    }

    await page.waitForFunction(() => window.__gwcStartupProbe && window.__gwcStartupProbe.readyMs !== null, undefined, { timeout: timeoutMs + 2000 });
    await page.locator('#__gwc_probe_button').click();
    await page.waitForFunction(() => window.__gwcStartupProbe && window.__gwcStartupProbe.interactionMs !== null, undefined, { timeout: timeoutMs });

    const timings = await collectBrowserTimings(page);
    const payload = {
      generatedAt: new Date().toISOString(),
      url,
      totalWallMs: Date.now() - startedAt,
      navigation: timings.navigation,
      wasmResources: timings.wasmResources,
      startup: timings.startup,
    };

    await fs.mkdir(path.dirname(outPath), { recursive: true });
    await fs.writeFile(outPath, JSON.stringify(payload, null, 2), 'utf8');
  } finally {
    await browser.close();
  }
}

main().catch((error) => {
  console.error(error && error.stack ? error.stack : String(error));
  process.exit(1);
});
