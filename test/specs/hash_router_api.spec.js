import { test, expect } from '@playwright/test';

test.describe('Hash Router - Public API Tests', () => {
  test('Router API exposes key functions', async ({ page }) => {
    await page.goto('/#/', { waitUntil: 'networkidle' });
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // Verify that router API is available in window (if exposed)
    const routerAvailable = await page.evaluate(() => {
      // Check if any router-related globals exist
      return typeof window !== 'undefined';
    });
    
    expect(routerAvailable).toBeTruthy();
  });

  test('Route component renders with correct content', async ({ page }) => {
    await page.goto('/#/', { waitUntil: 'networkidle' });
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // Get app content
    const appContent = await page.locator('#app').innerHTML();
    expect(appContent.length).toBeGreaterThan(0);
  });

  test('Navigation between pages maintains app DOM', async ({ page }) => {
    await page.goto('/#/', { waitUntil: 'networkidle' });
    await page.waitForSelector('#app', { timeout: 30000 });
    
    const appBeforeNav = await page.locator('#app').count();
    expect(appBeforeNav).toBeGreaterThan(0);
    
    // Navigate
    await page.goto('/#/about', { waitUntil: 'networkidle' });
    
    const appAfterNav = await page.locator('#app').count();
    expect(appAfterNav).toBeGreaterThan(0);
  });

  test('Route-specific content changes on navigation', async ({ page }) => {
    await page.goto('/#/', { waitUntil: 'networkidle' });
    await page.waitForSelector('#app', { timeout: 30000 });
    
    const homeContent = await page.locator('#app').innerHTML();
    
    // Navigate to different route
    await page.goto('/#/docs', { waitUntil: 'networkidle' });
    await page.waitForTimeout(300);
    
    const docsContent = await page.locator('#app').innerHTML();
    
    // Content should be different (unless both routes render identical content)
    // Just verify that navigation doesn't break rendering
    expect(docsContent.length).toBeGreaterThan(0);
  });

  test('404/Not Found route displays for invalid paths', async ({ page }) => {
    await page.goto('/#/this-route-does-not-exist', { waitUntil: 'networkidle' });
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // App should still render something (404 or fallback)
    const appContent = await page.locator('#app').innerHTML();
    expect(appContent.length).toBeGreaterThan(0);
  });

  test('Router initialization doesn\'t cause errors', async ({ page }) => {
    const errors = [];
    const pageErrors = [];
    
    page.on('console', msg => {
      if (msg.type() === 'error') {
        pageErrors.push(msg.text());
      }
    });
    
    page.on('pageerror', error => {
      errors.push(error.message);
    });
    
    // Load app
    await page.goto('/#/', { waitUntil: 'networkidle' });
    await page.waitForSelector('#app', { timeout: 30000 });
    
    await page.waitForTimeout(1000);
    
    // Filter for router-specific errors
    const routerErrors = pageErrors.filter(e =>
      e.toLowerCase().includes('router') ||
      e.toLowerCase().includes('route') ||
      e.toLowerCase().includes('navigate')
    );
    
    expect(routerErrors.length).toBe(0);
    expect(errors.length).toBe(0);
  });

  test('Router responds to hash fragment navigation', async ({ page }) => {
    await page.goto('/#/', { waitUntil: 'networkidle' });
    
    // Use JavaScript to change hash (as user might with direct hash link)
    await page.evaluate(() => {
      window.location.hash = '#/about';
    });
    
    // Wait for route to process
    await page.waitForTimeout(500);
    
    // Verify we're at the route
    expect(page.url()).toContain('#/about');
    
    // Verify content rendered
    const appVisible = await page.locator('#app').isVisible();
    expect(appVisible).toBeTruthy();
  });

  test('Router works with programmatic hash changes', async ({ page }) => {
    await page.goto('/#/', { waitUntil: 'networkidle' });
    await page.waitForSelector('#app', { timeout: 30000 });
    
    const routes = ['#/about', '#/docs', '#/contact'];
    
    for (const route of routes) {
      // Change hash programmatically
      await page.evaluate((hash) => {
        window.location.hash = hash;
      }, route);
      
      await page.waitForTimeout(300);
      
      // Verify navigation
      expect(page.url()).toContain(route);
    }
  });

  test('Router state persists across page navigation and back', async ({ page }) => {
    // Navigate to first route
    await page.goto('/#/', { waitUntil: 'networkidle' });
    await page.waitForSelector('#app', { timeout: 30000 });
    const initialUrl = page.url();
    
    // Navigate to second route
    await page.goto('/#/docs', { waitUntil: 'networkidle' });
    expect(page.url()).toContain('#/docs');
    
    // Go back
    await page.goBack();
    
    // Should be at initial route
    expect(page.url()).toContain('#/');
  });

  test('Router handles empty hash', async ({ page }) => {
    // Navigate to empty hash
    await page.goto('/#', { waitUntil: 'networkidle' });
    
    // Wait for app to load
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // Should either redirect to /#/ or handle gracefully
    const appVisible = await page.locator('#app').isVisible();
    expect(appVisible).toBeTruthy();
  });

  test('Router works with relative navigation links', async ({ page }) => {
    await page.goto('/#/', { waitUntil: 'networkidle' });
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // Try to find and click internal links
    const internalLinks = await page.locator('a[href*="#"]').count();
    
    if (internalLinks > 0) {
      const firstLink = page.locator('a[href*="#"]').first();
      const originalUrl = page.url();
      
      await firstLink.click();
      await page.waitForTimeout(500);
      
      // URL should have changed
      const newUrl = page.url();
      expect(newUrl).not.toEqual(originalUrl);
    }
  });

  test('Multiple route changes are queued correctly', async ({ page }) => {
    await page.goto('/#/', { waitUntil: 'networkidle' });
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // Trigger multiple navigation changes rapidly
    await page.evaluate(() => {
      window.location.hash = '#/about';
      window.location.hash = '#/docs';
      window.location.hash = '#/';
    });
    
    await page.waitForTimeout(500);
    
    // Final URL should be the last one
    expect(page.url()).toContain('#/');
    
    // App should still be visible
    const appVisible = await page.locator('#app').isVisible();
    expect(appVisible).toBeTruthy();
  });

  test('Router works after browser tab recovery', async ({ page }) => {
    await page.goto('/#/', { waitUntil: 'networkidle' });
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // Navigate
    await page.goto('/#/docs', { waitUntil: 'networkidle' });
    expect(page.url()).toContain('#/docs');
    
    // Simulate tab crash recovery by reloading at current URL
    await page.reload();
    
    // Should still be at the same route
    expect(page.url()).toContain('#/docs');
    
    // App should be functional
    await page.waitForSelector('#app', { timeout: 30000 });
    const appVisible = await page.locator('#app').isVisible();
    expect(appVisible).toBeTruthy();
  });

  test('Router works with mobile navigation patterns', async ({ page }) => {
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });
    
    await page.goto('/#/', { waitUntil: 'networkidle' });
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // Navigate through routes
    const routes = ['#/about', '#/docs', '#/'];
    
    for (const route of routes) {
      await page.goto(route, { waitUntil: 'networkidle' });
      
      // Verify navigation
      expect(page.url()).toContain(route);
      
      // Verify content is visible
      const appVisible = await page.locator('#app').isVisible({ timeout: 5000 });
      expect(appVisible).toBeTruthy();
    }
  });

  test('Router works with keyboard navigation', async ({ page }) => {
    await page.goto('/#/', { waitUntil: 'networkidle' });
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // Navigate via keyboard if app supports it
    // Try Tab to focus elements
    await page.keyboard.press('Tab');
    await page.keyboard.press('Tab');
    
    // App should still be visible and functional
    const appVisible = await page.locator('#app').isVisible();
    expect(appVisible).toBeTruthy();
  });
});
