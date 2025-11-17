import { test, expect } from '@playwright/test';

test.describe('GoWebComponents - Debug Rendering', () => {
  test('check console for errors', async ({ page }) => {
    const errors = [];
    const logs = [];
    
    page.on('console', msg => {
      logs.push(`${msg.type()}: ${msg.text()}`);
      if (msg.type() === 'error') {
        errors.push(msg.text());
      }
    });
    
    page.on('pageerror', error => {
      errors.push(`PAGE ERROR: ${error.message}`);
    });
    
    await page.goto('/');
    await page.waitForTimeout(5000); // Wait for WASM to load
    
    console.log('=== Console Logs ===');
    logs.forEach(log => console.log(log));
    
    console.log('\n=== Errors ===');
    errors.forEach(error => console.log(error));
    
    console.log('\n=== Page HTML ===');
    const html = await page.locator('#app').innerHTML();
    console.log(html);
  });

  test('check if Go runtime loaded', async ({ page }) => {
    await page.goto('/');
    await page.waitForTimeout(2000);
    
    const hasGo = await page.evaluate(() => typeof Go !== 'undefined');
    expect(hasGo).toBeTruthy();
    
    console.log('Go runtime is loaded');
  });

  test('check if WASM initialized', async ({ page }) => {
    const logs = [];
    page.on('console', msg => logs.push(msg.text()));
    
    await page.goto('/');
    await page.waitForTimeout(3000);
    
    const hasInitLog = logs.some(log => log.includes('WASM initialized'));
    console.log('All logs:', logs);
    console.log('Has init log:', hasInitLog);
  });

  test('check button onclick property', async ({ page }) => {
    const logs = [];
    page.on('console', msg => logs.push(`${msg.type()}: ${msg.text()}`));
    
    await page.goto('/');
    await page.waitForTimeout(3000);
    
    const buttonInfo = await page.evaluate(() => {
      const button = document.querySelector('button');
      return {
        exists: !!button,
        text: button?.textContent,
        hasOnclick: button?.onclick !== null && button?.onclick !== undefined,
        onclickType: typeof button?.onclick,
        onclickValue: button?.onclick?.toString(),
      };
    });
    
    console.log('Button info:', JSON.stringify(buttonInfo, null, 2));
    
    // Get paragraph text before click
    const beforeText = await page.locator('p').textContent();
    console.log('\n=== Before click ===');
    console.log('Paragraph text:', beforeText);
    
    // Click the button and wait for logs
    await page.click('button');
    await page.waitForTimeout(1000);
    
    // Get paragraph text after click
    const afterText = await page.locator('p').textContent();
    console.log('\n=== After click ===');
    console.log('Paragraph text:', afterText);
    console.log('Text changed:', beforeText !== afterText);
    
    console.log('\n=== Logs after click ===');
    logs.forEach(log => console.log(log));
    
    const clickLogs = logs.filter(log => log.includes('Button clicked') || log.includes('Setting count'));
    console.log('\n=== Click-related logs ===');
    console.log(clickLogs.length > 0 ? clickLogs : 'No click logs found');
  });
});
