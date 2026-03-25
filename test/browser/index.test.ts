import assert from 'node:assert/strict'
import test from 'node:test'

import { captureConsole, readDiagnostics, waitForAppReady } from './index'

class FakePage {
  constructor() {
    this.calls = []
    this.handlers = new Map()
    this.evaluations = []
  }

  async goto(url) {
    this.calls.push({ method: 'goto', url })
  }

  async waitForSelector(selector, options) {
    this.calls.push({ method: 'waitForSelector', selector, options })
  }

  async waitForFunction(predicate, arg, options) {
    this.calls.push({ method: 'waitForFunction', predicate, arg, options })
  }

  locator(selector) {
    this.calls.push({ method: 'locator', selector })
    return { selector }
  }

  on(event, handler) {
    this.handlers.set(event, handler)
  }

  off(event, handler) {
    const current = this.handlers.get(event)
    if (current === handler) {
      this.handlers.delete(event)
    }
  }

  emit(event, value) {
    const handler = this.handlers.get(event)
    if (handler) {
      handler(value)
    }
  }

  async evaluate(fn, arg) {
    this.evaluations.push({ fn, arg })
    if (typeof fn === 'function') {
      if (typeof arg === 'string' && typeof globalThis[arg] === 'undefined') {
        globalThis[arg] = { ok: true }
      }
      return fn(arg)
    }
    return null
  }
}

test('waitForAppReady navigates and waits for the ready selector', async () => {
  const page = new FakePage()

  const ready = await waitForAppReady(page, {
    url: '/dashboard',
    readySelector: '#dashboard-root',
    waitForFunction: () => true,
    functionArg: { phase: 'hydrated' },
    timeout: 5000,
  })

  assert.equal(page.calls[0].method, 'goto')
  assert.equal(page.calls[0].url, '/dashboard')
  assert.deepEqual(page.calls[1], {
    method: 'waitForSelector',
    selector: '#dashboard-root',
    options: { state: 'visible', timeout: 5000 },
  })
  assert.equal(page.calls[2].method, 'waitForFunction')
  assert.equal(typeof page.calls[2].predicate, 'function')
  assert.deepEqual(page.calls[2].arg, { phase: 'hydrated' })
  assert.deepEqual(page.calls[2].options, { timeout: 5000 })
  assert.deepEqual(page.calls[3], { method: 'locator', selector: '#dashboard-root' })
  assert.equal(ready.selector, '#dashboard-root')
  assert.deepEqual(ready.locator, { selector: '#dashboard-root' })
})

test('captureConsole records console and pageerror entries and detaches cleanly', () => {
  const page = new FakePage()
  const capture = captureConsole(page, {
    includeTypes: ['warning', 'pageerror'],
    maxEntries: 2,
    mapText: (value) => value.toUpperCase(),
  })

  page.emit('console', {
    type: () => 'log',
    text: () => 'ignore me',
    location: () => ({ url: '/ignored.js', lineNumber: 1, columnNumber: 2 }),
  })
  page.emit('console', {
    type: () => 'warning',
    text: () => 'watch this',
    location: () => ({ url: '/app.js', lineNumber: 10, columnNumber: 4 }),
  })
  page.emit('pageerror', new Error('boom'))

  assert.deepEqual(capture.snapshot(), [
    {
      source: 'console',
      type: 'warning',
      text: 'WATCH THIS',
      location: { url: '/app.js', lineNumber: 10, columnNumber: 4 },
    },
    {
      source: 'pageerror',
      type: 'pageerror',
      text: 'BOOM',
      location: undefined,
    },
  ])

  capture.clear()
  assert.deepEqual(capture.snapshot(), [])

  capture.stop()
  assert.equal(page.handlers.size, 0)
})

test('readDiagnostics supports explicit evaluators and app-owned globals', async () => {
  const page = new FakePage()

  const explicit = await readDiagnostics(page, {
    evaluate: () => ({ source: 'explicit', count: 1 }),
  })
  assert.deepEqual(explicit, { source: 'explicit', count: 1 })

  globalThis.__APP_DIAGNOSTICS__ = () => ({ source: 'global', count: 2 })
  const fromGlobal = await readDiagnostics(page, { globalName: '__APP_DIAGNOSTICS__' })
  assert.deepEqual(fromGlobal, { source: 'global', count: 2 })
  delete globalThis.__APP_DIAGNOSTICS__
})
