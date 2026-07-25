package app

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/monstercameron/GoWebComponents/v5/examples/server/ai-chat-wizard/internal/buildinfo"
)

const chatBootShellStyles = `<style>
html {
  margin: 0;
  padding: 0;
  height: 100%;
  overflow: hidden;
  font-size: 87%;
}

body {
  margin: 0;
  padding: 0;
  height: 100%;
  overflow: hidden;
  background: #020617;
  color: #ffffff;
}

#app {
  height: 100dvh;
  display: flex;
}

#boot-shell {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1.5rem;
  background: #020617;
  z-index: 10000;
  transition: opacity 320ms ease, visibility 320ms ease;
}

#boot-shell.is-hidden {
  opacity: 0;
  visibility: hidden;
  pointer-events: none;
}

.boot-card {
  width: 100%;
  max-width: 32rem;
  padding: 2rem;
  border-radius: 2rem;
  border: 1px solid rgba(255,255,255,0.06);
  background: rgba(255,255,255,0.02);
  backdrop-filter: blur(12px);
  box-shadow: 0 20px 80px rgba(0,0,0,0.55);
}

.boot-header-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 2rem;
}

.boot-labels .boot-brand {
  margin: 0;
  font-size: 0.625rem;
  text-transform: uppercase;
  letter-spacing: 0.35em;
  color: rgba(255,255,255,0.35);
}

.boot-labels .boot-version {
  margin: 0.375rem 0 0;
  font-family: "Geist Mono", ui-monospace, SFMono-Regular, Menlo, Monaco, "Courier New", monospace;
  font-size: 0.625rem;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: rgba(255,255,255,0.36);
}

.boot-labels .boot-heading {
  margin: 0.75rem 0 0;
  font-size: 1.25rem;
  font-weight: 500;
  letter-spacing: -0.02em;
  color: rgba(255,255,255,0.9);
}

.boot-percent {
  font-variant-numeric: tabular-nums;
  font-size: 0.875rem;
  color: rgba(255,255,255,0.4);
}

.boot-progress-track {
  position: relative;
  height: 2px;
  border-radius: 999px;
  overflow: hidden;
  background: rgba(255,255,255,0.08);
}

.boot-progress-fill {
  position: absolute;
  inset-block: 0;
  left: 0;
  width: 0%;
  border-radius: inherit;
  background: linear-gradient(90deg, rgba(34,211,238,0.7), #67e8f9, #7dd3fc);
  transition: width 300ms ease;
}

.boot-progress-fill::after {
  content: "";
  position: absolute;
  inset: 0;
  background: linear-gradient(90deg, transparent, rgba(255,255,255,0.24), transparent);
  transform: translateX(-100%);
  animation: boot-shimmer 1.6s linear infinite;
}

.boot-progress-fill.is-indeterminate {
  width: 32%;
  animation: boot-indeterminate 1.4s ease-in-out infinite;
}

.boot-progress-fill.is-indeterminate::after {
  animation-duration: 1s;
}

.boot-footer-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 1rem;
  font-size: 0.875rem;
  color: rgba(255,255,255,0.35);
}

.boot-footer-row .boot-wait {
  font-size: 0.6875rem;
  text-transform: uppercase;
  letter-spacing: 0.18em;
}

.boot-footer-row .boot-detail {
  font-size: 0.75rem;
  color: rgba(255,255,255,0.5);
  text-transform: none;
  letter-spacing: 0.02em;
}

/* finalizing state */
.boot-finalizing {
  display: none;
  min-height: 5.25rem;
  align-items: center;
  justify-content: space-between;
}

.boot-finalizing-text .boot-fin-label {
  margin: 0;
  font-size: 0.875rem;
  color: rgba(255,255,255,0.38);
}

.boot-finalizing-text .boot-fin-heading {
  margin: 0.5rem 0 0;
  font-size: 1rem;
  font-weight: 500;
  color: rgba(255,255,255,0.88);
}

.boot-spinner-wrap {
  position: relative;
  width: 2.5rem;
  height: 2.5rem;
  flex-shrink: 0;
}

.boot-spinner-ring {
  position: absolute;
  inset: 0;
  border-radius: 50%;
  border: 1px solid rgba(255,255,255,0.08);
}

.boot-spinner {
  position: absolute;
  inset: 0;
  border-radius: 50%;
  border: 2px solid transparent;
  border-top-color: rgba(103,232,249,0.8);
  border-right-color: rgba(125,211,252,0.6);
  animation: boot-spin 900ms linear infinite;
}

/* body toggle */
#boot-shell.is-finalizing .boot-progress-body { display: none; }
#boot-shell.is-finalizing .boot-finalizing { display: flex; }

.boot-shell-error .boot-progress-fill {
  background: linear-gradient(90deg, #ef4444, #fb7185);
}

@keyframes boot-spin {
  to { transform: rotate(360deg); }
}

@keyframes boot-shimmer {
  to { transform: translateX(100%); }
}

@keyframes boot-indeterminate {
  0% { transform: translateX(-10%); }
  100% { transform: translateX(240%); }
}
</style>`

const chatShellHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>RelayDesk - AI Chat Workspace</title>
  <link rel="preconnect" href="https://fonts.googleapis.com" />
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
  <link href="https://fonts.googleapis.com/css2?family=Space+Grotesk:wght@500;600;700&family=Geist:wght@400;500;600&family=Geist+Mono:wght@400;500&display=swap" rel="stylesheet" />
  <link rel="stylesheet" href="/static/css/tailwind.css" />
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/katex@0.16.11/dist/katex.min.css" />
  {{BOOT_STYLE}}
</head>
<body>
  <div id="boot-shell" aria-live="polite">
    <div class="boot-card">
      <div class="boot-header-row">
        <div class="boot-labels">
          <p class="boot-brand">RelayDesk</p>
          <p class="boot-version">{{APP_VERSION}}</p>
          <p id="boot-heading" class="boot-heading">Preparing interface</p>
        </div>
        <span id="boot-percent" class="boot-percent">0%</span>
      </div>
      <div class="boot-progress-body">
        <div class="boot-progress-track" aria-hidden="true">
          <div id="boot-progress-fill" class="boot-progress-fill"></div>
        </div>
        <div class="boot-footer-row">
          <span id="boot-stage">Loading</span>
          <span id="boot-detail" class="boot-detail">Please wait</span>
        </div>
      </div>
      <div class="boot-finalizing" aria-hidden="true">
        <div class="boot-finalizing-text">
          <p id="boot-fin-label" class="boot-fin-label">Finalizing</p>
          <p id="boot-fin-heading" class="boot-fin-heading">Almost ready</p>
        </div>
        <div class="boot-spinner-wrap">
          <div class="boot-spinner-ring"></div>
          <div class="boot-spinner"></div>
        </div>
      </div>
    </div>
  </div>

  <div id="app"></div>

  <script src="/static/script/wasm_exec.js"></script>
  <script src="https://cdn.jsdelivr.net/npm/katex@0.16.11/dist/katex.min.js"></script>
  <script src="https://cdn.jsdelivr.net/npm/katex@0.16.11/dist/contrib/auto-render.min.js"></script>
  <script src="https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.min.js"></script>
  <script src="/chat-bootstrap.js"></script>
</body>
</html>
`

var chatUsagePremiumPercent = 5.0
var chatPlatformFeeUSD = 29.0
var parseAgentBridgeBootstrapScript string

func setChatUsagePremiumPercent(parsePercent float64) {
	chatUsagePremiumPercent = parsePercent
}

func parseCurrentChatUsagePremiumPercent() float64 {
	return chatUsagePremiumPercent
}

func setChatPlatformFeeUSD(parseFee float64) {
	chatPlatformFeeUSD = parseFee
}

func parseCurrentChatPlatformFeeUSD() float64 {
	return chatPlatformFeeUSD
}

// parseSetAgentBridgeBootstrapScript stores the optional inline script that
// seeds the dogfood agent bridge before the client bootstrap runs.
func parseSetAgentBridgeBootstrapScript(parseScript string) {
	parseAgentBridgeBootstrapScript = strings.TrimSpace(parseScript)
}

const chatBootstrapJS = `window.__relaydesk_usage_premium_percent = {{USAGE_PREMIUM_PERCENT}};
window.__relaydesk_platform_fee_usd = {{PLATFORM_FEE_USD}};
const bootShell = document.getElementById('boot-shell');
const bootProgressFill = document.getElementById('boot-progress-fill');
const bootPercent = document.getElementById('boot-percent');
const appRoot = document.getElementById('app');
const mathDelimiters = [
  { left: '$$', right: '$$', display: true },
  { left: '\\[', right: '\\]', display: true },
  { left: '$', right: '$', display: false },
  { left: '\\(', right: '\\)', display: false },
];
const mermaidRenderCache = new Map();
let mermaidInitialized = false;
let mermaidSequence = 0;

let bootProgressValue = 0;

function looksLikeMathText(text) {
  return /\\\(|\\\)|\\\[|\\\]|\$/.test(text || '') || /^\s*\[[^\n]*\\[A-Za-z][\s\S]*\]\s*$/m.test(text || '');
}

function looksLikeStandaloneBracketMath(text) {
  const trimmed = String(text || '').trim();
  if (!/^\[[\s\S]+\]$/.test(trimmed)) {
    return false;
  }
  const inner = trimmed.slice(1, -1).trim();
  if (!inner || /[<>]/.test(inner)) {
    return false;
  }
  return /\\[A-Za-z]+|[=+\-*/^_]|(?:\d[\d.,]*)/.test(inner);
}

function normalizeStandaloneBracketMath(root) {
  if (!root) return;

  const targets = [];
  if (root.matches && root.matches('.prose p, .prose li, .prose blockquote')) {
    targets.push(root);
  }
  if (typeof root.querySelectorAll === 'function') {
    root.querySelectorAll('.prose p:not([data-math-normalized]), .prose li:not([data-math-normalized]), .prose blockquote:not([data-math-normalized])').forEach(node => targets.push(node));
  }

  targets.forEach(node => {
    if (!node || node.dataset.mathNormalized === '1' || node.children.length > 0) {
      return;
    }
    node.dataset.mathNormalized = '1';

    const text = node.textContent || '';
    if (!looksLikeStandaloneBracketMath(text)) {
      return;
    }

    const inner = text.trim().slice(1, -1).trim();
    node.textContent = '\\[' + inner + '\\]';
    node.classList.add('prose-math-block');
  });
}

function renderMathInRoot(root) {
  if (!root || typeof renderMathInElement !== 'function') return;
  normalizeStandaloneBracketMath(root);

  const targets = [];
  if (root.classList && root.classList.contains('prose')) {
    targets.push(root);
  }
  if (typeof root.querySelectorAll === 'function') {
    root.querySelectorAll('.prose').forEach(node => targets.push(node));
  }

  targets.forEach(node => {
    if (!node || node.querySelector('.katex, .katex-display')) {
      return;
    }
    if (!looksLikeMathText(node.textContent)) {
      return;
    }
    try {
      renderMathInElement(node, {
        delimiters: mathDelimiters,
        throwOnError: false,
        strict: 'ignore',
        output: 'htmlAndMathml',
        ignoredTags: ['script', 'noscript', 'style', 'textarea', 'pre', 'code', 'option'],
      });
    } catch (_) {
    }
  });
}

function ensureMermaidReady() {
  if (typeof mermaid === 'undefined') {
    return false;
  }
  if (!mermaidInitialized) {
    mermaid.initialize({
      startOnLoad: false,
      securityLevel: 'strict',
      theme: 'dark',
      fontFamily: 'inherit',
    });
    mermaidInitialized = true;
  }
  return true;
}

function isMermaidCodeBlock(pre) {
  const code = pre && pre.querySelector ? pre.querySelector('code') : null;
  if (!code) return false;
  const className = String(code.className || '');
  return /\blanguage-mermaid\b/i.test(className);
}

async function renderMermaidInRoot(root) {
  if (!root || !ensureMermaidReady()) return;

  const targets = [];
  if (root.matches && root.matches('.prose pre') && isMermaidCodeBlock(root)) {
    targets.push(root);
  }
  if (typeof root.querySelectorAll === 'function') {
    root.querySelectorAll('.prose pre:not([data-mermaid-processed])').forEach(pre => {
      if (isMermaidCodeBlock(pre)) {
        targets.push(pre);
      }
    });
  }

  for (const pre of targets) {
    if (!pre || pre.dataset.mermaidProcessed === '1') {
      continue;
    }
    pre.dataset.mermaidProcessed = '1';

    const code = pre.querySelector('code');
    const source = ((code || pre).textContent || '').trim();
    if (!source) {
      continue;
    }

    const wrap = document.createElement('div');
    wrap.className = 'prose-mermaid-wrap';

    const stage = document.createElement('div');
    stage.className = 'prose-mermaid-stage';
    wrap.appendChild(stage);

    const btn = document.createElement('button');
    btn.className = 'prose-copy-btn';
    btn.textContent = 'Copy';
    btn.addEventListener('click', () => {
      navigator.clipboard.writeText(source).then(() => {
        btn.textContent = 'Copied!';
        btn.classList.add('copied');
        setTimeout(() => { btn.textContent = 'Copy'; btn.classList.remove('copied'); }, 2000);
      }).catch(() => {});
    });
    wrap.appendChild(btn);

    pre.parentNode.insertBefore(wrap, pre);
    pre.remove();

    try {
      let svg = mermaidRenderCache.get(source);
      if (!svg) {
        const renderResult = await mermaid.render('gwc-mermaid-' + (++mermaidSequence), source);
        svg = renderResult && renderResult.svg ? renderResult.svg : '';
        if (svg) {
          mermaidRenderCache.set(source, svg);
        }
      }
      if (!svg) {
        throw new Error('empty Mermaid render result');
      }
      stage.innerHTML = svg;
    } catch (err) {
      const fallback = document.createElement('pre');
      fallback.textContent = source;
      stage.replaceChildren(fallback);

      const errorNote = document.createElement('div');
      errorNote.className = 'prose-mermaid-error';
      errorNote.textContent = 'Mermaid render failed: ' + String(err && err.message ? err.message : err);
      wrap.appendChild(errorNote);
    }
  }
}

function formatBytes(bytes) {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB'];
  let unitIndex = 0;
  let value = bytes;
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024;
    unitIndex += 1;
  }
  const rounded = value >= 100 ? value.toFixed(0) : value >= 10 ? value.toFixed(1) : value.toFixed(2);
  return rounded + ' ' + units[unitIndex];
}

function setBootText(id, value) {
  const getTarget = document.getElementById(id);
  if (!getTarget) {
    return;
  }
  const setValue = String(value || '').trim();
  if (setValue === '') {
    return;
  }
  getTarget.textContent = setValue;
}

function setBootProgress(progress, options) {
  const config = options || {};
  const clamped = Math.max(0, Math.min(100, progress));
  bootProgressValue = Math.max(bootProgressValue, clamped);
  const isFinalizing = !!config.finalizing;
  const isIndeterminate = !!config.indeterminate && !isFinalizing;
  bootShell.classList.toggle('is-finalizing', isFinalizing);
  bootProgressFill.classList.toggle('is-indeterminate', isIndeterminate);
  if (isIndeterminate) {
    bootProgressFill.style.width = '';
  } else {
    bootProgressFill.style.width = bootProgressValue + '%';
  }
  bootPercent.textContent = isFinalizing ? '100%' : Math.round(bootProgressValue) + '%';
}

function setBootPhase(statusText, detailText, stageLabel, progress, options) {
  setBootProgress(progress, options);
  setBootText('boot-heading', statusText);
  setBootText('boot-stage', stageLabel);
  setBootText('boot-detail', detailText);
  if (options && options.finalizing) {
    setBootText('boot-fin-label', stageLabel);
    setBootText('boot-fin-heading', statusText);
  }
}

function appHasMounted() {
  if (!appRoot) {
    return false;
  }
  if (appRoot.children && appRoot.children.length > 0) {
    return true;
  }
  return String(appRoot.textContent || '').trim() !== '';
}

function waitForAppMount(timeoutMs) {
  if (appHasMounted()) {
    return Promise.resolve();
  }
  return new Promise(resolve => {
    let settled = false;
    const finish = function() {
      if (settled) {
        return;
      }
      settled = true;
      if (observer) {
        observer.disconnect();
      }
      if (timer) {
        clearTimeout(timer);
      }
      resolve();
    };
    const observer = appRoot && typeof MutationObserver !== 'undefined'
      ? new MutationObserver(function() {
          if (appHasMounted()) {
            finish();
          }
        })
      : null;
    if (observer && appRoot) {
      observer.observe(appRoot, { childList: true, subtree: true, characterData: true });
    }
    const timer = setTimeout(finish, timeoutMs);
  });
}

function isMarketingBootRoute() {
  const pathname = String((window.location && window.location.pathname) || '').trim();
  return pathname === '/' ||
    pathname === '/home' ||
    pathname === '/capabilities' ||
    pathname === '/pricing' ||
    pathname === '/signup';
}

function getBootCopy() {
  if (isMarketingBootRoute()) {
    return {
      loadingHeading: 'Loading RelayDesk',
      finalizingHeading: 'Almost ready',
      finalizingDetail: 'Opening the page...',
      readyHeading: 'Welcome to RelayDesk',
      readyDetail: 'Your page is ready.',
    };
  }
  return {
    loadingHeading: 'Loading your workspace',
    finalizingHeading: 'Almost ready',
    finalizingDetail: 'Your workspace is almost ready...',
    readyHeading: 'Welcome to RelayDesk',
    readyDetail: 'Your workspace is ready.',
  };
}

function finishBoot() {
  const copy = getBootCopy();
  setBootPhase(copy.readyHeading, copy.readyDetail, 'Ready', 100);
  requestAnimationFrame(() => {
    bootShell.classList.add('is-hidden');
    // Remove the boot overlay entirely once the app has mounted so stale boot
    // copy can never reflow or flash during the first hydrated frame.
    setTimeout(() => {
      if (bootShell && bootShell.parentNode) {
        bootShell.parentNode.removeChild(bootShell);
      }
    }, 40);
  });
}

function failBoot(err) {
  console.error('WASM failed to load:', err);
  bootShell.classList.add('boot-shell-error');
  setBootPhase('Something went wrong', 'Please refresh the page to try again.', 'Error', 100);
  document.getElementById('app').textContent = 'RelayDesk failed to load. Please refresh the page.';
}

async function loadChatWasm() {
  const go = new Go();
  const params = new URLSearchParams(window.location.search);
  const brParam = (params.get('br') || '').toLowerCase();
  const useBrotli = brParam !== 'false' && brParam !== '0';
  const wasmQuery = useBrotli ? '?br=true' : '';
  window.__gwc_wasm_query = wasmQuery;
  const response = await fetch('/app/chat.wasm' + wasmQuery);
  if (!response.ok) {
    throw new Error('HTTP ' + response.status + ' while fetching /app/chat.wasm');
  }

  const contentEncoding = String(response.headers.get('content-encoding') || '').toLowerCase();
  const totalBytes = Number(response.headers.get('content-length') || 0);
  const copy = getBootCopy();
  if (contentEncoding && contentEncoding !== 'identity') {
    setBootPhase(
      copy.loadingHeading,
      'Downloading WebAssembly...',
      'Loading',
      88,
      { indeterminate: true }
    );
    const result = await WebAssembly.instantiateStreaming(Promise.resolve(response), go.importObject);
    setBootPhase(copy.finalizingHeading, copy.finalizingDetail, 'Finalizing', 98, { indeterminate: true, finalizing: true });
    const runPromise = go.run(result.instance);
    await waitForAppMount(8000);
    finishBoot();
    await runPromise;
    return;
  }

  if (!response.body || typeof response.body.getReader !== 'function') {
    setBootPhase(copy.loadingHeading, 'Loading...', 'Loading', 34, { indeterminate: true });
    const result = await WebAssembly.instantiateStreaming(Promise.resolve(response), go.importObject);
    setBootPhase('Almost there', 'Getting the final pieces ready...', 'Loading', 92, { indeterminate: true });
    setBootPhase(copy.finalizingHeading, copy.finalizingDetail, 'Finalizing', 98, { indeterminate: true, finalizing: true });
    const runPromise = go.run(result.instance);
    await waitForAppMount(8000);
    finishBoot();
    await runPromise;
    return;
  }

  const reader = response.body.getReader();
  const chunks = [];
  let receivedBytes = 0;
  let unknownSizeProgress = 6;

  setBootPhase(copy.loadingHeading, totalBytes > 0 ? '0 of ' + formatBytes(totalBytes) : 'Calculating download size...', 'Loading', 4);

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    chunks.push(value);
    receivedBytes += value.byteLength;

    if (totalBytes > 0) {
      const progress = 8 + (receivedBytes / totalBytes) * 68;
      setBootPhase(copy.loadingHeading, formatBytes(receivedBytes) + ' of ' + formatBytes(totalBytes), 'Loading', progress);
    } else {
      unknownSizeProgress = Math.min(72, unknownSizeProgress + 3.5);
      setBootPhase(copy.loadingHeading, formatBytes(receivedBytes) + ' received', 'Loading', unknownSizeProgress, { indeterminate: true });
    }
  }

  setBootPhase('Almost there', 'Downloaded ' + formatBytes(receivedBytes) + '. Finishing up...', 'Loading', 82, { indeterminate: true });

  const wasmResponse = new Response(new Blob(chunks), {
    headers: { 'Content-Type': 'application/wasm' },
  });
  const result = await WebAssembly.instantiateStreaming(Promise.resolve(wasmResponse), go.importObject);

  setBootPhase(copy.finalizingHeading, copy.finalizingDetail, 'Finalizing', 98, { indeterminate: true, finalizing: true });
  const runPromise = go.run(result.instance);
  await waitForAppMount(8000);
  finishBoot();
  await runPromise;
}

loadChatWasm().catch(failBoot);

const codeObserver = new MutationObserver(() => {
  renderMermaidInRoot(appRoot).catch(() => {});

  document.querySelectorAll('.prose pre:not([data-copy])').forEach(pre => {
    if (pre.dataset.mermaidProcessed === '1' || isMermaidCodeBlock(pre)) {
      return;
    }
    pre.dataset.copy = '1';

    const wrap = document.createElement('div');
    wrap.className = 'prose-pre-wrap';
    pre.parentNode.insertBefore(wrap, pre);
    wrap.appendChild(pre);

    const btn = document.createElement('button');
    btn.className = 'prose-copy-btn';
    btn.textContent = 'Copy';
    btn.addEventListener('click', () => {
      const code = pre.querySelector('code');
      navigator.clipboard.writeText((code || pre).textContent).then(() => {
        btn.textContent = 'Copied!';
        btn.classList.add('copied');
        setTimeout(() => { btn.textContent = 'Copy'; btn.classList.remove('copied'); }, 2000);
      }).catch(() => {});
    });
    wrap.appendChild(btn);
  });

  renderMathInRoot(appRoot);
});

if (appRoot) {
  codeObserver.observe(appRoot, { childList: true, subtree: true });
  renderMermaidInRoot(appRoot).catch(() => {});
  renderMathInRoot(appRoot);
}

document.addEventListener('click', function(e) {
  const wrap = e.target.closest('#chat-input-wrap');
  if (wrap && !e.target.closest('#send-btn')) {
    const inp = document.getElementById('chat-input');
    if (inp) inp.focus();
  }
}, true);

document.addEventListener('click', function(e) {
  if (e.target.closest('#send-btn')) {
    setTimeout(function() {
      const inp = document.getElementById('chat-input');
      if (inp) inp.focus();
    }, 150);
  }
}, false);

(function () {
  let cachedRect = null;

  function trackEmptyRect() {
    const el = document.getElementById('empty-state');
    if (!el) return;
    cachedRect = el.getBoundingClientRect();
    requestAnimationFrame(trackEmptyRect);
  }

  function playExitAnimation(el, rect) {
    const ghost = el.cloneNode(true);
    ghost.removeAttribute('id');
    Object.assign(ghost.style, {
      position: 'fixed',
      left: rect.left + 'px',
      top: rect.top + 'px',
      width: rect.width + 'px',
      margin: '0',
      pointerEvents: 'none',
      zIndex: '9999',
    });
    document.body.appendChild(ghost);

    const targetX = 28;
    const targetY = 32;
    const cx = rect.left + rect.width / 2;
    const cy = rect.top + rect.height / 2;
    const dx = targetX - cx;
    const dy = targetY - cy;

    ghost.animate(
      [
        { opacity: 1, transform: 'translate(0, 0) scale(1)', easing: 'cubic-bezier(0.4, 0, 1, 1)' },
        { opacity: 0, transform: 'translate(' + dx + 'px, ' + dy + 'px) scale(0.1)' },
      ],
      { duration: 360, fill: 'forwards' }
    ).onfinish = () => ghost.remove();
  }

  const listObserver = new MutationObserver(mutations => {
    for (const mut of mutations) {
      for (const node of mut.addedNodes) {
        if (node.id === 'empty-state') trackEmptyRect();
      }
      for (const node of mut.removedNodes) {
        if (node.id === 'empty-state' && cachedRect) {
          playExitAnimation(node, cachedRect);
          cachedRect = null;
        }
      }
    }
  });

  function attachListObserver() {
    const ml = document.getElementById('message-list');
    if (ml) {
      listObserver.observe(ml, { childList: true });
      trackEmptyRect();
    } else {
      requestAnimationFrame(attachListObserver);
    }
  }

  attachListObserver();
})();

(function () {
  function animateThreadScreenIn(container) {
    if (!container || !container.classList || !container.classList.contains('thread-screen')) return;
    container.classList.remove('thread-screen');
    requestAnimationFrame(() => container.classList.add('thread-screen'));
  }

  function playGhostExit(source, rect, className, keyframes, options) {
    if (!source || !rect || rect.width <= 0 || rect.height <= 0) return;
    const ghost = source.cloneNode(true);
    ghost.removeAttribute('id');
    ghost.classList.add(className);
    Object.assign(ghost.style, {
      position: 'fixed',
      left: rect.left + 'px',
      top: rect.top + 'px',
      width: rect.width + 'px',
      height: rect.height + 'px',
      margin: '0',
      pointerEvents: 'none',
    });
    document.body.appendChild(ghost);
    ghost.animate(keyframes, options).onfinish = () => ghost.remove();
  }

  function startConversationAnimations(list) {
    if (!list || list.__gwcConvAnimationsAttached) return;
    list.__gwcConvAnimationsAttached = true;

    const rowRects = new Map();
    const allRows = root => Array.from(root.querySelectorAll ? root.querySelectorAll('[data-convrow="1"]') : []);

    function trackRows() {
      allRows(list).forEach(row => {
        const convId = row.dataset.convid;
        if (convId) rowRects.set(convId, row.getBoundingClientRect());
      });
      requestAnimationFrame(trackRows);
    }

    const rowObserver = new MutationObserver(mutations => {
      for (const mutation of mutations) {
        mutation.addedNodes.forEach(node => {
          if (!(node instanceof HTMLElement)) return;
          const rows = node.dataset.convrow === '1' ? [node] : allRows(node);
          rows.forEach(row => {
            row.classList.remove('conv-row');
            requestAnimationFrame(() => row.classList.add('conv-row'));
          });
        });
        mutation.removedNodes.forEach(node => {
          if (!(node instanceof HTMLElement)) return;
          const rows = node.dataset.convrow === '1' ? [node] : allRows(node);
          rows.forEach(row => {
            const convId = row.dataset.convid;
            const rect = convId ? rowRects.get(convId) : null;
            playGhostExit(
              row,
              rect,
              'conv-row-removing',
              [
                { opacity: 1, transform: 'translateX(0) scale(1)', filter: 'blur(0px)' },
                { opacity: 0, transform: 'translateX(14px) scale(0.985)', filter: 'blur(5px)' }
              ],
              { duration: 220, easing: 'cubic-bezier(0.4, 0, 1, 1)', fill: 'forwards' }
            );
            if (convId) rowRects.delete(convId);
          });
        });
      }
    });

    rowObserver.observe(list, { childList: true, subtree: true });
    trackRows();
  }

  const appObserver = new MutationObserver(mutations => {
    for (const mutation of mutations) {
      mutation.addedNodes.forEach(node => {
        if (!(node instanceof HTMLElement)) return;
        if (node.id === 'thread-screen' || node.id === 'empty-state' || node.classList.contains('thread-screen')) {
          animateThreadScreenIn(node);
        }
        if (node.id === 'conversation-list') {
          startConversationAnimations(node);
        }
      });

      mutation.removedNodes.forEach(node => {
        if (!(node instanceof HTMLElement)) return;
        if (node.id !== 'thread-screen' && node.id !== 'empty-state') return;
        const rect = node.getBoundingClientRect();
        playGhostExit(
          node,
          rect,
          'thread-screen-ghost',
          [
            { opacity: 1, transform: 'translateY(0) scale(1)', filter: 'blur(0px)' },
            { opacity: 0, transform: 'translateY(-8px) scale(0.992)', filter: 'blur(10px)' }
          ],
          { duration: 210, easing: 'cubic-bezier(0.4, 0, 1, 1)', fill: 'forwards' }
        );
      });
    }
  });

  function attachAnimatedRegions() {
    const appRoot = document.getElementById('app');
    if (!appRoot) {
      requestAnimationFrame(attachAnimatedRegions);
      return;
    }
    appObserver.observe(appRoot, { childList: true, subtree: true });
    const convList = document.getElementById('conversation-list');
    if (convList) startConversationAnimations(convList);
    const threadScreen = document.getElementById('thread-screen');
    if (threadScreen) animateThreadScreenIn(threadScreen);
  }

  attachAnimatedRegions();
})();`

func parseServeChatShell(parseW http.ResponseWriter, _ *http.Request) {
	parseSetNoStoreResponseHeaders(parseW)
	parseW.Header().Set("Content-Type", "text/html; charset=utf-8")
	parseShellHTML := strings.Replace(chatShellHTML, "{{BOOT_STYLE}}", chatBootShellStyles, 1)
	parseShellHTML = strings.Replace(parseShellHTML, "{{APP_VERSION}}", buildinfo.GetBuildAppVersion(), 1)
	if parseAgentBridgeBootstrapScript != "" {
		parseShellHTML = strings.Replace(parseShellHTML, `  <script src="/chat-bootstrap.js"></script>`, "  "+parseAgentBridgeBootstrapScript+"\n  <script src=\"/chat-bootstrap.js\"></script>", 1)
	}
	_, _ = fmt.Fprint(parseW, parseShellHTML)
}

func parseServeChatBootstrapJS(parseW http.ResponseWriter, _ *http.Request) {
	parseSetNoStoreResponseHeaders(parseW)
	parseW.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	parsePremiumLiteral := strconv.FormatFloat(parseCurrentChatUsagePremiumPercent(), 'f', 6, 64)
	parsePlatformFeeLiteral := strconv.FormatFloat(parseCurrentChatPlatformFeeUSD(), 'f', 6, 64)
	parseBootstrapBody := strings.Replace(chatBootstrapJS, "{{USAGE_PREMIUM_PERCENT}}", parsePremiumLiteral, 1)
	parseBootstrapBody = strings.Replace(parseBootstrapBody, "{{PLATFORM_FEE_USD}}", parsePlatformFeeLiteral, 1)
	_, _ = fmt.Fprint(parseW, parseBootstrapBody)
}

// parseSetNoStoreResponseHeaders applies one fail-safe no-store cache policy for mutable shell and artifact responses.
func parseSetNoStoreResponseHeaders(parseW http.ResponseWriter) {
	if parseW == nil {
		return
	}
	parseW.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	parseW.Header().Set("Pragma", "no-cache")
	parseW.Header().Set("Expires", "0")
}
