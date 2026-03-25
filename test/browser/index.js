const defaultReadySelector = '#app';
const defaultTimeoutMs = 30_000;
const defaultDiagnosticGlobal = '__GWC_DIAGNOSTICS__';

export async function waitForAppReady(page, options = {}) {
  if (!page) {
    throw new Error('waitForAppReady requires a Playwright-style page')
  }

  const {
    url,
    readySelector = defaultReadySelector,
    readyState = 'visible',
    timeout = defaultTimeoutMs,
    waitForFunction,
    functionArg,
  } = options

  if (url) {
    await page.goto(url)
  }
  if (readySelector) {
    await page.waitForSelector(readySelector, { state: readyState, timeout })
  }
  if (waitForFunction) {
    await page.waitForFunction(waitForFunction, functionArg, { timeout })
  }

  return {
    selector: readySelector,
    timeout,
    locator: readySelector ? page.locator(readySelector) : null,
  }
}

export function captureConsole(page, options = {}) {
  if (!page) {
    throw new Error('captureConsole requires a Playwright-style page')
  }

  const {
    includeConsole = true,
    includePageErrors = true,
    includeTypes = [],
    maxEntries = 200,
    mapText = (value) => value,
  } = options

  const typeFilter = new Set(includeTypes.map((value) => String(value).toLowerCase()))
  const entries = []

  const push = (entry) => {
    if (typeFilter.size > 0 && !typeFilter.has(String(entry.type).toLowerCase())) {
      return
    }
    entries.push(entry)
    if (entries.length > maxEntries) {
      entries.splice(0, entries.length - maxEntries)
    }
  }

  const consoleHandler = includeConsole
    ? (message) => {
        const location = typeof message.location === 'function' ? message.location() : undefined
        push({
          source: 'console',
          type: typeof message.type === 'function' ? message.type() : 'log',
          text: mapText(typeof message.text === 'function' ? message.text() : ''),
          location: location && typeof location === 'object'
            ? {
                url: location.url ?? '',
                lineNumber: location.lineNumber ?? 0,
                columnNumber: location.columnNumber ?? 0,
              }
            : undefined,
        })
      }
    : null

  const pageErrorHandler = includePageErrors
    ? (error) => {
        push({
          source: 'pageerror',
          type: 'pageerror',
          text: mapText(error instanceof Error ? error.message : String(error ?? '')),
        })
      }
    : null

  if (consoleHandler) {
    page.on('console', consoleHandler)
  }
  if (pageErrorHandler) {
    page.on('pageerror', pageErrorHandler)
  }

  return {
    entries,
    snapshot() {
      return entries.map((entry) => ({
        ...entry,
        location: entry.location ? { ...entry.location } : undefined,
      }))
    },
    clear() {
      entries.length = 0
    },
    stop() {
      if (consoleHandler) {
        page.off('console', consoleHandler)
      }
      if (pageErrorHandler) {
        page.off('pageerror', pageErrorHandler)
      }
    },
  }
}

export async function readDiagnostics(page, options = {}) {
  if (!page) {
    throw new Error('readDiagnostics requires a Playwright-style page')
  }

  const { evaluate, globalName = defaultDiagnosticGlobal } = options
  if (evaluate) {
    return page.evaluate(evaluate)
  }

  return page.evaluate((name) => {
    const candidate = globalThis[name]
    if (typeof candidate === 'function') {
      return candidate()
    }
    return candidate ?? null
  }, globalName)
}
