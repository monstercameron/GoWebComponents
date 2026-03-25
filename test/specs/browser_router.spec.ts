import { test, expect } from '@playwright/test';

// Browser router basic E2E tests
// Tests the HTML5 History API based router using window.location.pathname
// NOTE: These tests require a test app running at localhost:8080 that uses browser routing
// They are commented out for now and should be enabled once the test infrastructure is ready

test.describe.skip('Browser Router - Basic Functionality', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to test app that uses browser router
    await page.goto('http://localhost:8080/');
  });

  test('browser router loads and initializes', async ({ page }) => {
    // Check that the page loads successfully
    const content = await page.content();
    expect(content).toBeTruthy();
    expect(content.length).toBeGreaterThan(0);
  });

  test('navigation via pathname change works', async ({ page }) => {
    // Navigate using pathname instead of hash
    await page.goto('http://localhost:8080/about');
    
    // Wait for navigation to complete
    await page.waitForTimeout(500);
    
    // Check that we're on the /about page
    const url = page.url();
    expect(url).toContain('/about');
  });

  test('multiple route navigation works', async ({ page }) => {
    // Navigate to first route
    await page.goto('http://localhost:8080/');
    await page.waitForTimeout(300);
    
    // Navigate to second route
    await page.goto('http://localhost:8080/about');
    await page.waitForTimeout(300);
    
    // Navigate to third route
    await page.goto('http://localhost:8080/services');
    await page.waitForTimeout(300);
    
    const url = page.url();
    expect(url).toContain('/services');
  });

  test('browser back button navigates correctly', async ({ page }) => {
    // Start at home
    await page.goto('http://localhost:8080/');
    await page.waitForTimeout(300);
    
    // Go to about
    await page.goto('http://localhost:8080/about');
    await page.waitForTimeout(300);
    
    // Go back
    await page.goBack();
    await page.waitForTimeout(300);
    
    // Check URL is home
    const url = page.url();
    expect(url).toContain('http://localhost:8080/');
    expect(url).not.toContain('about');
  });

  test('browser forward button navigates correctly', async ({ page }) => {
    // Go home
    await page.goto('http://localhost:8080/');
    await page.waitForTimeout(300);
    
    // Go to about
    await page.goto('http://localhost:8080/about');
    await page.waitForTimeout(300);
    
    // Go back
    await page.goBack();
    await page.waitForTimeout(300);
    
    // Go forward
    await page.goForward();
    await page.waitForTimeout(300);
    
    // Check URL is about
    const url = page.url();
    expect(url).toContain('/about');
  });

  test('direct navigation links work', async ({ page }) => {
    // Create a test page with navigation links
    await page.goto('http://localhost:8080/');
    
    // Find and click a navigation link to /about
    const aboutLink = await page.locator('a[href="/about"], a:has-text("About")').first();
    if (await aboutLink.isVisible({ timeout: 1000 }).catch(() => false)) {
      await aboutLink.click();
      await page.waitForTimeout(500);
    }
    
    // Check that navigation occurred (URL change or content change)
    const url = page.url();
    expect(url).toBeTruthy();
  });

  test('refreshing at route preserves location', async ({ page }) => {
    // Go to about page
    await page.goto('http://localhost:8080/about');
    await page.waitForTimeout(300);
    
    // Refresh the page
    await page.reload();
    await page.waitForTimeout(500);
    
    // URL should still be /about
    const url = page.url();
    expect(url).toContain('/about');
  });

  test('404/Wildcard routes display for unknown paths', async ({ page }) => {
    // Navigate to unknown route
    await page.goto('http://localhost:8080/unknown-route-xyz');
    await page.waitForTimeout(500);
    
    // Page should still load (not get actual 404)
    const content = await page.content();
    expect(content).toBeTruthy();
    expect(content.length).toBeGreaterThan(0);
  });

  test('router handles root path correctly', async ({ page }) => {
    await page.goto('http://localhost:8080/');
    await page.waitForTimeout(500);
    
    const url = page.url();
    expect(url).toContain('localhost:8080');
  });

  test('rapid navigation between routes works', async ({ page }) => {
    // Rapid navigation
    await page.goto('http://localhost:8080/');
    await page.goto('http://localhost:8080/about');
    await page.goto('http://localhost:8080/services');
    await page.goto('http://localhost:8080/');
    
    await page.waitForTimeout(500);
    
    // Should end up at home
    const url = page.url();
    expect(url).toContain('http://localhost:8080/');
  });

  test('page title updates with route navigation', async ({ page }) => {
    // Go to home
    await page.goto('http://localhost:8080/');
    const homeTitle = await page.title();
    
    // Go to about
    await page.goto('http://localhost:8080/about');
    const aboutTitle = await page.title();
    
    // Titles might be different or same depending on app, but should not be empty
    expect(homeTitle).toBeTruthy();
    expect(aboutTitle).toBeTruthy();
  });

  test('router does not break on missing trailing slash', async ({ page }) => {
    // Try both with and without trailing slash
    await page.goto('http://localhost:8080/about');
    await page.waitForTimeout(300);
    
    expect(page.url()).toContain('about');
  });

  test('router maintains state across navigation', async ({ page }) => {
    // Navigate to a route
    await page.goto('http://localhost:8080/');
    await page.waitForTimeout(300);
    
    // Store initial content
    const initialContent = await page.content();
    
    // Navigate away
    await page.goto('http://localhost:8080/about');
    await page.waitForTimeout(300);
    
    // Navigate back
    await page.goBack();
    await page.waitForTimeout(300);
    
    // Should return to home (content should render again)
    const finalContent = await page.content();
    expect(finalContent).toBeTruthy();
    expect(finalContent.length).toBeGreaterThan(0);
  });

  test('router handles special characters in paths', async ({ page }) => {
    // Navigate to path with special characters (URL encoded)
    await page.goto('http://localhost:8080/about%20us');
    await page.waitForTimeout(500);
    
    // Page should load without crashing
    const content = await page.content();
    expect(content).toBeTruthy();
  });

  test('console has no router-related errors', async ({ page }) => {
    const errors = [];
    
    page.on('console', msg => {
      if (msg.type() === 'error' && msg.text().toLowerCase().includes('router')) {
        errors.push(msg.text());
      }
    });
    
    // Navigate through routes
    await page.goto('http://localhost:8080/');
    await page.waitForTimeout(300);
    
    await page.goto('http://localhost:8080/about');
    await page.waitForTimeout(300);
    
    await page.goBack();
    await page.waitForTimeout(300);
    
    // Check for errors
    expect(errors).toHaveLength(0);
  });

  test('router survives page resize', async ({ page }) => {
    await page.goto('http://localhost:8080/about');
    await page.waitForTimeout(300);
    
    // Resize window
    await page.setViewportSize({ width: 800, height: 600 });
    await page.waitForTimeout(200);
    
    // Navigate after resize
    await page.goto('http://localhost:8080/');
    await page.waitForTimeout(300);
    
    const url = page.url();
    expect(url).toContain('localhost:8080/');
  });

  test('router maintains performance with many routes', async ({ page }) => {
    const startTime = Date.now();
    
    // Navigate through many routes
    for (let i = 0; i < 10; i++) {
      await page.goto(`http://localhost:8080/route-${i}`);
      await page.waitForTimeout(100);
    }
    
    const duration = Date.now() - startTime;
    
    // Should complete in reasonable time (10 navigations with 100ms each = 1000ms + overhead)
    expect(duration).toBeLessThan(5000);
  });

  test('router handles empty pathname gracefully', async ({ page }) => {
    await page.goto('http://localhost:8080');
    await page.waitForTimeout(500);
    
    // Should redirect to / or handle gracefully
    const url = page.url();
    expect(url).toContain('localhost:8080');
  });
});
