const { chromium } = require('@playwright/test');

async function expectText(page, text) {
  await page.waitForFunction(
    expected => document.body && document.body.innerText.includes(expected),
    text,
    { timeout: 15000 }
  );
}

async function main() {
  const browser = await chromium.launch();

  try {
    const page = await browser.newPage();
    await page.goto('http://127.0.0.1:8081/static/index.html', { waitUntil: 'networkidle' });

    await page.waitForSelector('#app.loaded', { timeout: 15000 });
    await expectText(page, 'Context API Example');
    await expectText(page, 'Theme default outside provider: light');
    await expectText(page, 'Theme from GoUseContext: midnight | Density: compact');
    await expectText(page, 'Theme from Consumer: midnight');
    await expectText(page, 'Nested provider theme: nested-override');
    await expectText(page, 'Current provider theme: midnight');

    await page.getByRole('button', { name: 'Toggle provider theme' }).click();

    await expectText(page, 'Theme from GoUseContext: sunrise | Density: compact');
    await expectText(page, 'Theme from Consumer: sunrise');
    await expectText(page, 'Current provider theme: sunrise');
    await expectText(page, 'Nested provider theme: nested-override');

    console.log('Context browser smoke test passed');
  } finally {
    await browser.close();
  }
}

main().catch(err => {
  console.error(err);
  process.exit(1);
});