import { test, expect, Page, Route } from '@playwright/test';

type MockUser = {
  id: number;
  name: string;
  username: string;
  email: string;
  website: string;
};

const users: MockUser[] = [
  {
    id: 1,
    name: 'Alice Async',
    username: 'alice',
    email: 'alice@example.test',
    website: 'alice.test',
  },
  {
    id: 2,
    name: 'Bob Boundary',
    username: 'bob',
    email: 'bob@example.test',
    website: 'bob.test',
  },
];

async function fulfillJSON(route: Route, body: unknown, delayMs = 0, status = 200) {
  if (delayMs > 0) {
    await new Promise(resolve => setTimeout(resolve, delayMs));
  }

  await route.fulfill({
    status,
    contentType: 'application/json',
    body: JSON.stringify(body),
  });
}

async function fulfillText(route: Route, body: string, delayMs = 0, status = 200) {
  if (delayMs > 0) {
    await new Promise(resolve => setTimeout(resolve, delayMs));
  }

  await route.fulfill({
    status,
    contentType: 'text/plain; charset=utf-8',
    body,
  });
}

async function installFetchMocks(page: Page, options?: { failUser2First?: boolean }) {
  let user2Attempts = 0;

  await page.route('https://jsonplaceholder.typicode.com/users', async route => {
    await fulfillJSON(route, users, 150);
  });

  await page.route(/https:\/\/jsonplaceholder\.typicode\.com\/users\/\d+$/, async route => {
    const url = new URL(route.request().url());
    const id = Number(url.pathname.split('/').pop());
    const user = users.find(entry => entry.id === id);

    if (!user) {
      await fulfillJSON(route, { error: 'missing user' }, 0, 404);
      return;
    }

    if (id === 2 && options?.failUser2First && user2Attempts === 0) {
      user2Attempts += 1;
      await fulfillText(route, 'temporary detail failure', 250, 500);
      return;
    }

    if (id === 2) {
      user2Attempts += 1;
    }

    await fulfillJSON(route, user, 450);
  });
}

test.describe('08-Fetch', () => {
  test('renders async boundaries and deferred lazy content', async ({ page }) => {
    await installFetchMocks(page);

    await page.goto('/08-fetch/fetch.html');

    await expect(page.getByRole('heading', { name: 'User Directory', exact: true })).toBeVisible({ timeout: 15000 });
    await expect(page.getByRole('button', { name: 'Reload Resources', exact: true })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Cancel In-Flight Work', exact: true })).toBeVisible();

    await expect(page.getByText('Alice Async', { exact: true })).toBeVisible();
    await expect(page.getByText('Bob Boundary', { exact: true })).toBeVisible();
    await expect(page.getByText('Username: @alice', { exact: true })).toBeVisible();
    await expect(page.getByText('Async UI primitive demo', { exact: true })).toBeVisible();

    await page.getByRole('button', { name: /Bob Boundary/ }).click();
    await expect(page.getByText('Username: @bob', { exact: true })).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Async UI primitive demo', { exact: true })).toBeVisible();
  });

  test('renders detail error fallback and retries successfully', async ({ page }) => {
    await installFetchMocks(page, { failUser2First: true });

    await page.goto('/08-fetch/fetch.html');

    await expect(page.getByText('Alice Async', { exact: true })).toBeVisible({ timeout: 15000 });

    await page.getByRole('button', { name: /Bob Boundary/ }).click();
    await expect(page.getByText('Detail error:', { exact: false })).toBeVisible({ timeout: 10000 });
    await expect(page.getByRole('button', { name: 'Retry Detail', exact: true })).toBeVisible();

    await page.getByRole('button', { name: 'Retry Detail', exact: true }).click();
    await expect(page.getByText('Username: @bob', { exact: true })).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('Async UI primitive demo', { exact: true })).toBeVisible();
  });
});