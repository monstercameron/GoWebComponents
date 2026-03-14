import { chromium } from '@playwright/test';
import { dirname } from 'path';
import { fileURLToPath } from 'url';
import { spawn } from 'child_process';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const testsDir = dirname(__dirname);

const server = spawn('npx', ['http-server', 'benchmark', '-p', '8080'], {
  cwd: testsDir,
  stdio: 'ignore',
  shell: true,
});

try {
  await new Promise((resolve) => setTimeout(resolve, 3000));

  const browser = await chromium.launch();
  const page = await browser.newPage();
  page.on('pageerror', (err) => console.error('pageerror:', err));
  page.on('console', (msg) => console.log('console:', msg.type(), msg.text()));

  await page.goto('http://127.0.0.1:8080/benchmark.html');
  await page.waitForSelector('#btn-render', { timeout: 30000 });

  await page.click('#btn-render');
  await page.waitForFunction(
    () => document.querySelector('#item-count')?.textContent === 'Count: 250',
    { timeout: 30000 },
  );

  console.log('after render count:', await page.textContent('#item-count'));
  console.log('first item before update:', await page.textContent('.list-item'));

  await page.dispatchEvent('#btn-update', 'click');
  await page.waitForTimeout(5000);

  const snapshot = await page.evaluate(() => ({
    readyState: document.readyState,
    bodyLength: document.body?.innerHTML.length ?? 0,
    itemCount: document.querySelector('#item-count')?.textContent ?? null,
    firstItem: document.querySelector('.list-item')?.textContent ?? null,
  }));
  console.log('after update snapshot:', snapshot);

  await browser.close();
} finally {
  server.kill();
}
