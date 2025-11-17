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
    
    const button = page.locator('button').first();
    
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
    
    const button = page.locator('button').first();
    
    // Rapid clicks
    const clicks = 10;
    for (let i = 0; i < clicks; i++) {
      await button.click({ delay: 10 });
    }
    
    // Wait for all renders to complete (some renders are batched)
    await page.waitForTimeout(1000);
    
    // Should show final count
    const countText = await page.locator('p').first().textContent();
    expect(countText).toContain('Count: 10');
  });

  test('memory usage is stable', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });

    // Check if performance.memory API is available
    const hasMemory = await page.evaluate(() => !!(window.performance && window.performance.memory));
    if (!hasMemory) {
      // Skip test when memory API isn't available in this environment
      test.skip(true, 'performance.memory not supported in this environment');
      return;
    }

    const initial = await page.evaluate(() => performance.memory.usedJSHeapSize);

    // Perform a large number of updates
    const button = page.locator('button').first();
    for (let i = 0; i < 200; i++) {
      await button.click();
    }
    await page.waitForTimeout(500);

    const final = await page.evaluate(() => performance.memory.usedJSHeapSize);

    // Allow some growth but avoid large leaks; allow 30% growth margin
    expect(final).toBeLessThan(initial * 1.3);
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

  test('invalid props are handled', async ({ page }) => {
    // Ensure invalid props do not crash the app or log uncaught errors
    const errors = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        errors.push(msg.text());
      }
    });

    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    // BadProps component renders with invalid props
    await page.waitForSelector('#bad-props', { timeout: 30000 });
    await page.waitForTimeout(100);

    // Verify the element exists and no uncaught errors are in console
    const text = await page.locator('#bad-props').textContent();
    expect(text).toContain('BadProps');
    const critical = errors.filter(e => !e.includes('favicon') && !e.includes('DevTools'));
    expect(critical).toHaveLength(0);
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
  test('buttons have accessible text', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    const button = page.locator('button').first();
    const text = await button.textContent();
    
    expect(text.trim()).toBeTruthy();
    expect(text.length).toBeGreaterThan(0);
  });

  test('semantic HTML is used', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    await page.waitForSelector('#main-heading', { timeout: 30000 });
    const heading = page.locator('#main-heading').first();
    await expect(heading).toBeVisible();
    const headingText = await heading.textContent();
    expect(headingText).toContain('GoWebComponents');
  });

  test('ARIA attributes are present where needed', async ({ page }) => {
    await page.goto('/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // Check increment button has aria-label
    await expect(page.locator('button[aria-label="Increment main"]').first()).toBeVisible();

    // Check reusable components region has proper role and label
    const region = page.locator('#reusable-components[role="region"][aria-label="Reusable Counters Section"]');
    await expect(region).toBeVisible();

    // Check reactivity demo section region
    const reactRegion = page.locator('div[role="region"][aria-label="Reactivity Demo Section"]');
    await expect(reactRegion).toBeVisible();
  });
});
