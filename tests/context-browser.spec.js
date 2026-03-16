const { test, expect } = require('@playwright/test');

test('context api renders and updates in the wasm app', async ({ page }) => {
  await page.goto('/static/index.html', { waitUntil: 'networkidle' });

  await page.waitForSelector('#app.loaded');

  const body = page.locator('body');
  await expect(body).toContainText('Context API Example');
  await expect(body).toContainText('Theme default outside provider: light');
  await expect(body).toContainText('Theme from GoUseContext: midnight | Density: compact');
  await expect(body).toContainText('Theme from Consumer: midnight');
  await expect(body).toContainText('Nested provider theme: nested-override');
  await expect(body).toContainText('Current provider theme: midnight');

  await page.getByRole('button', { name: 'Toggle provider theme' }).click();

  await expect(body).toContainText('Theme from GoUseContext: sunrise | Density: compact');
  await expect(body).toContainText('Theme from Consumer: sunrise');
  await expect(body).toContainText('Current provider theme: sunrise');
  await expect(body).toContainText('Nested provider theme: nested-override');
});