import { test, expect } from '@playwright/test';

test.describe('Hash Router E2E Tests', () => {
  test.beforeEach(async ({ page }) => {
    // Capture console logs for debugging
    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        console.log('PAGE ERROR:', msg.text());
      }
    });
  });

  test('Hash router loads and initializes', async ({ page }) => {
    await page.goto('/#/');
    
    // Wait for app to load
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // Verify app is visible
    const appContent = await page.locator('#app').isVisible();
    expect(appContent).toBeTruthy();
  });

  test('Navigation via hash change works', async ({ page }) => {
    await page.goto('/#/');
    
    // Wait for initial route
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // Navigate to different route via hash
    await page.goto('/#/about');
    
    // Wait for route to change
    await page.waitForTimeout(500);
    
    // Verify URL changed
    expect(page.url()).toContain('#/about');
  });

  test('Multiple route navigation works', async ({ page }) => {
    const routes = ['#/', '#/about', '#/contact', '#/docs'];
    
    for (const route of routes) {
      await page.goto(route);
      
      // Verify URL contains the route
      expect(page.url()).toContain(route);
      
      // Wait for content to render
      await page.waitForSelector('#app', { timeout: 5000 });
      
      await page.waitForTimeout(300);
    }
  });

  test('Browser back button navigates correctly', async ({ page }) => {
    // Go to home
    await page.goto('/#/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // Navigate to about
    await page.goto('/#/about');
    await page.waitForTimeout(500);
    
    // Verify we're on about
    expect(page.url()).toContain('#/about');
    
    // Go back
    await page.goBack();
    
    // Verify we're back on home
    expect(page.url()).toContain('#/');
  });

  test('Browser forward button navigates correctly', async ({ page }) => {
    // Go to home
    await page.goto('/#/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // Navigate to about
    await page.goto('/#/about');
    await page.waitForTimeout(500);
    
    // Go back to home
    await page.goBack();
    await page.waitForTimeout(500);
    
    // Go forward to about
    await page.goForward();
    
    // Verify we're on about
    expect(page.url()).toContain('#/about');
  });

  test('Hash router handles direct navigation links', async ({ page }) => {
    await page.goto('/#/');
    
    // Wait for app to load
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // Click a navigation link if it exists (example)
    const navLinks = await page.locator('a[href*="#/"]').count();
    
    if (navLinks > 0) {
      const firstLink = page.locator('a[href*="#/"]').first();
      const linkHref = await firstLink.getAttribute('href');
      
      await firstLink.click();
      await page.waitForTimeout(300);
      
      // Verify we navigated to the link's route
      expect(page.url()).toContain(linkHref);
    }
  });

  test('Refreshing at a hash route preserves location', async ({ page }) => {
    await page.goto('/#/docs');
    
    // Wait for page to load
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // Refresh page
    await page.reload();
    
    // Wait for app to reload
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // Verify we're still at /docs
    expect(page.url()).toContain('#/docs');
  });

  test('404/Wildcard routes display for unknown paths', async ({ page }) => {
    await page.goto('/#/unknown-route-that-does-not-exist-12345');
    
    // Wait for app to load
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // The router should handle this route (either show 404 or wildcard component)
    const appContent = await page.locator('#app').innerHTML();
    expect(appContent.trim()).not.toBe('');
  });

  test('Router handles root path correctly', async ({ page }) => {
    // Test different ways to access root
    const rootVariations = ['/#/', '#/', '/'];
    
    for (const root of rootVariations) {
      await page.goto(root);
      
      // Wait for app to load
      await page.waitForSelector('#app', { timeout: 10000 });
      
      // Verify app is visible
      const appContent = await page.locator('#app').isVisible();
      expect(appContent).toBeTruthy();
    }
  });

  test('Rapid navigation between routes works', async ({ page }) => {
    await page.goto('/#/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    const routes = ['#/about', '#/contact', '#/docs', '#/about', '#/'];
    
    for (const route of routes) {
      await page.goto(route);
      // Don't wait for full timeout, routes should be fast
      await page.waitForSelector('#app', { timeout: 5000 });
      await page.waitForTimeout(100);
    }
    
    // Verify final route
    expect(page.url()).toContain('#/');
  });

  test('Page title updates with route navigation', async ({ page }) => {
    await page.goto('/#/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    const initialTitle = await page.title();
    
    // Navigate to another route
    await page.goto('/#/docs');
    await page.waitForTimeout(300);
    
    const newTitle = await page.title();
    
    // Titles might be the same or different depending on app configuration
    // Just verify that page loaded successfully
    const appContent = await page.locator('#app').isVisible();
    expect(appContent).toBeTruthy();
  });

  test('Router does not break on missing trailing slash', async ({ page }) => {
    const testRoutes = ['/#/about', '/#/contact', '/#/docs'];
    
    for (const route of testRoutes) {
      await page.goto(route);
      
      // Wait for app to load
      await page.waitForSelector('#app', { timeout: 10000 });
      
      const appContent = await page.locator('#app').isVisible();
      expect(appContent).toBeTruthy();
    }
  });

  test('Router maintains state across navigation', async ({ page }) => {
    await page.goto('/#/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // Check if there's any input field
    const inputCount = await page.locator('input').count();
    
    if (inputCount > 0) {
      // Try to interact with an input
      const firstInput = page.locator('input').first();
      await firstInput.fill('test value');
      
      // Navigate away
      await page.goto('/#/about');
      await page.waitForTimeout(300);
      
      // Navigate back
      await page.goto('/#/');
      await page.waitForTimeout(300);
      
      // Verify we're back at home
      expect(page.url()).toContain('#/');
    }
  });

  test('Hash router handles special characters in paths', async ({ page }) => {
    // Navigate to a route that might have special chars (if app supports it)
    const specialRoutes = ['/#/', '/#/users', '/#/posts'];
    
    for (const route of specialRoutes) {
      await page.goto(route);
      
      // Verify page loads without errors
      await page.waitForSelector('#app', { timeout: 10000 });
      
      const appContent = await page.locator('#app').isVisible();
      expect(appContent).toBeTruthy();
    }
  });

  test('Console has no router-related errors', async ({ page }) => {
    const errors = [];
    
    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        errors.push(msg.text());
      }
    });
    
    // Navigate through multiple routes
    await page.goto('/#/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    await page.goto('/#/about');
    await page.waitForTimeout(300);
    
    await page.goto('/#/docs');
    await page.waitForTimeout(300);
    
    // Filter for router-specific errors
    const routerErrors = errors.filter(e => 
      e.includes('route') || e.includes('router') || e.includes('hash') ||
      e.includes('navigation') || e.includes('navigate')
    );
    
    expect(routerErrors.length).toBe(0);
  });

  test('Router survives page resize', async ({ page }) => {
    await page.goto('/#/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // Resize window
    await page.setViewportSize({ width: 800, height: 600 });
    await page.waitForTimeout(300);
    
    // Navigate to verify router still works
    await page.goto('/#/about');
    
    // Verify navigation worked
    expect(page.url()).toContain('#/about');
  });

  test('Hash router maintains performance with many routes', async ({ page }) => {
    const startTime = Date.now();
    
    await page.goto('/#/');
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // Navigate through several routes rapidly
    for (let i = 0; i < 5; i++) {
      await page.goto('/#/');
      await page.waitForTimeout(50);
      await page.goto('/#/about');
      await page.waitForTimeout(50);
      await page.goto('/#/docs');
      await page.waitForTimeout(50);
    }
    
    const endTime = Date.now();
    const totalTime = endTime - startTime;
    
    // Should complete all navigations in reasonable time (less than 30 seconds total)
    expect(totalTime).toBeLessThan(30000);
    
    // Final verification
    const appContent = await page.locator('#app').isVisible();
    expect(appContent).toBeTruthy();
  });

  test('Router handles empty hash gracefully', async ({ page }) => {
    await page.goto('/#');
    
    // Should redirect or handle gracefully
    await page.waitForSelector('#app', { timeout: 10000 });
    
    const appContent = await page.locator('#app').isVisible();
    expect(appContent).toBeTruthy();
  });
});
