import { expect, test } from '@playwright/test';

const installabilityURL = 'http://localhost:8081/97-pwa-installability/installability.html';

test.describe('97-PWA Installability', () => {
  test('renders installability guidance and manifest wiring', async ({ page }) => {
    await page.goto(installabilityURL);

    await expect(page.getByRole('heading', { name: 'PWA installability', exact: true })).toBeVisible({ timeout: 15000 });
    await expect(page.locator('#pwa-installability-manifest-preview')).toContainText('GoWebComponents PWA Installability Demo');
    await expect(page.locator('h1 + p')).toContainText('Observe manifest validity, install-prompt readiness, and service-worker lifecycle');
    await expect(page.locator('#pwa-installability-reasons')).toContainText(/Waiting for browser installability signals|browser has not exposed/i);
  });

  test('handles installability events and service-worker update requests', async ({ page }) => {
    await page.goto(installabilityURL);

    await expect(page.getByRole('heading', { name: 'PWA installability', exact: true })).toBeVisible({ timeout: 15000 });
    await expect(page.locator('#pwa-installability-sw-status')).toContainText(/Registering service worker|registered|failed/i, { timeout: 15000 });

    await page.evaluate(() => {
      const installEvent = new Event('beforeinstallprompt');
      Object.defineProperty(installEvent, 'prompt', {
        configurable: true,
        value: () => Promise.resolve(),
      });
      Object.defineProperty(installEvent, 'userChoice', {
        configurable: true,
        value: Promise.resolve({ outcome: 'accepted', platform: 'web' }),
      });
      Object.defineProperty(installEvent, 'preventDefault', {
        configurable: true,
        value: () => undefined,
      });
      window.dispatchEvent(installEvent);
    });

    await page.getByRole('button', { name: 'Prompt install', exact: true }).click();
    await expect(page.locator('#pwa-installability-prompt-status')).toContainText('outcome=accepted', { timeout: 10000 });

    await page.evaluate(() => {
      window.dispatchEvent(new Event('appinstalled'));
    });
    await expect(page.locator('#pwa-installability-reasons')).toContainText(/installed display mode|already running/i, { timeout: 10000 });

    await page.getByRole('button', { name: 'Update service worker', exact: true }).click();
    await expect(page.locator('#pwa-installability-sw-status')).toContainText(/update requested|update failed|Requesting service worker update|registration failed|not ready yet/i, { timeout: 10000 });
  });
});