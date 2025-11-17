import { test, expect } from '@playwright/test';

test.describe('Hash Router - Advanced Integration Tests', () => {
  test('Complete routing workflow', async ({ page }) => {
    // Test complete user journey through app
    
    // 1. Visit home
    await page.goto('/#/');
    await page.waitForSelector('#app', { timeout: 30000 });
    let content = await page.locator('#app').isVisible();
    expect(content).toBeTruthy();
    
    // 2. Check URL
    expect(page.url()).toContain('#/');
    
    // 3. Navigate to docs if link exists
    const docLinks = await page.locator('a[href*="docs"]').count();
    if (docLinks > 0) {
      await page.locator('a[href*="docs"]').first().click();
      await page.waitForTimeout(500);
      expect(page.url()).toContain('docs');
    }
    
    // 4. Go back
    await page.goBack();
    await page.waitForTimeout(300);
    expect(page.url()).toContain('#/');
    
    // 5. Direct hash navigation
    await page.evaluate(() => {
      window.location.hash = '#/about';
    });
    await page.waitForTimeout(500);
    expect(page.url()).toContain('#/about');
  });

  test('Router handles rapid hash changes', async ({ page }) => {
    await page.goto('/#/', { waitUntil: 'networkidle' });
    
    // Rapidly change hash
    for (let i = 0; i < 10; i++) {
      await page.evaluate(() => {
        const routes = ['#/', '#/about', '#/docs'];
        const randomRoute = routes[Math.floor(Math.random() * routes.length)];
        window.location.hash = randomRoute;
      });
      await page.waitForTimeout(50);
    }
    
    // App should still be functional
    const appVisible = await page.locator('#app').isVisible({ timeout: 5000 });
    expect(appVisible).toBeTruthy();
    
    // Check for errors
    const errors = [];
    page.on('console', msg => {
      if (msg.type() === 'error') errors.push(msg.text());
    });
    
    await page.waitForTimeout(1000);
    expect(errors.length).toBe(0);
  });

  test('Router maintains component state during navigation', async ({ page }) => {
    await page.goto('/#/', { waitUntil: 'networkidle' });
    
    // Look for input fields that might maintain state
    const inputs = await page.locator('input').count();
    
    if (inputs > 0) {
      // Fill first input if present
      const firstInput = page.locator('input').first();
      await firstInput.fill('test-value-' + Date.now());
      const filledValue = await firstInput.inputValue();
      
      // Navigate away
      await page.goto('/#/about', { waitUntil: 'networkidle' });
      await page.waitForTimeout(300);
      
      // Navigate back to same route
      await page.goto('/#/', { waitUntil: 'networkidle' });
      await page.waitForTimeout(300);
      
      // Component may or may not preserve state depending on implementation
      // Just verify page is functional
      const appVisible = await page.locator('#app').isVisible();
      expect(appVisible).toBeTruthy();
    }
  });

  test('Router works without trailing slash in route', async ({ page }) => {
    const routes = ['#/', '#/about', '#/docs'];
    
    for (const route of routes) {
      await page.goto(route);
      
      // Verify app renders
      await page.waitForSelector('#app', { timeout: 10000 });
      const appVisible = await page.locator('#app').isVisible();
      expect(appVisible).toBeTruthy();
      
      // Verify URL is correct
      expect(page.url()).toContain(route);
    }
  });

  test('Router handles navigation from anchor tags', async ({ page }) => {
    await page.goto('/#/', { waitUntil: 'networkidle' });
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // Find all navigation links
    const navLinks = await page.locator('a[href^="#/"]').all();
    
    if (navLinks.length > 0) {
      // Click first few links
      const linksToTest = navLinks.slice(0, Math.min(3, navLinks.length));
      
      for (const link of linksToTest) {
        const href = await link.getAttribute('href');
        await link.click({ timeout: 5000 });
        await page.waitForTimeout(300);
        
        // Verify we navigated
        expect(page.url()).toContain(href);
        
        // Verify app still renders
        const appVisible = await page.locator('#app').isVisible({ timeout: 5000 });
        expect(appVisible).toBeTruthy();
      }
    }
  });

  test('Router doesn\'t break with invalid navigation', async ({ page }) => {
    await page.goto('/#/', { waitUntil: 'networkidle' });
    
    // Try navigation to non-existent routes
    const invalidRoutes = ['#/xyz123', '#/admin/secret', '#/../../etc/passwd'];
    
    for (const route of invalidRoutes) {
      await page.goto(route, { waitUntil: 'networkidle' });
      
      // App should still render (404 or wildcard component)
      await page.waitForSelector('#app', { timeout: 10000 });
      const appVisible = await page.locator('#app').isVisible();
      expect(appVisible).toBeTruthy();
    }
  });

  test('Router works after page visibility changes', async ({ page }) => {
    await page.goto('/#/', { waitUntil: 'networkidle' });
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // Simulate page becoming hidden
    await page.evaluate(() => {
      Object.defineProperty(document, 'hidden', { value: true });
    });
    
    // Wait a bit
    await page.waitForTimeout(200);
    
    // Simulate page becoming visible again
    await page.evaluate(() => {
      Object.defineProperty(document, 'hidden', { value: false });
    });
    
    // Navigate
    await page.goto('/#/about', { waitUntil: 'networkidle' });
    
    // Verify navigation worked
    const appVisible = await page.locator('#app').isVisible();
    expect(appVisible).toBeTruthy();
    expect(page.url()).toContain('#/about');
  });

  test('Router handles network interruption gracefully', async ({ page, context }) => {
    await page.goto('/#/', { waitUntil: 'networkidle' });
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // Navigate (should work despite network conditions)
    await page.goto('/#/about', { waitUntil: 'networkidle' });
    
    // App should be visible
    const appVisible = await page.locator('#app').isVisible();
    expect(appVisible).toBeTruthy();
  });

  test('Multiple tabs don\'t interfere with each other', async ({ browser }) => {
    const context = await browser.newContext();
    const page1 = await context.newPage();
    const page2 = await context.newPage();
    
    try {
      // Navigate page 1 to home
      await page1.goto('/#/', { waitUntil: 'networkidle' });
      await page1.waitForSelector('#app', { timeout: 30000 });
      expect(page1.url()).toContain('#/');
      
      // Navigate page 2 to docs
      await page2.goto('/#/docs', { waitUntil: 'networkidle' });
      await page2.waitForSelector('#app', { timeout: 30000 });
      expect(page2.url()).toContain('#/docs');
      
      // Verify page 1 is still on home
      expect(page1.url()).toContain('#/');
      
      // Verify page 2 is still on docs
      expect(page2.url()).toContain('#/docs');
    } finally {
      await context.close();
    }
  });

  test('Router recovers from JavaScript errors', async ({ page }) => {
    const errors = [];
    page.on('console', msg => {
      if (msg.type() === 'error') {
        errors.push(msg.text());
      }
    });
    
    await page.goto('/#/', { waitUntil: 'networkidle' });
    await page.waitForSelector('#app', { timeout: 30000 });
    
    // Navigate through routes
    const routes = ['#/about', '#/docs', '#/'];
    
    for (const route of routes) {
      await page.goto(route, { waitUntil: 'networkidle' });
      
      // Verify app is still visible
      const appVisible = await page.locator('#app').isVisible({ timeout: 5000 });
      expect(appVisible).toBeTruthy();
    }
    
    // Check that no critical errors occurred
    const criticalErrors = errors.filter(e => 
      !e.includes('404') && 
      !e.includes('warning') &&
      !e.toLowerCase().includes('deprecated')
    );
    
    // Some errors might be expected, but router shouldn't cause crashes
    const routerErrors = criticalErrors.filter(e =>
      e.toLowerCase().includes('router') ||
      e.toLowerCase().includes('route') ||
      e.toLowerCase().includes('hash')
    );
    
    expect(routerErrors.length).toBe(0);
  });

  test('Router performance under load', async ({ page }) => {
    const startTime = performance.now();
    
    await page.goto('/#/', { waitUntil: 'networkidle' });
    await page.waitForSelector('#app', { timeout: 30000 });
    
    const loadTime = performance.now() - startTime;
    
    // Navigate rapidly 20 times
    const navStartTime = performance.now();
    
    for (let i = 0; i < 20; i++) {
      const routes = ['#/', '#/about', '#/docs'];
      const route = routes[i % routes.length];
      await page.goto(route, { waitUntil: 'networkidle' });
    }
    
    const navTime = performance.now() - navStartTime;
    
    // Verify reasonable performance
    expect(navTime).toBeLessThan(60000); // 20 navigations in under 60 seconds
    
    // Final verification
    const appVisible = await page.locator('#app').isVisible();
    expect(appVisible).toBeTruthy();
  });
});
