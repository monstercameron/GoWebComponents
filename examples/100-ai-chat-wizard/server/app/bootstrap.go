package app

import (
	"fmt"
	"net/http"
	"strings"
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
  background: #212121;
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
  background:
    radial-gradient(circle at top, rgba(25,195,125,0.14), transparent 36%),
    radial-gradient(circle at bottom right, rgba(99,102,241,0.16), transparent 30%),
    linear-gradient(180deg, #171717 0%, #212121 50%, #181818 100%);
  z-index: 10000;
  transition: opacity 320ms ease, visibility 320ms ease;
}

#boot-shell.is-hidden {
  opacity: 0;
  visibility: hidden;
  pointer-events: none;
}

.boot-card {
  width: min(92vw, 28rem);
  padding: 1.35rem;
  border-radius: 1.5rem;
  border: 1px solid rgba(255,255,255,0.08);
  background: linear-gradient(180deg, rgba(255,255,255,0.055), rgba(255,255,255,0.02));
  backdrop-filter: blur(16px);
  box-shadow: 0 22px 80px rgba(0,0,0,0.34);
}

.boot-top {
  display: flex;
  align-items: center;
  gap: 0.9rem;
  margin-bottom: 1rem;
}

.boot-badge {
  width: 2.85rem;
  height: 2.85rem;
  border-radius: 999px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #19c37d, #0ea47e);
  color: white;
  font-weight: 700;
  letter-spacing: 0.04em;
  box-shadow: 0 10px 32px rgba(25,195,125,0.28);
}

.boot-heading {
  margin: 0;
  font-size: 1rem;
  font-weight: 600;
  letter-spacing: -0.01em;
  color: rgba(255,255,255,0.96);
}

.boot-subheading {
  margin: 0.18rem 0 0;
  font-size: 0.82rem;
  color: rgba(255,255,255,0.44);
  text-transform: uppercase;
  letter-spacing: 0.16em;
}

.boot-status-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 0.6rem;
}

.boot-status-text {
  margin: 0;
  font-size: 0.95rem;
  color: rgba(255,255,255,0.82);
}

.boot-metrics {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  flex-shrink: 0;
}

.boot-percent {
  font-variant-numeric: tabular-nums;
  font-size: 0.9rem;
  color: rgba(255,255,255,0.64);
  min-width: 3.2rem;
  text-align: right;
}

.boot-spinner {
  width: 1rem;
  height: 1rem;
  border-radius: 999px;
  border: 2px solid rgba(255,255,255,0.16);
  border-top-color: rgba(255,255,255,0.92);
  animation: boot-spin 900ms linear infinite;
  flex-shrink: 0;
}

.boot-progress-track {
  position: relative;
  height: 0.52rem;
  border-radius: 999px;
  overflow: hidden;
  background: rgba(255,255,255,0.08);
  box-shadow: inset 0 1px 2px rgba(0,0,0,0.25);
}

.boot-progress-fill {
  position: relative;
  width: 0%;
  height: 100%;
  border-radius: inherit;
  background: linear-gradient(90deg, #19c37d 0%, #53d7ac 45%, #a2f4d8 100%);
  box-shadow: 0 0 24px rgba(25,195,125,0.3);
  transition: width 220ms ease;
}

.boot-progress-fill::after {
  content: "";
  position: absolute;
  inset: 0;
  background: linear-gradient(90deg, transparent, rgba(255,255,255,0.28), transparent);
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

.boot-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  margin-top: 0.7rem;
  color: rgba(255,255,255,0.42);
  font-size: 0.83rem;
}

.boot-detail {
  margin: 0;
  min-height: 1.2rem;
}

.boot-stage {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.3rem 0.55rem;
  border-radius: 999px;
  background: rgba(255,255,255,0.06);
  border: 1px solid rgba(255,255,255,0.08);
  white-space: nowrap;
}

.boot-stage-dot {
  width: 0.38rem;
  height: 0.38rem;
  border-radius: 999px;
  background: #53d7ac;
  box-shadow: 0 0 10px rgba(83,215,172,0.8);
}

.boot-shell-error .boot-progress-fill {
  background: linear-gradient(90deg, #ef4444 0%, #fb7185 100%);
  box-shadow: 0 0 18px rgba(239,68,68,0.28);
}

.boot-shell-error .boot-stage-dot {
  background: #fb7185;
  box-shadow: 0 0 10px rgba(251,113,133,0.7);
}

@keyframes boot-spin {
  to { transform: rotate(360deg); }
}

@keyframes boot-shimmer {
  to { transform: translateX(100%); }
}

@keyframes boot-indeterminate {
  0% { transform: translateX(-18%); }
  50% { transform: translateX(150%); }
  100% { transform: translateX(-18%); }
}
</style>`

const chatShellHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>GoWebComponents Lab - Go WASM UI Experiment</title>
  <link rel="stylesheet" href="/static/css/tailwind.css" />
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/katex@0.16.11/dist/katex.min.css" />
  {{BOOT_STYLE}}
</head>
<body>
  <div id="boot-shell" aria-live="polite">
    <div class="boot-card">
      <div class="boot-top">
        <div class="boot-badge">Go</div>
        <div>
          <p class="boot-heading">Preparing chat runtime</p>
          <p class="boot-subheading">Go WASM bootstrap</p>
        </div>
      </div>
      <div class="boot-status-row">
        <p id="boot-status-text" class="boot-status-text">Connecting to /app/chat.wasm</p>
        <div class="boot-metrics">
          <span id="boot-percent" class="boot-percent">0%</span>
          <div class="boot-spinner" aria-hidden="true"></div>
        </div>
      </div>
      <div class="boot-progress-track" aria-hidden="true">
        <div id="boot-progress-fill" class="boot-progress-fill"></div>
      </div>
      <div class="boot-meta">
        <p id="boot-detail" class="boot-detail">Starting download...</p>
        <div class="boot-stage">
          <span class="boot-stage-dot"></span>
          <span id="boot-stage-label">Download</span>
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

const chatBootstrapJS = `const bootShell = document.getElementById('boot-shell');
const bootStatusText = document.getElementById('boot-status-text');
const bootProgressFill = document.getElementById('boot-progress-fill');
const bootPercent = document.getElementById('boot-percent');
const bootDetail = document.getElementById('boot-detail');
const bootStageLabel = document.getElementById('boot-stage-label');
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

function setBootProgress(progress, options) {
  const config = options || {};
  const clamped = Math.max(0, Math.min(100, progress));
  bootProgressValue = Math.max(bootProgressValue, clamped);
  bootProgressFill.classList.toggle('is-indeterminate', !!config.indeterminate);
  if (!config.indeterminate) {
    bootProgressFill.style.width = bootProgressValue + '%';
  }
  bootPercent.textContent = config.indeterminate ? '...' : Math.round(bootProgressValue) + '%';
}

function setBootPhase(statusText, detailText, stageLabel, progress, options) {
  bootStatusText.textContent = statusText;
  bootDetail.textContent = detailText;
  bootStageLabel.textContent = stageLabel;
  setBootProgress(progress, options);
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

function finishBoot() {
  setBootPhase('Launching chat', 'WASM runtime is ready.', 'Ready', 100);
  requestAnimationFrame(() => {
    bootShell.classList.add('is-hidden');
  });
}

function failBoot(err) {
  console.error('WASM failed to load:', err);
  bootShell.classList.add('boot-shell-error');
  const errorText = err && err.stack ? err.stack : String(err);
  setBootPhase('Could not load /app/chat.wasm', errorText, 'Error', 100);
  document.getElementById('app').textContent = 'Could not load /app/chat.wasm. ' + errorText;
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
  if (contentEncoding && contentEncoding !== 'identity') {
    setBootPhase(
      'Downloading /app/chat.wasm',
      'Compressed WebAssembly stream detected (' + contentEncoding + '). Waiting for the browser to decode and compile it...',
      'Compile',
      88,
      { indeterminate: true }
    );
    const result = await WebAssembly.instantiateStreaming(Promise.resolve(response), go.importObject);
    setBootPhase('Starting Go runtime', 'Decoded WebAssembly. Waiting for the chat UI to mount...', 'Startup', 98, { indeterminate: true });
    const runPromise = go.run(result.instance);
    await waitForAppMount(8000);
    finishBoot();
    await runPromise;
    return;
  }

  if (!response.body || typeof response.body.getReader !== 'function') {
    setBootPhase('Downloading /app/chat.wasm', 'Streaming progress is unavailable in this browser.', 'Download', 34, { indeterminate: true });
    const result = await WebAssembly.instantiateStreaming(Promise.resolve(response), go.importObject);
    setBootPhase('Decompressing and compiling WebAssembly', 'Finalizing the Go runtime.', 'Compile', 92, { indeterminate: true });
    setBootPhase('Starting Go runtime', 'Download complete. Waiting for the chat UI to mount...', 'Startup', 98, { indeterminate: true });
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

  setBootPhase('Downloading /app/chat.wasm', totalBytes > 0 ? '0 of ' + formatBytes(totalBytes) : 'Waiting for stream size...', 'Download', 4);

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    chunks.push(value);
    receivedBytes += value.byteLength;

    if (totalBytes > 0) {
      const progress = 8 + (receivedBytes / totalBytes) * 68;
      setBootPhase('Downloading /app/chat.wasm', formatBytes(receivedBytes) + ' of ' + formatBytes(totalBytes), 'Download', progress);
    } else {
      unknownSizeProgress = Math.min(72, unknownSizeProgress + 3.5);
      setBootPhase('Downloading /app/chat.wasm', formatBytes(receivedBytes) + ' received', 'Download', unknownSizeProgress, { indeterminate: true });
    }
  }

  setBootPhase('Decompressing and compiling WebAssembly', 'Downloaded ' + formatBytes(receivedBytes) + '. Preparing the chat runtime...', 'Compile', 82, { indeterminate: true });

  const wasmResponse = new Response(new Blob(chunks), {
    headers: { 'Content-Type': 'application/wasm' },
  });
  const result = await WebAssembly.instantiateStreaming(Promise.resolve(wasmResponse), go.importObject);

  setBootPhase('Starting Go runtime', 'Download complete. Waiting for the chat UI to mount...', 'Startup', 98, { indeterminate: true });
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

func serveChatShell(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, strings.Replace(chatShellHTML, "{{BOOT_STYLE}}", chatBootShellStyles, 1))
}

func serveChatBootstrapJS(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	_, _ = fmt.Fprint(w, chatBootstrapJS)
}
