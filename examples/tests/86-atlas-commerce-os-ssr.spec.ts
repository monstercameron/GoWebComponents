import { expect, test, type Page } from '@playwright/test';

async function signInMockSession(page: Page, nextPath = '/app/dashboard') {
  await page.goto(`/auth/mock-sign-in?next=${encodeURIComponent(nextPath)}`);
  await expect(page).toHaveTitle(/Atlas Mock Sign In/);

  const signInForm = page.locator('form').filter({ hasText: 'Inventory Manager' }).first();
  await signInForm.getByRole('button', { name: 'Start session', exact: true }).click();

  await expect(page).toHaveURL(new RegExp(nextPath.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')));
}

test.describe('86-Atlas Commerce OS SSR server', () => {
  test('hydrates once and keeps client takeover on link navigation', async ({ page }) => {
    let mainFrameNavigations = 0;
    page.on('request', request => {
      if (request.isNavigationRequest() && request.frame() === page.mainFrame()) {
        mainFrameNavigations += 1;
      }
    });

    await page.goto('/shop/frame-desk');
    await page.waitForLoadState('networkidle');

    const probe = await page.evaluate(() => {
      (window as Window & { __atlasNavProbe?: string }).__atlasNavProbe = 'atlas-client-takeover';
      return (window as Window & { __atlasNavProbe?: string }).__atlasNavProbe;
    });
    const navigationsBeforeClick = mainFrameNavigations;

    await page.getByRole('link', { name: 'Warehouses', exact: true }).click();

    await expect(page).toHaveURL(/\/warehouses$/);
    await expect(page.getByRole('heading', { name: 'Sorted products and live volume, without warehouse picking.', exact: true })).toBeVisible();
    await expect
      .poll(() =>
        page.evaluate(() => (window as Window & { __atlasNavProbe?: string }).__atlasNavProbe ?? ''),
      )
      .toBe(probe);
    expect(mainFrameNavigations).toBe(navigationsBeforeClick);
  });

  test('navigates from catalog to product detail without reloading', async ({ page }) => {
    let mainFrameNavigations = 0;
    page.on('request', request => {
      if (request.isNavigationRequest() && request.frame() === page.mainFrame()) {
        mainFrameNavigations += 1;
      }
    });

    await page.goto('/shop');
    await page.waitForLoadState('networkidle');

    const probe = await page.evaluate(() => {
      (window as Window & { __atlasNavProbe?: string }).__atlasNavProbe = 'atlas-product-takeover';
      return (window as Window & { __atlasNavProbe?: string }).__atlasNavProbe;
    });
    const navigationsBeforeClick = mainFrameNavigations;

    await page.getByRole('link', { name: /Frame Bench/i }).first().click();

    await expect(page).toHaveURL(/\/shop\/frame-bench$/);
    await expect(page).toHaveTitle(/Atlas Frame Bench/);
    await expect(page.getByRole('heading', { name: 'Atlas Frame Bench', exact: true })).toBeVisible();
    await expect
      .poll(() =>
        page.evaluate(() => (window as Window & { __atlasNavProbe?: string }).__atlasNavProbe ?? ''),
      )
      .toBe(probe);
    expect(mainFrameNavigations).toBe(navigationsBeforeClick);
  });

  test('returns from warehouses to the landing page without keeping stale route content', async ({ page }) => {
    let mainFrameNavigations = 0;
    page.on('request', request => {
      if (request.isNavigationRequest() && request.frame() === page.mainFrame()) {
        mainFrameNavigations += 1;
      }
    });

    await page.goto('/');
    await page.waitForLoadState('networkidle');

    const probe = await page.evaluate(() => {
      (window as Window & { __atlasNavProbe?: string }).__atlasNavProbe = 'atlas-public-home-return';
      return (window as Window & { __atlasNavProbe?: string }).__atlasNavProbe;
    });
    const navigationsBeforeClick = mainFrameNavigations;

    await page.getByRole('link', { name: 'Warehouses', exact: true }).click();
    await expect(page).toHaveURL(/\/warehouses$/);
    await expect(page.getByRole('heading', { name: 'Atlas Product Volume', exact: true })).toBeVisible();

    await page.getByRole('link', { name: 'Storefront', exact: true }).click();

    await expect(page).toHaveURL(/\/$/);
    await expect(page).toHaveTitle(/Atlas Commerce OS/);
    await expect(page.getByRole('heading', { name: 'Atlas Commerce OS', exact: true })).toBeVisible();
    await expect(page.getByText('A darker, cleaner storefront for complex workspace buying.', { exact: true })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Atlas Product Volume', exact: true })).toHaveCount(0);
    await expect(page.getByText('Sorted products and live volume, without warehouse picking.', { exact: true })).toHaveCount(0);
    await expect
      .poll(() =>
        page.evaluate(() => (window as Window & { __atlasNavProbe?: string }).__atlasNavProbe ?? ''),
      )
      .toBe(probe);
    expect(mainFrameNavigations).toBe(navigationsBeforeClick);
  });

  test('navigates from internal warehouse route to storefront home without reloading', async ({ page }) => {
    let mainFrameNavigations = 0;
    page.on('request', request => {
      if (request.isNavigationRequest() && request.frame() === page.mainFrame()) {
        mainFrameNavigations += 1;
      }
    });

    await signInMockSession(page, '/app/warehouses');
    await page.waitForLoadState('networkidle');

    const probe = await page.evaluate(() => {
      (window as Window & { __atlasNavProbe?: string }).__atlasNavProbe = 'atlas-home-takeover';
      return (window as Window & { __atlasNavProbe?: string }).__atlasNavProbe;
    });
    const navigationsBeforeClick = mainFrameNavigations;

    await page.getByRole('link', { name: 'Storefront', exact: true }).click();

    await expect(page).toHaveURL(/\/$/);
    await expect(page).toHaveTitle(/Atlas Commerce OS/);
    await expect(page.getByRole('heading', { name: 'Atlas Commerce OS', exact: true })).toBeVisible();
    await expect(page.getByText('A darker, cleaner storefront for complex workspace buying.', { exact: true })).toBeVisible();
    await expect
      .poll(() =>
        page.evaluate(() => (window as Window & { __atlasNavProbe?: string }).__atlasNavProbe ?? ''),
      )
      .toBe(probe);
    expect(mainFrameNavigations).toBe(navigationsBeforeClick);
  });

  test('serves direct SSR product routes with hydration bootstrap', async ({ page }) => {
    await page.goto('/shop');

    await expect(page).toHaveTitle(/Atlas Shop/);
    await expect(page.getByText('View availability options', { exact: true }).first()).toBeVisible();

    await page.goto('/shop/frame-desk');

    await expect(page).toHaveTitle(/Atlas Frame Desk/);
    await expect(page.getByRole('heading', { name: 'Atlas Frame Desk', exact: true })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Frame Desk', exact: true })).toBeVisible();
    await expect(page.getByText('Reserve upcoming availability', { exact: true })).toBeVisible();
    await expect(page.getByText('Customer reviews and questions', { exact: true })).toBeVisible();
    await expect(page.getByText(/thumbs up/i).first()).toBeVisible();
    await expect(page.getByText('Share your review or question', { exact: true })).toBeVisible();

    await page.goto('/warehouses/new-jersey-hub');

    await expect(page).toHaveTitle(/Atlas Warehouse Detail/);
    await expect(page.getByText('Regional availability picks', { exact: true })).toBeVisible();
    await expect(page.getByText('Browse products in this region', { exact: true })).toBeVisible();
    await expect(page.getByText('View delivery options', { exact: true }).first()).toBeVisible();

    await expect(page.locator('script#__ATLAS_BOOTSTRAP__')).toHaveCount(1);
    await expect(page.locator('#app')).not.toContainText('Atlas SSR Preview');
    await expect(page).toHaveURL(/\/warehouses\/new-jersey-hub$/);
  });

  test('serves internal SSR entry routes', async ({ page }) => {
    await signInMockSession(page, '/app/dashboard');
    await page.goto('/app/dashboard');

    await expect(page).toHaveURL(/\/app\/dashboard$/);
    await expect(page).toHaveTitle(/Atlas Ops Dashboard/);
    await expect(page.getByRole('heading', { name: 'Atlas Ops Dashboard', exact: true })).toBeVisible();
    await expect(page.locator('script#__ATLAS_BOOTSTRAP__')).toHaveCount(1);

    await page.goto('/app/warehouses');
    await expect(page).toHaveTitle(/Atlas Warehouses Internal/);
    await expect(page.getByText('Warehouse network', { exact: true })).toBeVisible();
    await expect(page.getByText('New Jersey Hub', { exact: true })).toBeVisible();

    await page.goto('/app/warehouses/new-jersey-hub');
    await expect(page.getByText('Search items', { exact: true })).toBeVisible();
    await expect(page.getByText('Warehouse items', { exact: true }).first()).toBeVisible();
    await expect(page.locator('main a[href^="/shop"]')).toHaveCount(0);
    await expect(page.locator('form[action="/api/app/products"] select[name="warehouse_id"]')).toHaveCount(0);
    await expect(page.locator('form[action="/api/app/products"] input[name="warehouse_id"][type="hidden"][value="new-jersey-hub"]')).toHaveCount(1);
    await expect(page.locator('form[action="/api/app/purchase-orders"] select[name="warehouse_id"]')).toHaveCount(0);
    await expect(page.locator('form[action="/api/app/purchase-orders"] input[name="warehouse_id"][type="hidden"][value="new-jersey-hub"]')).toHaveCount(1);

    await page.goto('/app/warehouses/new-jersey-hub/items/frame-desk');
    await expect(page.getByText('Market and sales readout', { exact: true })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Save warehouse item', exact: true })).toBeVisible();
    await expect(page.locator('main a[href^="/shop"]')).toHaveCount(0);
    await expect(page.locator('form[action="/api/app/products/frame-desk/update"] select[name="warehouse_id"]')).toHaveCount(0);
    await expect(page.locator('form[action="/api/app/products/frame-desk/update"] input[name="warehouse_id"][type="hidden"][value="new-jersey-hub"]')).toHaveCount(1);
    await expect(page.locator('form[action="/api/app/purchase-orders"] select[name="warehouse_id"]')).toHaveCount(0);
    await expect(page.locator('form[action="/api/app/purchase-orders"] input[name="warehouse_id"][type="hidden"][value="new-jersey-hub"]')).toHaveCount(1);

    await page.goto('/app/transfers/tr-seed-001');
    await expect(page).toHaveTitle(/Atlas Transfer Detail/);
    await expect(page.getByText('Transfer lines', { exact: true })).toBeVisible();
    await expect(page.getByText('frame-desk', { exact: true })).toBeVisible();

    await page.goto('/app/purchase-orders/po-1042');
    await expect(page).toHaveTitle(/Atlas Purchase Order Detail/);
    await expect(page.getByText('Northline Fabrication', { exact: true })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Update order', exact: true })).toBeVisible();

    await page.goto('/app/receiving/rcv-illinois-001');
    await expect(page).toHaveTitle(/Atlas Receiving Session/);
    await expect(page.getByText('Receiving lines', { exact: true })).toBeVisible();
    await expect(page.getByText(/cable-bridge/)).toBeVisible();

    await page.goto('/app/comments');
    await expect(page).toHaveTitle(/Atlas Buyer Follow-Up/);
    await expect(page.locator('main')).toContainText('Cable routing depth');

    await page.goto('/app/settings');
    await expect(page).toHaveTitle(/Atlas Settings/);
    await expect(page.getByRole('button', { name: 'Save preferences', exact: true })).toBeVisible();
    await expect(page.getByText('Low stock triage', { exact: true })).toBeVisible();
  });

  test('submits public and internal SSR form flows against the native server', async ({ page }) => {
    await page.goto('/shop/frame-desk');
    const publicRouteBeforeSubmit = page.url();

    const commentForm = page.locator('form[action="/api/public/products/frame-desk/comments"]');
    await commentForm.getByLabel('Name').fill('Playwright Operator');
    await commentForm.getByLabel('Thumbs up').check();
    await commentForm.getByLabel('Headline').fill('Lead time');
    await commentForm.getByLabel('Comment').fill('Can Atlas hold a Friday install window for this desk?');
    await commentForm.getByRole('button', { name: 'Share feedback', exact: true }).click();

    await expect(page).toHaveURL(/\/shop\/frame-desk$/);
    await expect(page).not.toHaveURL(/atlas_notice=comment-submitted/);
    await expect(commentForm.getByText('Your comment was submitted for review.', { exact: true })).toBeVisible();
    await expect(page.getByText('Lead time', { exact: true })).toBeVisible();
    await expect(page.getByText('Can Atlas hold a Friday install window for this desk?', { exact: true })).toBeVisible();
    await expect(page.getByText('pending', { exact: true })).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Atlas Frame Desk', exact: true })).toBeVisible();
    expect(page.url()).toBe(publicRouteBeforeSubmit);

    await signInMockSession(page, '/app/dashboard');
    await page.goto('/app/dashboard');
    await expect(page).toHaveTitle(/Atlas Ops Dashboard/);
    await expect(page.getByRole('heading', { name: 'Atlas Ops Dashboard', exact: true })).toBeVisible();

    const preferencesForm = page.locator('form[action="/api/app/preferences"]');
    await preferencesForm.getByLabel('Theme').fill('light');
    await preferencesForm.getByLabel('Locale').fill('fr');
    await preferencesForm.getByLabel('Density').fill('comfortable');
    await preferencesForm.getByLabel('Default warehouse').fill('illinois-hub');
    await preferencesForm.getByRole('button', { name: 'Save preferences', exact: true }).click();

    await expect(page).toHaveURL(/atlas_notice=preferences-saved/);
    await expect(page.getByText('preferences-saved', { exact: true })).toBeVisible();
    await expect(page.getByText(/^Locale$/, { exact: true }).first().locator('xpath=..')).toContainText('fr');
    await expect(page.getByText(/^Density$/, { exact: true }).first().locator('xpath=..')).toContainText('comfortable');
    await expect(page.locator('form[action="/api/app/preferences"]').getByLabel('Default warehouse')).toHaveValue('illinois-hub');

    await page.goto('/app/purchase-orders/po-1042');
    await expect(page).toHaveTitle(/Atlas Purchase Order Detail/);
    const purchaseOrderForm = page.locator('form[action="/api/app/purchase-orders/po-1042/status"]');
    await purchaseOrderForm.getByLabel('Status').fill('approved');
    await purchaseOrderForm.getByLabel('Note').fill('SSR purchase-order approval check.');
    await purchaseOrderForm.getByRole('button', { name: 'Update order', exact: true }).click();

    await expect(page).toHaveURL(/atlas_notice=purchase-order-updated/);
    await expect(page.getByText('purchase-order-updated', { exact: true })).toBeVisible();
    await expect(page.locator('form[action="/api/app/purchase-orders/po-1042/status"]').getByLabel('Status')).toHaveValue('approved');

    await page.goto('/app/receiving/rcv-illinois-001');
    await expect(page).toHaveTitle(/Atlas Receiving Session/);
    const receivingForm = page.locator('form[action="/api/app/receiving/rcv-illinois-001/reconcile"]');
    await receivingForm.getByLabel('Status').fill('closed');
    await receivingForm.getByLabel('Discrepancy summary').fill('SSR receiving reconciliation check.');
    await receivingForm.getByRole('button', { name: 'Close session', exact: true }).click();

    await expect(page).toHaveURL(/atlas_notice=receiving-reconciled/);
    await expect(page.getByText('receiving-reconciled', { exact: true })).toBeVisible();
    await expect(page.locator('main')).toContainText('SSR receiving reconciliation check.');
  });
});