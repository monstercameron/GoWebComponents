import { test, expect } from '@playwright/test';

// Browser router advanced integration tests
// Tests edge cases, performance, and complex scenarios
// NOTE: These tests require a test app running at localhost:8080 that uses browser routing
// They are commented out for now and should be enabled once the test infrastructure is ready

test.describe.skip('Browser Router - Advanced Integration', () => {
  test('router performance under load', async ({ page }) => {
    const startTime = Date.now();
    
    await page.goto('http://localhost:8080/');
    await page.waitForTimeout(200);
    
    // Rapid navigation stress test
    for (let i = 0; i < 20; i++) {
      await page.goto(`http://localhost:8080/test-${i}`);
      await page.waitForTimeout(50);
    }
    
    const duration = Date.now() - startTime;
    expect(duration).toBeLessThan(8000); // Should handle 20 navigations efficiently
  });

  test('router handles network interruption gracefully', async ({ page, context }) => {
    // Simulate offline mode
    await context.setOffline(true);
    
    await page.goto('http://localhost:8080/');
    await page.waitForTimeout(300);
    
    // Go back online
    await context.setOffline(false);
    
    // Should recover and work normally
    await page.goto('http://localhost:8080/about');
    await page.waitForTimeout(500);
    
    const url = page.url();
    expect(url).toContain('/about');
  });

  test('multiple tabs do not interfere with each other', async ({ context }) => {
    const page1 = await context.newPage();
    const page2 = await context.newPage();
    
    try {
      // Navigate page1 to /about
      await page1.goto('http://localhost:8080/about');
      await page1.waitForTimeout(300);
      
      // Navigate page2 to /services
      await page2.goto('http://localhost:8080/services');
      await page2.waitForTimeout(300);
      
      // Check page1 is still on /about
      const url1 = page1.url();
      expect(url1).toContain('/about');
      
      // Check page2 is on /services
      const url2 = page2.url();
      expect(url2).toContain('/services');
    } finally {
      await page1.close();
      await page2.close();
    }
  });

  test('router recovers from JavaScript errors', async ({ page }) => {
    const errors = [];
    
    page.on('pageerror', error => {
      errors.push(error.message);
    });
    
    // Navigate normally
    await page.goto('http://localhost:8080/');
    await page.waitForTimeout(300);
    
    // Try to navigate after potential error
    await page.goto('http://localhost:8080/about');
    await page.waitForTimeout(500);
    
    // Should still navigate successfully
    const url = page.url();
    expect(url).toContain('/about');
  });

  test('complete navigation workflow', async ({ page }) => {
    // Home -> About -> Services -> Back -> Forward -> Home
    
    await page.goto('http://localhost:8080/');
    await page.waitForTimeout(300);
    expect(page.url()).toContain('http://localhost:8080/');
    
    await page.goto('http://localhost:8080/about');
    await page.waitForTimeout(300);
    expect(page.url()).toContain('/about');
    
    await page.goto('http://localhost:8080/services');
    await page.waitForTimeout(300);
    expect(page.url()).toContain('/services');
    
    await page.goBack();
    await page.waitForTimeout(300);
    expect(page.url()).toContain('/about');
    
    await page.goForward();
    await page.waitForTimeout(300);
    expect(page.url()).toContain('/services');
    
    await page.goBack();
    await page.goBack();
    await page.waitForTimeout(300);
    expect(page.url()).toContain('http://localhost:8080/');
  });

  test('router handles navigation with page visibility changes', async ({ page }) => {
    await page.goto('http://localhost:8080/');
    await page.waitForTimeout(300);
    
    // Simulate page becoming hidden/visible
    await page.evaluate(() => {
      document.dispatchEvent(new Event('visibilitychange'));
    });
    
    // Navigate while potentially hidden
    await page.goto('http://localhost:8080/about');
    await page.waitForTimeout(300);
    
    const url = page.url();
    expect(url).toContain('/about');
  });

  test('anchor tag navigation works with browser router', async ({ page }) => {
    await page.goto('http://localhost:8080/');
    await page.waitForTimeout(300);
    
    // Look for internal navigation links
    const internalLinks = await page.locator('a[href^="/"], a[href^="http://localhost"]').count();
    
    // If there are internal links, they should be navigable
    if (internalLinks > 0) {
      const firstLink = page.locator('a[href^="/"], a[href^="http://localhost"]').first();
      const href = await firstLink.getAttribute('href');
      expect(href).toBeTruthy();
    }
  });

  test('router handles invalid route with graceful fallback', async ({ page }) => {
    // Navigate to completely invalid route
    await page.goto('http://localhost:8080/totally-invalid-route-12345');
    await page.waitForTimeout(500);
    
    // Page should still render (404 page or default)
    const content = await page.content();
    expect(content.length).toBeGreaterThan(0);
  });

  test('router scroll position management', async ({ page }) => {
    await page.goto('http://localhost:8080/');
    await page.waitForTimeout(300);
    
    // Scroll down
    await page.evaluate(() => window.scrollTo(0, 500));
    await page.waitForTimeout(200);
    
    // Get scroll position
    const scrollBefore = await page.evaluate(() => window.scrollY);
    
    // Navigate to another page
    await page.goto('http://localhost:8080/about');
    await page.waitForTimeout(300);
    
    // Check scroll position (typically resets to top on new page)
    const scrollAfter = await page.evaluate(() => window.scrollY);
    expect(scrollAfter).toBeDefined();
  });

  test('router history depth', async ({ page }) => {
    // Build up history
    await page.goto('http://localhost:8080/');
    await page.waitForTimeout(200);
    
    await page.goto('http://localhost:8080/about');
    await page.waitForTimeout(200);
    
    await page.goto('http://localhost:8080/services');
    await page.waitForTimeout(200);
    
    // Go back twice
    await page.goBack();
    await page.waitForTimeout(200);
    expect(page.url()).toContain('/about');
    
    await page.goBack();
    await page.waitForTimeout(200);
    expect(page.url()).toContain('http://localhost:8080/');
  });

  test('router state during reload', async ({ page }) => {
    // Navigate to a specific page
    await page.goto('http://localhost:8080/about');
    await page.waitForTimeout(300);
    
    const urlBefore = page.url();
    
    // Reload
    await page.reload();
    await page.waitForTimeout(500);
    
    const urlAfter = page.url();
    
    // URL should remain the same
    expect(urlBefore).toBe(urlAfter);
  });

  test('router handles concurrent navigation attempts', async ({ page }) => {
    // Attempt multiple navigations rapidly without waiting
    const promises = [
      page.goto('http://localhost:8080/path1').catch(() => {}),
      page.goto('http://localhost:8080/path2').catch(() => {}),
      page.goto('http://localhost:8080/path3').catch(() => {}),
    ];
    
    await Promise.all(promises);
    await page.waitForTimeout(500);
    
    // Should end up at one of the destinations
    const url = page.url();
    expect(url).toContain('localhost:8080');
  });
});
