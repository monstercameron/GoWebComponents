import { expect, test } from '@playwright/test';
import fs from 'node:fs';
import path from 'node:path';

type ExampleEntry = {
  dir: string;
  htmlFile: string;
  url: string;
};

type LinkRecord = {
  href: string;
  text: string;
};

type PageRuntime = {
  pageErrors: string[];
  consoleErrors: string[];
  requestFailures: string[];
};

const examplesRoot = path.resolve(__dirname, '..');
const catalogOrigin = 'http://127.0.0.1:8091';
const routeNotFoundPattern = /route not found|page not found|example not found/i;
const samePageAnchorPattern = /^#(?![!/]|\?|$)/;

function discoverExamples(): ExampleEntry[] {
  return fs
    .readdirSync(examplesRoot, { withFileTypes: true })
    .filter((entry) => entry.isDirectory() && /^\d{2}-/.test(entry.name))
    .map((entry) => {
      const htmlFile = fs
        .readdirSync(path.join(examplesRoot, entry.name))
        .filter((name) => name.endsWith('.html'))
        .sort()[0];

      if (!htmlFile) {
        throw new Error(`No HTML entrypoint found for ${entry.name}`);
      }

      return {
        dir: entry.name,
        htmlFile,
        url: `/examples/${entry.name}/${htmlFile}`,
      };
    })
    .sort((left, right) => left.dir.localeCompare(right.dir));
}

async function settlePage(page: Parameters<typeof test>[1] extends never ? never : any) {
  await page.waitForLoadState('domcontentloaded');
  await page.waitForLoadState('networkidle', { timeout: 15000 }).catch(() => undefined);
  await page.waitForTimeout(400);
  await expect(page.locator('body')).toBeVisible();
}

function attachRuntimeCapture(page: Parameters<typeof test>[1] extends never ? never : any): PageRuntime {
  const runtime: PageRuntime = {
    pageErrors: [],
    consoleErrors: [],
    requestFailures: [],
  };

  page.on('pageerror', (error: Error) => {
    runtime.pageErrors.push(error.message);
  });

  page.on('console', (message: { type(): string; text(): string }) => {
    if (message.type() === 'error') {
      runtime.consoleErrors.push(message.text());
    }
  });

  page.on('requestfailed', (request: { method(): string; url(): string; failure(): { errorText?: string } | null }) => {
    const url = request.url();
    if (!url.startsWith(catalogOrigin)) {
      return;
    }
    runtime.requestFailures.push(`${request.method()} ${url} :: ${request.failure()?.errorText ?? 'failed'}`);
  });

  return runtime;
}

async function collectLinks(page: Parameters<typeof test>[1] extends never ? never : any): Promise<LinkRecord[]> {
  return page.locator('a[href]').evaluateAll((nodes) =>
    nodes
      .map((node) => {
        const anchor = node as HTMLAnchorElement;
        return {
          href: (anchor.getAttribute('href') || '').trim(),
          text: (anchor.textContent || '').replace(/\s+/g, ' ').trim(),
        };
      })
      .filter((link) => link.href.length > 0),
  );
}

function normalizeTarget(currentUrl: string, href: string): string | null {
  if (!href || href === '#' || href.startsWith('mailto:') || href.startsWith('tel:') || href.startsWith('javascript:')) {
    return null;
  }

  const target = new URL(href, currentUrl);
  if (target.origin !== catalogOrigin) {
    return null;
  }

  return target.toString();
}

async function assertNoRuntimeFailures(page: Parameters<typeof test>[1] extends never ? never : any, runtime: PageRuntime, label: string) {
  const bodyText = ((await page.locator('body').innerText().catch(() => '')) || '').trim();
  const failures = [
    ...runtime.pageErrors.map((message) => `pageerror: ${message}`),
    ...runtime.consoleErrors.map((message) => `console: ${message}`),
    ...runtime.requestFailures.map((message) => `request: ${message}`),
  ];

  expect(bodyText.length, `${label} should render body text`).toBeGreaterThan(0);
  expect(bodyText, `${label} rendered a route-not-found screen`).not.toMatch(routeNotFoundPattern);
  expect(failures, `${label} emitted runtime failures`).toEqual([]);
}

async function validateSamePageAnchor(
  page: Parameters<typeof test>[1] extends never ? never : any,
  currentUrl: string,
  href: string,
) {
  const target = new URL(href, currentUrl);
  const anchorId = decodeURIComponent(target.hash.slice(1));
  const exists = await page.evaluate((id: string) => {
    return Boolean(document.getElementById(id) || document.querySelector(`a[name="${CSS.escape(id)}"]`));
  }, anchorId);
  expect(exists, `anchor ${href} should exist on ${currentUrl}`).toBeTruthy();
}

const examples = discoverExamples();
const seedUrls = ['/examples', '/examples/list', ...examples.map((example) => example.url)];

test.describe('Example link integrity', () => {
  test('every same-origin example link resolves from the mounted pages', async ({ context }) => {
    test.setTimeout(10 * 60 * 1000);

    const queue = [...seedUrls];
    const visited = new Set<string>();
    const enqueued = new Set<string>(queue.map((url) => new URL(url, catalogOrigin).toString()));
    const linkOrigins = new Map<string, string[]>();
    const checkedAnchors = new Set<string>();

    while (queue.length > 0) {
      const next = queue.shift();
      if (!next) {
        continue;
      }

      const targetUrl = new URL(next, catalogOrigin).toString();
      if (visited.has(targetUrl)) {
        continue;
      }
      visited.add(targetUrl);

      const crawlPage = await context.newPage();
      const runtime = attachRuntimeCapture(crawlPage);
      const response = await crawlPage.goto(targetUrl, { waitUntil: 'domcontentloaded' });
      expect(response, `navigation should return a response for ${targetUrl}`).not.toBeNull();
      expect(response?.ok(), `navigation failed for ${targetUrl}`).toBeTruthy();
      await settlePage(crawlPage);
      await assertNoRuntimeFailures(crawlPage, runtime, targetUrl);

      const links = await collectLinks(crawlPage);
      for (const link of links) {
        if (samePageAnchorPattern.test(link.href)) {
          const anchorKey = `${targetUrl}::${link.href}`;
          if (!checkedAnchors.has(anchorKey)) {
            checkedAnchors.add(anchorKey);
            await validateSamePageAnchor(crawlPage, targetUrl, link.href);
          }
          continue;
        }

        const normalized = normalizeTarget(targetUrl, link.href);
        if (!normalized) {
          continue;
        }

        if (!normalized.includes('/examples/')) {
          continue;
        }

        const origins = linkOrigins.get(normalized) || [];
        origins.push(`${targetUrl} -> ${link.text || link.href}`);
        linkOrigins.set(normalized, origins);

        if (!enqueued.has(normalized)) {
          enqueued.add(normalized);
          queue.push(normalized);
        }
      }

      await crawlPage.close();
    }

    expect(visited.size, 'the crawler should visit the catalog and example entry pages').toBeGreaterThanOrEqual(seedUrls.length);

    const request = context.request;
    for (const [target, origins] of linkOrigins.entries()) {
      const response = await request.get(target);
      expect(response.ok(), `link target ${target} failed from ${origins.join(', ')}`).toBeTruthy();
    }
  });
});