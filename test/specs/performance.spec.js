import { test, expect } from '@playwright/test';

test.describe('GoWebComponents - Performance', () => {
  test('WASM loads within reasonable time', async ({ page }) => {
    const startTime = Date.now();
    
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    const loadTime = Date.now() - startTime;
    
    // Should load in less than 5 seconds
    expect(loadTime).toBeLessThan(5000);
  });

  test('UI updates are fast', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    const button = page.locator('button');
    
    const startTime = Date.now();
    await button.click();
    await page.waitForSelector('p:has-text("Count: 1")', { timeout: 1000 });
    const updateTime = Date.now() - startTime;
    
    // Update should happen in less than 500ms
    expect(updateTime).toBeLessThan(500);
  });

  test('multiple rapid clicks are handled', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    const button = page.locator('button');
    
    // Rapid clicks
    const clicks = 10;
    for (let i = 0; i < clicks; i++) {
      await button.click({ delay: 10 });
    }
    
    await page.waitForTimeout(200);
    
    // Should show final count
    const countText = await page.locator('p').textContent();
    expect(countText).toContain('Count: 10');
  });

  test.skip('memory usage is stable', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // TODO: Monitor memory usage over time
  });
});

test.describe('GoWebComponents - Error Handling', () => {
  test('app handles errors gracefully', async ({ page }) => {
    const errors = [];
    page.on('pageerror', error => errors.push(error.message));
    
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // Should have no uncaught errors
    expect(errors).toHaveLength(0);
  });

  test.skip('invalid props are handled', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // TODO: Test error boundaries or prop validation
  });
});

test.describe('GoWebComponents - Browser Compatibility', () => {
  test('console has no errors', async ({ page }) => {
    const errors = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        errors.push(msg.text());
      }
    });
    
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // Filter out known non-critical warnings
    const criticalErrors = errors.filter(e => 
      !e.includes('favicon') && 
      !e.includes('DevTools')
    );
    
    expect(criticalErrors).toHaveLength(0);
  });

  test('no network errors', async ({ page }) => {
    const failedRequests = [];
    page.on('requestfailed', request => {
      failedRequests.push(request.url());
    });
    
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // Filter out non-critical failures (favicon, etc.)
    const criticalFailures = failedRequests.filter(url => 
      url.includes('.wasm') || url.includes('wasm_exec.js')
    );
    
    expect(criticalFailures).toHaveLength(0);
  });
});

test.describe('GoWebComponents - Accessibility', () => {
  test.skip('buttons have accessible text', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    const button = page.locator('button');
    const text = await button.textContent();
    
    expect(text.trim()).toBeTruthy();
    expect(text.length).toBeGreaterThan(0);
  });

  test.skip('semantic HTML is used', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // Check for semantic elements
    const hasHeading = await page.locator('h1, h2, h3').count();
    expect(hasHeading).toBeGreaterThan(0);
  });

  test.skip('ARIA attributes are present where needed', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // TODO: Test ARIA attributes
  });
});
