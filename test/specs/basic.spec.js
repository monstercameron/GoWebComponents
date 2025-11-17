import { test, expect } from '@playwright/test';

test.describe('GoWebComponents Basic Tests', () => {
  test('WASM loads successfully', async ({ page }) => {
    await page.goto('/');
    
    // Wait for WASM to load (loading bar should disappear)
    await expect(page.locator('#loading-container')).toBeHidden({ timeout: 30000 });
    
    // App should be visible
    await expect(page.locator('#app')).toBeVisible();
  });

  test.skip('WASM binary is served correctly', async ({ page }) => {
    // Skipped: Direct WASM navigation triggers download in browser
    const response = await page.goto('/main.wasm');
    expect(response.status()).toBe(200);
    expect(response.headers()['content-type']).toContain('application/wasm');
  });

  test('wasm_exec.js is loaded', async ({ page }) => {
    await page.goto('/');
    
    // Check that Go runtime is available
    const hasGo = await page.evaluate(() => typeof Go !== 'undefined');
    expect(hasGo).toBeTruthy();
  });

  test('DOM renders from WASM', async ({ page }) => {
    await page.goto('/');
    
    // Wait for WASM to load
    await page.waitForSelector('#app', { state: 'visible', timeout: 30000 });
    
    // Check that WASM has rendered content into #app
    const appContent = await page.locator('#app').innerHTML();
    expect(appContent.trim()).not.toBe('');
  });

  test.skip('page title updates after WASM loads', async ({ page }) => {
    // Skipped: Test app doesn't change document title
    await page.goto('/');
    
    // Initial title
    const initialTitle = await page.title();
    
    // Wait for WASM to load and update title
    await page.waitForFunction(
      () => !document.title.includes('Loading'),
      { timeout: 30000 }
    );
    
    const finalTitle = await page.title();
    expect(finalTitle).not.toContain('Loading');
    expect(finalTitle).not.toBe(initialTitle);
  });
});

test.describe('GoWebComponents DOM Tests', () => {
  test('renders basic HTML elements', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { state: 'visible', timeout: 30000 });
    
    // Check for common HTML elements that should be rendered
    const hasDiv = await page.locator('#app div').count();
    expect(hasDiv).toBeGreaterThan(0);
  });

  test.skip('console shows expected panic from unimplemented render', async ({ page }) => {
    // Skipped: render is now implemented
    const errors = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        errors.push(msg.text());
      }
    });
    
    await page.goto('/');
    await page.waitForTimeout(2000); // Wait for WASM to try to render
    
    // Should have an error about render.To not being implemented
    const hasRenderError = errors.some(e => 
      e.includes('render.To') || e.includes('not yet implemented')
    );
    expect(hasRenderError).toBeTruthy();
  });
});
