/**
 * End-to-end tests for the AI Chat Wizard (example 100).
 *
 * Tests are intentionally serial because the server holds a single-writer
 * SQLite instance.  Four areas are covered:
 *
 *   1. Initial load — WASM boots and the page is ready.
 *   2. Conversation history list — the two seeded items appear in the sidebar.
 *   3. Load a conversation — clicking a sidebar entry replays its messages.
 *   4. Sidebar toggle — the ◀/▶ collapse controls hide and restore the list.
 *   5. New chat — clears the active conversation, sidebar list persists.
 *   6. Delete conversation — confirm modal, item removed from sidebar.
 */
import { test, expect, type Page } from '@playwright/test';

// ── helpers ──────────────────────────────────────────────────────────────────

/**
 * Navigate to the chat page and wait until the WASM app has mounted.
 * We consider the app ready when either:
 *  a) the sidebar "New chat" button appears (happy path), or
 *  b) the #app element has visible text content (fallback).
 *
 * The WASM can take a few seconds on a cold start because the browser must
 * download, compile, and instantiate the ~24 MB binary.
 */
async function loadChatPage(page: Page): Promise<void> {
  await page.goto('/');

  const newChatButton = page.getByRole('button', { name: 'New chat' });
  const emailField = page.getByPlaceholder('you@example.com');
  await Promise.race([
    newChatButton.waitFor({ state: 'visible', timeout: 60_000 }).catch(() => null),
    emailField.waitFor({ state: 'visible', timeout: 60_000 }).catch(() => null),
  ]);

  if (await emailField.isVisible().catch(() => false)) {
    await emailField.fill('demo@example.com');
    await page.getByPlaceholder('Enter a password').fill('password123');
    await page.getByRole('button', { name: /Log in|Sign in/i }).click();
  }

  // Wait for wasm_exec.js + wasm instantiation to complete.
  // The "New chat" button is the first interactive element rendered by the app.
  await expect(newChatButton).toBeVisible({ timeout: 60_000 });
}

/**
 * Wait until the sidebar conversation list is populated (not showing the
 * "No conversations yet" placeholder) and contains at least one item.
 */
async function waitForConversationList(page: Page, timeout = 15_000): Promise<void> {
  // The sidebar renders "No conversations yet" when the gRPC list is empty.
  // Once ListConversations returns, that placeholder is replaced by buttons.
  await expect(
    page.getByText('No conversations yet'),
  ).toBeHidden({ timeout });
}

function visibleToolbarSelect(page: Page, index: number) {
  return page.locator('.toolbar-select:visible').nth(index);
}

// ── test suite ───────────────────────────────────────────────────────────────

test.describe('100-AI-Chat-Wizard — chat history', () => {
  test.describe.configure({ mode: 'serial' });

  // ── 1. Initial page load ──────────────────────────────────────────────────

  test('page loads and WASM mounts', async ({ page }) => {
    await loadChatPage(page);

    // Branding
    await expect(page.getByText('RelayDesk').first()).toBeVisible();

    // Sidebar controls
    await expect(page.getByRole('button', { name: 'New chat' })).toBeVisible();

    // Input area
    await expect(page.locator('#chat-input')).toBeVisible();

    // Empty-state placeholder — app is ready but no active conversation
    await expect(page.getByText('Explore the Go WASM UI experiment')).toBeVisible();
  });

  // ── 2. Conversation history list populates ────────────────────────────────

  test('sidebar shows seeded conversation history', async ({ page }) => {
    await loadChatPage(page);
    await waitForConversationList(page);

    // The seed script creates two conversations with these titles.
    // The server's ListConversations returns them newest-first.
    await expect(
      page.getByRole('button', { name: /WebAssembly and Go/i }).first(),
    ).toBeVisible({ timeout: 10_000 });

    await expect(
      page.getByRole('button', { name: /Golang Goroutines Explained/i }).first(),
    ).toBeVisible({ timeout: 5_000 });
  });

  // ── 3. Load a conversation ────────────────────────────────────────────────

  test('clicking a conversation loads its messages', async ({ page }) => {
    await loadChatPage(page);
    await waitForConversationList(page);

    // Click the goroutines conversation
    await page
      .getByRole('button', { name: /Golang Goroutines Explained/i })
      .first()
      .click();

    // The user message from the seed should appear in the chat area
    await expect(
      page.getByText('How do goroutines work in Go?'),
    ).toBeVisible({ timeout: 10_000 });

    // The assistant reply should also appear
    await expect(
      page.getByText(/Goroutines are lightweight threads/i),
    ).toBeVisible({ timeout: 5_000 });

    // Empty state should be gone
    await expect(page.getByText('Explore the Go WASM UI experiment')).toBeHidden();
  });

  test('canvas fenced blocks render in the live preview pane', async ({ page }) => {
    await loadChatPage(page);
    await waitForConversationList(page);

    await page
      .getByRole('button', { name: /Canvas Preview Demo/i })
      .first()
      .click();

    await expect(page.locator('#canvas-preview-pane')).toBeVisible({ timeout: 10_000 });
    await expect(page.locator('#canvas-preview-pane iframe')).toBeVisible();
    await expect(page.frameLocator('#canvas-preview-pane iframe').locator('#canvas')).toBeVisible();
    await expect(page.frameLocator('#canvas-preview-pane iframe').locator('.demo-card')).toContainText('Canvas demo');
  });

  test('loading a conversation shows message and thread costs', async ({ page }) => {
    await loadChatPage(page);
    await waitForConversationList(page);

    await page
      .getByRole('button', { name: /Golang Goroutines Explained/i })
      .first()
      .click();

    await expect(page.getByText('GPT-5.4 $0.00075')).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText('Thread total $0.00075')).toBeVisible({ timeout: 5_000 });
  });

  test('loading a conversation restores its model selection', async ({ page }) => {
    await loadChatPage(page);
    await waitForConversationList(page);

    const gpt54Mini = page.getByRole('button', { name: 'GPT-5.4 mini', exact: true });
    const gpt54 = page.getByRole('button', { name: 'GPT-5.4', exact: true });

    await page
      .getByRole('button', { name: /WebAssembly and Go/i })
      .first()
      .click();

    await expect(
      page.getByText('What is WebAssembly?'),
    ).toBeVisible({ timeout: 10_000 });
    await expect(gpt54Mini).toHaveClass(/(^|\s)text-black(\s|$)/);
    await expect(gpt54).not.toHaveClass(/(^|\s)text-black(\s|$)/);

    await page
      .getByRole('button', { name: /Golang Goroutines Explained/i })
      .first()
      .click();

    await expect(
      page.getByText('How do goroutines work in Go?'),
    ).toBeVisible({ timeout: 10_000 });
    await expect(gpt54).toHaveClass(/(^|\s)text-black(\s|$)/);
    await expect(gpt54Mini).not.toHaveClass(/(^|\s)text-black(\s|$)/);
  });

  // ── 4. Sidebar toggle ─────────────────────────────────────────────────────

  test('thinking selector persists inline preference', async ({ page }) => {
    await loadChatPage(page);

    const highThinking = page.locator('[data-thinkingeffort="high"]').last();
    const offThinking = page.locator('[data-thinkingeffort="off"]').last();

    await expect(highThinking).toBeVisible();
    await highThinking.click();
    await expect(highThinking).toHaveClass(/bg-\[#19c37d\]|bg-\[#19c37d\]\/20/);

    await page.reload();
    await loadChatPage(page);
    await expect(highThinking).toHaveClass(/bg-\[#19c37d\]|bg-\[#19c37d\]\/20/, { timeout: 10_000 });

    await offThinking.click();
    await expect(offThinking).toHaveClass(/bg-\[#19c37d\]|bg-\[#19c37d\]\/20/);
  });

  test('provider and model selection sync across open tabs', async ({ page, context }) => {
    await loadChatPage(page);

    const secondPage = await context.newPage();
    await loadChatPage(secondPage);

    await visibleToolbarSelect(page, 0).selectOption('cerebras');
    await visibleToolbarSelect(page, 1).selectOption('gpt-oss-120b');

    await expect(visibleToolbarSelect(secondPage, 0)).toHaveValue('cerebras', { timeout: 10_000 });
    await expect(visibleToolbarSelect(secondPage, 1)).toHaveValue('gpt-oss-120b', { timeout: 10_000 });

    await secondPage.close();
  });

  test('reasoning mode filters the current provider model list', async ({ page }) => {
    await loadChatPage(page);

    await visibleToolbarSelect(page, 0).selectOption('cerebras');
    await visibleToolbarSelect(page, 1).selectOption('gpt-oss-120b');
    await visibleToolbarSelect(page, 2).selectOption('high');

    await expect(page.getByText('Reasoning mode is on, so only reasoning-capable models are shown.').last()).toBeVisible();

    const modelValues = await visibleToolbarSelect(page, 1).evaluate((element) =>
      Array.from((element as HTMLSelectElement).options).map((option) => option.value),
    );

    expect(modelValues).toContain('gpt-oss-120b');
    expect(modelValues).toContain('zai-glm-4.7');
    expect(modelValues).not.toContain('llama3.1-8b');
    expect(modelValues).not.toContain('qwen-3-235b-a22b-instruct-2507');
  });

  test('settings locale switch translates the UI and persists across reload', async ({ page }) => {
    await loadChatPage(page);

    await page.getByRole('button', { name: /Edit settings/i }).click();
    await expect(page.locator('#name-input')).toBeVisible();

    await page.locator('button').filter({ hasText: 'Espa' }).click();
    await page.getByRole('button', { name: 'Save' }).click();

    await expect(page.getByRole('button', { name: 'Nuevo chat' })).toBeVisible({ timeout: 10_000 });
    await expect(page.getByText('Explora el experimento de UI Go WASM')).toBeVisible();
    await expect(page.locator('[data-current-locale="es"]')).toBeVisible();

    await page.reload();
    await expect(page.getByRole('button', { name: 'Nuevo chat' })).toBeVisible({ timeout: 60_000 });
    await expect(page.getByText('Explora el experimento de UI Go WASM')).toBeVisible();
    await expect(page.locator('[data-current-locale="es"]')).toBeVisible();
  });

  test('selected text can be quoted into the composer', async ({ page }) => {
    await loadChatPage(page);
    const heading = page.getByText('Explore the Go WASM UI experiment');
    await expect(heading).toBeVisible();
    await page.locator('#empty-state').evaluate(() => {
      const host = document.getElementById('empty-state');
      if (!host) {
        throw new Error('empty state host not found');
      }
      const heading = Array.from(host.querySelectorAll('*')).find((node) =>
        node.textContent?.includes('Explore the Go WASM UI experiment'),
      ) as HTMLElement | undefined;
      if (!heading) {
        throw new Error('empty state heading not found');
      }
      const rect = heading.getBoundingClientRect();
      const selectedText = 'Explore the Go WASM UI experiment';
      const selectionStub = {
        rangeCount: 1,
        isCollapsed: false,
        toString: () => selectedText,
        getRangeAt: () => ({
          getBoundingClientRect: () => rect,
        }),
        removeAllRanges: () => {},
      };
      Object.defineProperty(window, 'getSelection', {
        configurable: true,
        value: () => selectionStub,
      });
      heading.dispatchEvent(new MouseEvent('mouseup', {
        bubbles: true,
        clientX: rect.left + (rect.width / 2),
        clientY: rect.top,
      }));
    });

    await expect(page.locator('#quote-selection-spinner')).toBeVisible();
    const quoteButton = page.getByRole('button', { name: 'Quote text' });
    await expect(quoteButton).toBeVisible({ timeout: 2_000 });
    await quoteButton.click();

    await expect(page.locator('#chat-input')).toHaveValue(/> Explore the Go WASM UI experiment/);
  });

  test('sidebar collapse and expand', async ({ page }) => {
    await loadChatPage(page);
    await waitForConversationList(page);

    // The conversation titles should be visible initially
    await expect(
      page.getByRole('button', { name: /WebAssembly and Go/i }).first(),
    ).toBeInViewport();

    // Click the collapse (◀) button — it is the button containing "◀"
    await page.getByRole('button', { name: '◀' }).click();

    // After collapse the sidebar width is 0 (overflow-hidden, CSS transition).
    // The elements remain in the DOM but are no longer in the viewport.
    // Wait for the 250ms CSS transition to complete before asserting.
    await page.waitForTimeout(400);
    await expect(
      page.getByRole('button', { name: /WebAssembly and Go/i }).first(),
    ).not.toBeInViewport({ timeout: 3_000 });

    // The main panel now shows an expand (▶) button
    await expect(page.getByRole('button', { name: '▶' })).toBeVisible();

    // Click expand
    await page.getByRole('button', { name: '▶' }).click();

    // Sidebar is open again — wait for the CSS transition
    await page.waitForTimeout(400);
    await expect(
      page.getByRole('button', { name: /WebAssembly and Go/i }).first(),
    ).toBeInViewport({ timeout: 3_000 });
  });

  // ── 5. New chat button ────────────────────────────────────────────────────

  test('new chat button clears active conversation', async ({ page }) => {
    await loadChatPage(page);

    const gpt54Mini = page.getByRole('button', { name: 'GPT-5.4 mini', exact: true }).last();
    const gpt54 = page.getByRole('button', { name: 'GPT-5.4', exact: true }).last();

    await gpt54.click();
    await expect(gpt54).toHaveClass(/(^|\s)text-black(\s|$)/);
    await expect(gpt54Mini).not.toHaveClass(/(^|\s)text-black(\s|$)/);

    // Click "New chat"
    await page.getByRole('button', { name: 'New chat' }).click();

    // The empty-state placeholder re-appears
    await expect(
      page.getByText('Explore the Go WASM UI experiment'),
    ).toBeVisible({ timeout: 5_000 });

    // New draft threads reset to the middle model option.
    await expect(gpt54Mini).toHaveClass(/(^|\s)text-black(\s|$)/);
    await expect(gpt54).not.toHaveClass(/(^|\s)text-black(\s|$)/);

  });

  // ── 6. Delete a conversation ──────────────────────────────────────────────

  test('deleting a conversation removes it from the sidebar', async ({ page }) => {
    await loadChatPage(page);
    await waitForConversationList(page);

    // Hover over "WebAssembly and Go" to reveal the ✕ delete button.
    // The button is inside a .group div that shows it on hover.
    const convRow = page
      .locator('.group')
      .filter({ has: page.getByRole('button', { name: /WebAssembly and Go/i }) })
      .first();

    await convRow.hover();

    // Click the ✕ button (text content is "✕")
    const deleteBtn = convRow.getByRole('button', { name: '✕' });
    await expect(deleteBtn).toBeVisible({ timeout: 3_000 });
    await deleteBtn.click();

    // A confirmation modal should appear
    await expect(page.getByText('Delete conversation?')).toBeVisible({ timeout: 3_000 });
    await expect(page.getByText('This will permanently delete')).toBeVisible();

    // Confirm deletion
    await page.getByRole('button', { name: 'Delete' }).click();

    // Modal disappears
    await expect(page.getByText('Delete conversation?')).toBeHidden({ timeout: 5_000 });

    // The deleted conversation should no longer appear in the sidebar.
    // Wait a moment for the gRPC delete + list refresh round-trip.
    await expect(
      page.getByRole('button', { name: /WebAssembly and Go/i }),
    ).toBeHidden({ timeout: 10_000 });

    // The other conversation ("Golang Goroutines Explained") is still there.
    await expect(
      page.getByRole('button', { name: /Golang Goroutines Explained/i }).first(),
    ).toBeVisible();
  });
});
