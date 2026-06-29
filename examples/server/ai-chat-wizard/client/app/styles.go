//go:build js && wasm

package app

const chatWizardStyles = `
:root {
  /* surfaces */
  --bg-0: #08070d;
  --bg-1: #0d0c14;
  --bg-2: #131220;
  --bg-3: #1a1830;
  /* text */
  --text-1: #f2f1f8;
  --text-2: #a4a3b8;
  --text-3: #66657e;
  /* accent: electric violet */
  --accent: #8e7bff;
  --accent-bright: #ab9cff;
  --accent-deep: #6450e8;
  /* secondary accent: mint, used sparingly for live/success */
  --mint: #4de9b8;
  --warn: #f6b84b;
  --danger: #fb7185;
  /* borders */
  --border-1: rgba(255,255,255,0.07);
  --border-2: rgba(255,255,255,0.12);
  --border-accent: rgba(142,123,255,0.35);
  /* shadows */
  --shadow-1: 0 4px 16px rgba(0,0,0,0.35);
  --shadow-2: 0 16px 48px rgba(0,0,0,0.45);
  --glow-accent: 0 0 24px rgba(142,123,255,0.25);
  /* radii */
  --r-sm: 0.5rem;
  --r-md: 0.875rem;
  --r-lg: 1.25rem;
  --r-xl: 1.75rem;
  /* type */
  --font-body: 'Geist', system-ui, sans-serif;
  --font-display: 'Space Grotesk', system-ui, sans-serif;
  --font-mono: 'Geist Mono', ui-monospace, monospace;
  /* motion */
  --ease-out: cubic-bezier(0.22, 1, 0.36, 1);
  --ease-spring: cubic-bezier(0.34, 1.4, 0.44, 1);
}

html {
  margin: 0;
  padding: 0;
  height: 100%;
  overflow: hidden;
  font-size: 93.75%;
}

body {
  margin: 0;
  padding: 0;
  height: 100%;
  overflow: hidden;
  background: var(--bg-0);
  color: var(--text-1);
  font-family: var(--font-body);
}

::selection {
  background: rgba(142,123,255,0.35);
}

:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

button:not(:disabled),
a[href],
summary,
[role="button"]:not([aria-disabled="true"]),
[role="link"]:not([aria-disabled="true"]),
select:not(:disabled),
input[type="button"]:not(:disabled),
input[type="submit"]:not(:disabled) {
  cursor: pointer;
}

button:disabled,
[aria-disabled="true"],
select:disabled,
input[type="button"]:disabled,
input[type="submit"]:disabled {
  cursor: not-allowed;
}

/* ── Reduced motion ─────────────────────────────────────────────────────── */
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
    scroll-behavior: auto !important;
  }
}
.reduce-motion *, .reduce-motion *::before, .reduce-motion *::after {
  animation-duration: 0.01ms !important;
  animation-iteration-count: 1 !important;
  transition-duration: 0.01ms !important;
  scroll-behavior: auto !important;
}

/* ── Typography utilities ──────────────────────────────────────────────── */
.font-display {
  font-family: var(--font-display);
  letter-spacing: -0.02em;
}

.font-mono-tech {
  font-family: var(--font-mono);
}

/* ── App root ──────────────────────────────────────────────────────────── */
#app {
  height: 100dvh;
  display: flex;
}

/* ── Boot shell ────────────────────────────────────────────────────────── */
#boot-shell {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background:
    radial-gradient(circle at 20% 20%, rgba(142,123,255,0.12), transparent 40%),
    radial-gradient(circle at 80% 80%, rgba(100,80,232,0.10), transparent 36%),
    var(--bg-0);
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
  padding: 1.5rem;
  border-radius: var(--r-xl);
  border: 1px solid var(--border-1);
  background: linear-gradient(180deg, rgba(255,255,255,0.05), rgba(255,255,255,0.02));
  backdrop-filter: blur(16px) saturate(1.2);
  box-shadow: var(--shadow-2), inset 0 1px 0 rgba(255,255,255,0.05);
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
  background: linear-gradient(135deg, var(--accent), var(--accent-deep));
  color: white;
  font-weight: 700;
  letter-spacing: 0.04em;
  box-shadow: var(--glow-accent);
}

.boot-heading {
  margin: 0;
  font-size: 1rem;
  font-weight: 600;
  letter-spacing: -0.01em;
  color: var(--text-1);
}

.boot-subheading {
  margin: 0.18rem 0 0;
  font-size: 0.82rem;
  color: var(--text-3);
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
  color: var(--text-2);
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
  color: var(--text-3);
  min-width: 3.2rem;
  text-align: right;
}

.boot-spinner {
  width: 1rem;
  height: 1rem;
  border-radius: 999px;
  border: 2px solid rgba(255,255,255,0.12);
  border-top-color: var(--accent-bright);
  animation: boot-spin 900ms linear infinite;
  flex-shrink: 0;
}

.boot-progress-track {
  position: relative;
  height: 0.52rem;
  border-radius: 999px;
  overflow: hidden;
  background: rgba(255,255,255,0.07);
  box-shadow: inset 0 1px 2px rgba(0,0,0,0.25);
}

.boot-progress-fill {
  position: relative;
  width: 0%;
  height: 100%;
  border-radius: inherit;
  background: linear-gradient(90deg, var(--accent-deep) 0%, var(--accent) 50%, var(--accent-bright) 100%);
  box-shadow: 0 0 12px rgba(142,123,255,0.3);
  transition: width 220ms ease;
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
  color: var(--text-3);
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
  background: rgba(142,123,255,0.08);
  border: 1px solid var(--border-accent);
  white-space: nowrap;
}

.boot-stage-dot {
  width: 0.38rem;
  height: 0.38rem;
  border-radius: 999px;
  background: var(--mint);
  box-shadow: 0 0 6px rgba(77,233,184,0.5);
}

.boot-shell-error .boot-progress-fill {
  background: linear-gradient(90deg, #ef4444 0%, var(--danger) 100%);
  box-shadow: 0 0 9px rgba(251,113,133,0.2);
}

.boot-shell-error .boot-stage-dot {
  background: var(--danger);
  box-shadow: 0 0 6px rgba(251,113,133,0.4);
}

@keyframes boot-spin {
  to { transform: rotate(360deg); }
}

@keyframes boot-shimmer {
  to { transform: translateX(100%); }
}

@keyframes boot-indeterminate {
  0% { transform: translateX(-55%); }
  50% { transform: translateX(150%); }
  100% { transform: translateX(-55%); }
}

/* ── Chat input ─────────────────────────────────────────────────────────── */
#chat-input {
  field-sizing: content;
  max-height: 14rem;
  scrollbar-width: thin;
  scrollbar-color: rgba(142,123,255,0.3) transparent;
}

#chat-input::-webkit-scrollbar {
  width: 0.55rem;
  height: 0.55rem;
}

#chat-input::-webkit-scrollbar-track {
  background: transparent;
  border-radius: 999px;
}

#chat-input::-webkit-scrollbar-thumb {
  border-radius: 999px;
  background: rgba(255,255,255,0.14);
  transition: background 160ms ease;
}

#chat-input::-webkit-scrollbar-thumb:hover {
  background: rgba(142,123,255,0.45);
}

#chat-input::-webkit-scrollbar-corner {
  background: transparent;
}

/* ── Scrollbar system ──────────────────────────────────────────────────── */
.chat-scrollbar {
  scrollbar-width: thin;
  scrollbar-color: rgba(142,123,255,0.3) transparent;
}

.chat-scrollbar::-webkit-scrollbar {
  width: 0.55rem;
  height: 0.55rem;
}

.chat-scrollbar::-webkit-scrollbar-track {
  background: transparent;
  border-radius: 999px;
}

.chat-scrollbar::-webkit-scrollbar-thumb {
  border-radius: 999px;
  background: rgba(255,255,255,0.14);
  transition: background 160ms ease;
}

.chat-scrollbar::-webkit-scrollbar-thumb:hover {
  background: rgba(142,123,255,0.45);
}

.chat-scrollbar::-webkit-scrollbar-corner {
  background: transparent;
}

.chat-scrollbar--sidebar::-webkit-scrollbar-thumb {
  background: rgba(255,255,255,0.10);
}

.chat-scrollbar--sidebar::-webkit-scrollbar-thumb:hover {
  background: rgba(142,123,255,0.38);
}

.chat-scrollbar--panel::-webkit-scrollbar-track {
  background: transparent;
}

/* ── Chat shell backgrounds ─────────────────────────────────────────────── */
.chat-shell {
  position: relative;
  isolation: isolate;
  background:
    radial-gradient(circle at 8% -8%, rgba(142,123,255,0.08), transparent 32%),
    radial-gradient(circle at 94% 6%, rgba(100,80,232,0.07), transparent 28%),
    radial-gradient(circle at 50% 110%, rgba(77,233,184,0.04), transparent 30%),
    var(--bg-0);
}

.chat-toolbar-shell {
  border-color: var(--border-1);
  background: rgba(13,12,20,0.92);
  box-shadow: inset 0 -1px 0 var(--border-1);
}

.chat-thread-surface {
  border: 1px solid var(--border-1);
  border-radius: var(--r-lg);
  background: rgba(13,12,20,0.72);
  box-shadow: var(--shadow-1);
  backdrop-filter: blur(6px);
}

/* ── Sidebar ────────────────────────────────────────────────────────────── */
.sidebar {
  transition: width 250ms ease, min-width 250ms ease;
}

.sidebar-open {
  width: 270px;
  min-width: 270px;
}

.sidebar-closed {
  width: 0;
  min-width: 0;
}

/* ── Cursor ──────────────────────────────────────────────────────────────── */
@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0; }
}

.cursor::after {
  content: "\258D";
  animation: blink 0.9s step-start infinite;
}

/* ── Thinking dots ───────────────────────────────────────────────────────── */
@keyframes thinking-bounce {
  0%, 80%, 100% { transform: translateY(0); opacity: 0.35; }
  40% { transform: translateY(-5px); opacity: 1; }
}

.thinking-dots {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 0;
}

.thinking-dots span {
  display: inline-block;
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--accent-bright), var(--accent));
  box-shadow: 0 0 8px rgba(142,123,255,0.45);
  animation: thinking-bounce 1.3s ease-in-out infinite;
}

.thinking-dots span:nth-child(2) {
  animation-delay: 0.18s;
}

.thinking-dots span:nth-child(3) {
  animation-delay: 0.36s;
}

/* ── Thought section animations ──────────────────────────────────────────── */
@keyframes thought-heading-in {
  from {
    opacity: 0;
    transform: translateY(7px) translateX(-4px);
    filter: blur(3px);
  }
  to {
    opacity: 1;
    transform: translateY(0) translateX(0);
    filter: blur(0);
  }
}

.thought-section-heading-enter {
  animation: thought-heading-in 320ms cubic-bezier(0.2, 0.8, 0.2, 1) both;
  /* --stagger-i is set per card from Go (StyleVar in bubble.go), so the entry
     stagger scales to any section count instead of capping at nth-child(5). */
  animation-delay: calc(var(--stagger-i, 0) * 45ms);
  will-change: opacity, transform, filter;
}

@keyframes thought-heading-flicker {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.92; }
}

.thought-section-heading-streaming .thought-section-title {
  animation: thought-heading-flicker 2.4s ease-in-out infinite;
}

@keyframes thought-live-pulse {
  0%, 100% { opacity: 0.52; }
  50% { opacity: 0.82; }
}

.thought-section-live {
  animation: thought-live-pulse 1.9s ease-in-out infinite;
}

.thought-section-card {
  border: 1px solid var(--border-1);
  border-radius: var(--r-md);
  background: var(--bg-2);
}

@media (prefers-reduced-motion: reduce) {
  .thought-section-heading-enter,
  .thought-section-heading-streaming .thought-section-title,
  .thought-section-live {
    animation: none !important;
  }
}

/* ── Message + screen animations ─────────────────────────────────────────── */
@keyframes msg-in {
  from { opacity: 0; transform: translateY(8px); filter: blur(2px); }
  to { opacity: 1; transform: translateY(0); filter: blur(0); }
}

.msg-bubble {
  animation: msg-in 240ms var(--ease-out) both;
}

@keyframes overlay-fade-in {
  from { opacity: 0; }
  to { opacity: 1; }
}

@keyframes modal-slide-in {
  from { opacity: 0; transform: scale(0.96) translateY(6px); filter: blur(4px); }
  to { opacity: 1; transform: scale(1) translateY(0); filter: blur(0); }
}

.overlay-in {
  animation: overlay-fade-in 180ms ease both;
}

.modal-in {
  animation: modal-slide-in 260ms var(--ease-out) both;
}

@keyframes conv-row-in {
  from { opacity: 0; transform: translateX(-10px) scale(0.985); filter: blur(4px); }
  to { opacity: 1; transform: translateX(0); filter: blur(0); }
}

@keyframes conv-row-out {
  from { opacity: 1; transform: translateX(0) scale(1); filter: blur(0); }
  to { opacity: 0; transform: translateX(14px) scale(0.985); filter: blur(5px); }
}

.conv-row {
  animation: conv-row-in 220ms cubic-bezier(0.2, 0.8, 0.2, 1) both;
  transform-origin: left center;
}

.conv-row-removing {
  overflow: hidden;
  transform-origin: left center;
}

@keyframes screen-fade-in {
  from { opacity: 0; transform: translateY(10px) scale(0.992); filter: blur(8px); }
  to { opacity: 1; transform: translateY(0) scale(1); filter: blur(0); }
}

@keyframes screen-fade-out {
  from { opacity: 1; transform: translateY(0) scale(1); filter: blur(0); }
  to { opacity: 0; transform: translateY(-8px) scale(0.992); filter: blur(10px); }
}

.thread-screen {
  animation: screen-fade-in 280ms cubic-bezier(0.2, 0.8, 0.2, 1) both;
  transform-origin: center top;
}

.thread-screen-ghost {
  pointer-events: none;
  z-index: 30;
}

/* ── Fade-up animations ──────────────────────────────────────────────────── */
@keyframes fade-up {
  from { opacity: 0; transform: translateY(16px); filter: blur(4px); }
  to { opacity: 1; transform: translateY(0); filter: blur(0); }
}

.fade-up { animation: fade-up 300ms var(--ease-out) both; }
.fade-up-d1 { animation-delay: 120ms; }
.fade-up-d2 { animation-delay: 220ms; }
.fade-up-d3 { animation-delay: 320ms; }

/* ── Toolbar controls ────────────────────────────────────────────────────── */
.control-group-card {
  border: 1px solid var(--border-1);
  background: rgba(19,18,32,0.88);
  border-radius: var(--r-md);
  animation: controlStripIn 220ms var(--ease-out) both;
}

.control-group-mobile {
  animation: controlStripIn 220ms var(--ease-out) both;
}

.control-group-label {
  color: var(--text-3);
  font-size: 0.60rem;
  font-weight: 600;
  letter-spacing: 0.16em;
  text-transform: uppercase;
}

.control-group-shell {
  border: 1px solid var(--border-1);
  background: rgba(255,255,255,0.02);
}

.toolbar-select {
  transition:
    transform 160ms ease,
    border-color 180ms ease,
    box-shadow 180ms ease,
    background 180ms ease,
    color 180ms ease;
  box-shadow:
    inset 0 1px 0 rgba(255,255,255,0.04),
    0 0 0 rgba(142,123,255,0);
}

.toolbar-select:hover:not(:disabled) {
  transform: translateY(-1px);
  border-color: rgba(142,123,255,0.28);
  background: rgba(26,24,48,0.98);
  box-shadow:
    inset 0 1px 0 rgba(255,255,255,0.05),
    0 8px 20px rgba(0,0,0,0.18),
    0 0 0 1px rgba(142,123,255,0.1);
}

.toolbar-select:focus,
.toolbar-select:focus-visible {
  outline: none;
  border-color: rgba(142,123,255,0.5);
  box-shadow:
    inset 0 1px 0 rgba(255,255,255,0.05),
    0 0 0 3px rgba(142,123,255,0.25);
}

.toolbar-select:active:not(:disabled) {
  transform: translateY(1px) scale(0.992);
  background: rgba(19,18,32,0.98);
  box-shadow:
    inset 0 2px 4px rgba(0,0,0,0.28),
    0 4px 10px rgba(0,0,0,0.16);
}

.toolbar-select-option {
  background: var(--bg-2);
  color: var(--text-1);
  transition: background 140ms ease, color 140ms ease;
}

.toolbar-select-option:hover {
  background: var(--bg-3);
  color: #ffffff;
}

.toolbar-select-option:checked {
  background: rgba(142,123,255,0.22);
  color: var(--accent-bright);
}

.control-chip {
  border: 1px solid transparent;
  color: var(--text-2);
  transition:
    color 180ms ease,
    background 180ms ease,
    border-color 180ms ease,
    transform 180ms ease,
    box-shadow 180ms ease;
}

.control-chip:hover {
  color: var(--text-1);
  background: rgba(255,255,255,0.07);
  transform: translateY(-1px);
}

.control-chip-active-default {
  color: var(--bg-0);
  background: linear-gradient(180deg, var(--text-1), rgba(210,208,230,0.96));
  border-color: var(--text-1);
  box-shadow: 0 6px 18px rgba(0,0,0,0.22);
}

.control-chip-active-accent {
  color: #fff;
  background: linear-gradient(135deg, var(--accent), var(--accent-deep));
  border-color: var(--border-accent);
  box-shadow: var(--glow-accent);
}

.control-chip-idle {
  background: transparent;
}

@keyframes controlStripIn {
  from { opacity: 0; transform: translateY(-5px); }
  to { opacity: 1; transform: translateY(0); }
}

/* ── Quote selection UI ──────────────────────────────────────────────────── */
.quote-selection-ui {
  position: fixed;
  z-index: 60;
  transform: translate(-50%, -125%);
}

.quote-selection-spinner-card {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 2.3rem;
  height: 2.3rem;
  border-radius: 999px;
  border: 1px solid var(--border-accent);
  background: linear-gradient(180deg, rgba(19,18,32,0.98), rgba(13,12,20,0.96));
  box-shadow: var(--shadow-2), inset 0 1px 0 rgba(255,255,255,0.04);
  animation: quoteSelectionSpinnerIn 140ms ease-out both;
}

.quote-selection-spinner-dot {
  width: 0.95rem;
  height: 0.95rem;
  border-radius: 999px;
  border: 2px solid rgba(142,123,255,0.18);
  border-top-color: var(--accent-bright);
  animation: quoteSelectionSpin 720ms linear infinite;
}

.quote-selection-chip {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.6rem 0.95rem;
  border-radius: 999px;
  border: 1px solid var(--border-accent);
  background: linear-gradient(180deg, rgba(26,24,48,0.98), rgba(13,12,20,0.96));
  color: var(--accent-bright);
  font-size: 0.92rem;
  font-weight: 600;
  letter-spacing: 0.01em;
  box-shadow: var(--shadow-2), inset 0 1px 0 rgba(255,255,255,0.04);
  transition: transform 180ms ease, border-color 180ms ease, background 180ms ease;
  animation: quoteSelectionChipIn 220ms cubic-bezier(0.16, 1, 0.3, 1) both;
}

.quote-selection-chip:hover {
  transform: translate(-50%, -125%) scale(1.03);
  border-color: rgba(171,156,255,0.5);
  background: linear-gradient(180deg, rgba(30,27,55,0.98), rgba(19,18,32,0.96));
}

.quote-selection-chip:active {
  transform: translate(-50%, -125%) scale(0.98);
}

@keyframes quoteSelectionSpin {
  to { transform: rotate(360deg); }
}

@keyframes quoteSelectionSpinnerIn {
  from { opacity: 0; transform: translate(-50%, -118%) scale(0.88); filter: blur(4px); }
  to { opacity: 1; transform: translate(-50%, -125%) scale(1); filter: blur(0); }
}

@keyframes quoteSelectionChipIn {
  from { opacity: 0; transform: translate(-50%, -116%) scale(0.9); filter: blur(6px); }
  to { opacity: 1; transform: translate(-50%, -125%) scale(1); filter: blur(0); }
}

/* ── Prose rendering ─────────────────────────────────────────────────────── */
.prose {
  line-height: 1.7;
  color: var(--text-1);
}

.prose p {
  margin: 0.5em 0;
}

.prose p:first-child {
  margin-top: 0;
}

.prose p:last-child {
  margin-bottom: 0;
}

.prose h1,
.prose h2,
.prose h3,
.prose h4 {
  font-family: var(--font-display);
  font-weight: 600;
  margin: 1em 0 0.4em;
  color: var(--text-1);
}

.prose h1 {
  font-size: 1.4em;
}

.prose h2 {
  font-size: 1.2em;
}

.prose h3 {
  font-size: 1.05em;
}

.prose ul,
.prose ol {
  padding-left: 1.4em;
  margin: 0.5em 0;
}

.prose li {
  margin: 0.2em 0;
}

.prose ul {
  list-style: disc;
}

.prose ol {
  list-style: decimal;
}

.prose pre {
  background: var(--bg-2);
  color: var(--text-1);
  border-radius: var(--r-sm);
  border: 1px solid var(--border-1);
  padding: 0.9em 1.1em;
  overflow-x: auto;
  font-size: 0.85em;
  margin: 0;
  white-space: pre;
  font-family: var(--font-mono);
}

.prose code {
  background: rgba(142,123,255,0.1);
  border-radius: 0.25rem;
  padding: 0.15em 0.35em;
  font-size: 0.87em;
  font-family: var(--font-mono);
  color: var(--accent-bright);
}

.prose pre code {
  background: none;
  padding: 0;
  color: inherit;
}

.prose blockquote {
  border-left: 3px solid var(--accent-deep);
  padding-left: 0.9em;
  color: var(--text-2);
  margin: 0.6em 0;
}

.prose table {
  border-collapse: collapse;
  width: 100%;
  margin: 0.75em 0;
  font-size: 0.9em;
}

.prose th,
.prose td {
  border: 1px solid var(--border-2);
  padding: 0.4em 0.7em;
}

.prose th {
  background: var(--bg-2);
  color: var(--text-1);
  font-family: var(--font-display);
}

.prose hr {
  border: none;
  border-top: 1px solid var(--border-1);
  margin: 1em 0;
}

.prose a {
  color: var(--accent-bright);
  text-decoration: underline;
  text-decoration-color: rgba(142,123,255,0.4);
  transition: color 160ms ease;
}

.prose a:hover {
  color: #fff;
  text-decoration-color: var(--accent-bright);
}

.prose .katex {
  color: inherit;
  font-size: 1em;
}

.prose .katex-display {
  margin: 0.85em 0;
  overflow-x: auto;
  overflow-y: hidden;
  padding: 0.85rem 1rem 0.95rem;
  border-radius: var(--r-md);
  border: 1px solid var(--border-1);
  background: rgba(142,123,255,0.04);
}

.prose .katex-display > .katex {
  display: inline-block;
  min-width: max-content;
}

.prose .katex-display::-webkit-scrollbar {
  height: 0.55rem;
}

.prose .katex-display::-webkit-scrollbar-thumb {
  border-radius: 999px;
  background: rgba(142,123,255,0.28);
}

.prose .katex-display::-webkit-scrollbar-track {
  background: transparent;
}

.prose-pre-wrap {
  position: relative;
  margin: 0.75em 0;
  border-radius: var(--r-sm);
  overflow: hidden;
}

.prose-pre-wrap > pre {
  border-radius: 0;
  margin: 0;
}

.prose-copy-btn {
  position: absolute;
  top: 0.4rem;
  right: 0.4rem;
  background: rgba(142,123,255,0.1);
  color: var(--text-2);
  border: 1px solid var(--border-accent);
  border-radius: 0.3rem;
  padding: 0.15em 0.55em;
  font-size: 0.75em;
  cursor: pointer;
  line-height: 1.5;
  transition: background 0.15s, color 0.15s, box-shadow 0.15s;
}

.prose-copy-btn:hover {
  background: rgba(142,123,255,0.2);
  color: var(--text-1);
  box-shadow: 0 0 0 3px rgba(142,123,255,0.15);
}

.prose-copy-btn.copied {
  color: var(--mint);
  border-color: rgba(77,233,184,0.4);
}

.prose-mermaid-wrap {
  position: relative;
  margin: 0.75em 0;
  border-radius: var(--r-md);
  overflow: hidden;
  border: 1px solid var(--border-1);
  background: var(--bg-2);
}

.prose-mermaid-stage {
  padding: 1rem 1rem 0.8rem;
  overflow-x: auto;
}

.prose .mermaid {
  min-width: max-content;
}

.prose .mermaid svg {
  display: block;
  max-width: 100%;
  height: auto;
  margin: 0 auto;
}

.prose-mermaid-error {
  color: #fecaca;
  background: rgba(127,29,29,0.28);
  border-top: 1px solid rgba(248,113,113,0.28);
  padding: 0.8rem 1rem;
  font-size: 0.82em;
  white-space: pre-wrap;
}

/* ── Marketing page ──────────────────────────────────────────────────────── */
.marketing-page {
  overflow-y: auto;
  overflow-x: hidden;
  height: auto;
}

/* ── Feature cards ───────────────────────────────────────────────────────── */
.feature-card {
  background: var(--bg-2);
  border: 1px solid var(--border-1);
  border-radius: var(--r-lg);
  transition: border-color 200ms ease, transform 200ms ease, box-shadow 200ms ease;
}

.feature-card:hover {
  border-color: var(--border-accent);
  transform: scale(1.01);
  box-shadow: var(--shadow-2), var(--glow-accent);
}

/* ── Pricing cards ───────────────────────────────────────────────────────── */
.pricing-card-featured {
  border: 1px solid var(--border-accent);
  box-shadow: var(--glow-accent), var(--shadow-2);
  background: var(--bg-2);
  transition: box-shadow 300ms ease, transform 220ms ease;
}

.pricing-card-featured:hover {
  transform: translateY(-4px);
  box-shadow: 0 0 52px rgba(142,123,255,0.22), var(--shadow-2);
}

.pricing-card-enterprise {
  border: 1px dashed var(--border-2);
  background: var(--bg-1);
  transition: border-color 250ms ease, transform 220ms ease;
}

.pricing-card-enterprise:hover {
  border-color: rgba(255,255,255,0.28);
  transform: translateY(-3px);
}

/* ── Section divider ─────────────────────────────────────────────────────── */
.section-divider {
  display: flex;
  align-items: center;
  gap: 1rem;
  color: var(--text-3);
  font-family: var(--font-body);
  font-size: 0.6875rem;
  font-weight: 500;
  letter-spacing: 0.22em;
  text-transform: uppercase;
}

.section-divider::before,
.section-divider::after {
  content: '';
  flex: 1;
  height: 1px;
  background: var(--border-1);
}

/* ── CTA button shimmer ──────────────────────────────────────────────────── */
@keyframes cta-shimmer {
  from { background-position: -200% center; }
  to { background-position: 200% center; }
}

.cta-btn {
  position: relative;
  overflow: hidden;
  background: linear-gradient(135deg, var(--accent), var(--accent-deep));
  color: #fff;
  box-shadow: var(--glow-accent);
  transition: transform 200ms ease, box-shadow 200ms ease;
}

.cta-btn:hover {
  transform: translateY(-1px);
  box-shadow: 0 0 32px rgba(142,123,255,0.35);
}

.cta-btn::after {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(
    105deg,
    transparent 38%,
    rgba(255,255,255,0.22) 50%,
    transparent 62%
  );
  background-size: 200% 100%;
  opacity: 0;
  pointer-events: none;
  transition: opacity 180ms ease;
}

.cta-btn:hover::after {
  opacity: 1;
  animation: cta-shimmer 800ms linear;
}

/* ── Nav link ────────────────────────────────────────────────────────────── */
.nav-link {
  position: relative;
}

.nav-link::after {
  content: '';
  position: absolute;
  bottom: -2px;
  left: 0;
  width: 0;
  height: 1px;
  background: var(--accent);
  transition: width 240ms var(--ease-out);
}

.nav-link:hover::after {
  width: 100%;
}

/* ── Status dot live ─────────────────────────────────────────────────────── */
@keyframes status-pulse {
  0%, 100% { opacity: 1; box-shadow: 0 0 0 0 rgba(77,233,184,0.5); }
  50% { opacity: 0.85; box-shadow: 0 0 0 5px rgba(77,233,184,0); }
}

.status-dot-live {
  display: inline-block;
  width: 0.5rem;
  height: 0.5rem;
  border-radius: 50%;
  background: var(--mint);
  animation: status-pulse 2.2s ease-in-out infinite;
}

/* ── Journey progress ────────────────────────────────────────────────────── */
.journey-progress-wrap {
  border: 1px solid var(--border-1);
  border-radius: var(--r-lg);
  background: linear-gradient(180deg, rgba(255,255,255,0.04), rgba(255,255,255,0.01));
  box-shadow: inset 0 1px 0 rgba(255,255,255,0.05);
  padding: 0.62rem 0.72rem;
}

/* ── Page background ─────────────────────────────────────────────────────── */
.page-bg {
  background-color: var(--bg-0);
  background-image:
    url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='60' height='60'%3E%3Cpath d='M30 0 L60 17 L60 43 L30 60 L0 43 L0 17 Z' fill='none' stroke='rgba(142,123,255,0.03)' stroke-width='0.5'/%3E%3C/svg%3E"),
    radial-gradient(ellipse 80% 50% at 50% -10%, rgba(142,123,255,0.07), transparent);
}

/* ── Metric glow ─────────────────────────────────────────────────────────── */
@keyframes metric-glow {
  0%, 100% { text-shadow: 0 0 0 rgba(142,123,255,0); }
  50% { text-shadow: 0 0 18px rgba(142,123,255,0.38), 0 0 36px rgba(142,123,255,0.14); }
}

.metric-glow {
  animation: metric-glow 3.5s ease-in-out infinite;
}

/* ── Hero gradient text ──────────────────────────────────────────────────── */
@keyframes gradient-shift {
  0%, 100% { background-position: 0% 50%; }
  50% { background-position: 100% 50%; }
}

.hero-gradient-text {
  background: linear-gradient(
    135deg,
    var(--accent-bright) 0%,
    var(--accent) 30%,
    var(--mint) 65%,
    var(--accent-bright) 100%
  );
  background-size: 300% 300%;
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  animation: gradient-shift 9s ease infinite;
}

/* ── Hero word stagger ───────────────────────────────────────────────────── */
@keyframes hero-word-in {
  from {
    opacity: 0;
    transform: translateY(24px);
    filter: blur(4px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
    filter: blur(0);
  }
}

.hero-word {
  display: inline-block;
  animation: hero-word-in 600ms var(--ease-out) both;
}

.hero-word:nth-child(1) { animation-delay: 0ms; }
.hero-word:nth-child(2) { animation-delay: 80ms; }
.hero-word:nth-child(3) { animation-delay: 160ms; }
.hero-word:nth-child(4) { animation-delay: 240ms; }
.hero-word:nth-child(5) { animation-delay: 320ms; }
.hero-word:nth-child(6) { animation-delay: 400ms; }
.hero-word:nth-child(7) { animation-delay: 480ms; }
.hero-word:nth-child(8) { animation-delay: 560ms; }

/* ── Scroll-driven reveals ───────────────────────────────────────────────── */
@keyframes scroll-reveal-in {
  from { opacity: 0; transform: translateY(22px); }
  to { opacity: 1; transform: translateY(0); }
}

@supports (animation-timeline: scroll()) {
  .scroll-reveal {
    animation: scroll-reveal-in ease-out both;
    animation-timeline: view();
    animation-range: entry 0% entry 28%;
  }
  .scroll-reveal-d1 { animation-range: entry 6% entry 34%; }
  .scroll-reveal-d2 { animation-range: entry 12% entry 40%; }
}

/* ── Landing page ────────────────────────────────────────────────────────── */
.landing-root {
  position: relative;
  isolation: isolate;
  background:
    radial-gradient(circle at 8% 0%, rgba(142,123,255,0.15), transparent 36%),
    radial-gradient(circle at 92% 14%, rgba(100,80,232,0.12), transparent 30%),
    radial-gradient(circle at 50% 120%, rgba(77,233,184,0.06), transparent 34%),
    var(--bg-0);
  font-family: var(--font-display);
}

.landing-shell {
  position: relative;
  z-index: 2;
}

.landing-header {
  background: rgba(8,7,13,0.68);
  backdrop-filter: blur(18px);
}

.landing-brand-badge {
  width: 2.6rem;
  height: 2.6rem;
  border-radius: var(--r-md);
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 800;
  letter-spacing: 0.04em;
  color: #fff;
  background: linear-gradient(135deg, var(--accent-bright), var(--accent-deep));
  box-shadow: var(--glow-accent);
}

.landing-backdrop-grid {
  position: fixed;
  inset: 0;
  pointer-events: none;
  z-index: 0;
  background:
    linear-gradient(rgba(142,123,255,0.025) 1px, transparent 1px),
    linear-gradient(90deg, rgba(142,123,255,0.025) 1px, transparent 1px);
  background-size: 38px 38px;
  mask-image: radial-gradient(circle at 50% 40%, rgba(0,0,0,0.85), rgba(0,0,0,0.15) 70%, transparent);
}

.landing-orb {
  position: fixed;
  width: 26rem;
  height: 26rem;
  border-radius: 999px;
  filter: blur(52px);
  opacity: 0.45;
  pointer-events: none;
  z-index: 0;
}

.landing-orb-a {
  top: -11rem;
  left: -7rem;
  background: rgba(142,123,255,0.5);
  animation: landingOrbDrift 15s ease-in-out infinite alternate;
}

.landing-orb-b {
  top: 10rem;
  right: -11rem;
  background: rgba(100,80,232,0.4);
  animation: landingOrbDrift 18s ease-in-out infinite alternate-reverse;
}

.landing-orb-c {
  bottom: -12rem;
  left: 30%;
  background: rgba(77,233,184,0.2);
  animation: landingOrbDrift 22s ease-in-out infinite alternate;
}

.landing-glass-card {
  border: 1px solid var(--border-2);
  border-radius: var(--r-xl);
  padding: 1.6rem;
  background:
    linear-gradient(170deg, rgba(255,255,255,0.07), rgba(255,255,255,0.02)),
    rgba(13,12,20,0.8);
  box-shadow:
    inset 0 1px 0 rgba(255,255,255,0.07),
    var(--shadow-2);
  backdrop-filter: blur(16px) saturate(1.2);
}

.landing-soft-card {
  border: 1px solid var(--border-1);
  border-radius: var(--r-lg);
  padding: 1.25rem;
  background:
    linear-gradient(180deg, rgba(255,255,255,0.04), rgba(255,255,255,0.015)),
    rgba(13,12,20,0.76);
  box-shadow:
    inset 0 1px 0 rgba(255,255,255,0.05),
    var(--shadow-1);
}

.landing-plan-highlight {
  border-color: var(--border-accent);
  box-shadow:
    inset 0 1px 0 rgba(255,255,255,0.08),
    var(--glow-accent), var(--shadow-2);
}

.landing-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  border-radius: 999px;
  padding: 0.38rem 0.78rem;
  border: 1px solid var(--border-accent);
  background: rgba(142,123,255,0.1);
  color: var(--accent-bright);
  font-size: 0.69rem;
  text-transform: uppercase;
  letter-spacing: 0.24em;
  font-weight: 700;
}

.landing-hero-title {
  font-size: clamp(2.6rem, 6vw, 4.4rem);
  line-height: 1.04;
  letter-spacing: -0.03em;
  font-weight: 700;
}

.landing-hero-emphasis {
  background: linear-gradient(120deg, var(--accent-bright) 0%, var(--accent) 45%, var(--mint) 100%);
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
}

.landing-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--r-md);
  padding: 0.78rem 1.2rem;
  font-size: 0.84rem;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  text-decoration: none;
  transition: transform 200ms ease, border-color 200ms ease, box-shadow 200ms ease, background 220ms ease, color 220ms ease;
}

.landing-btn:hover {
  transform: translateY(-1px);
}

.landing-btn-primary {
  color: #fff;
  border: 1px solid rgba(142,123,255,0.7);
  background: linear-gradient(135deg, var(--accent), var(--accent-deep));
  box-shadow: var(--glow-accent);
}

.landing-btn-primary:hover {
  box-shadow: 0 0 36px rgba(142,123,255,0.4), var(--shadow-2);
}

.landing-btn-secondary {
  color: var(--text-2);
  border: 1px solid var(--border-2);
  background: rgba(255,255,255,0.03);
}

.landing-btn-secondary:hover {
  border-color: rgba(255,255,255,0.22);
  background: rgba(255,255,255,0.07);
  color: var(--text-1);
}

.landing-chip {
  border-radius: 999px;
  border: 1px solid var(--border-2);
  background: rgba(255,255,255,0.04);
  color: var(--text-2);
  padding: 0.34rem 0.7rem;
  font-size: 0.7rem;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}

.landing-live-dot {
  display: inline-flex;
  align-items: center;
  border-radius: 999px;
  border: 1px solid rgba(77,233,184,0.32);
  background: rgba(77,233,184,0.1);
  color: rgba(77,233,184,0.95);
  font-size: 0.66rem;
  font-weight: 700;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  padding: 0.3rem 0.62rem;
  animation: landingPulse 2s ease-in-out infinite;
}

.landing-kpi-card {
  border: 1px solid var(--border-1);
  border-radius: var(--r-md);
  background: rgba(255,255,255,0.03);
  padding: 0.9rem 1rem;
}

.landing-kpi-value {
  margin: 0;
  font-size: clamp(1.5rem, 2.2vw, 2.1rem);
  font-weight: 700;
  letter-spacing: -0.03em;
  color: var(--text-1);
}

.landing-kpi-label {
  margin: 0.32rem 0 0;
  font-size: 0.72rem;
  text-transform: uppercase;
  letter-spacing: 0.16em;
  color: var(--text-2);
}

.landing-feed-item {
  display: flex;
  align-items: flex-start;
  gap: 0.58rem;
  border: 1px solid var(--border-1);
  border-radius: var(--r-md);
  padding: 0.62rem 0.74rem;
  background: rgba(255,255,255,0.03);
  color: var(--text-2);
  font-size: 0.77rem;
  line-height: 1.45;
}

.landing-feed-pulse {
  width: 0.48rem;
  height: 0.48rem;
  margin-top: 0.26rem;
  border-radius: 999px;
  background: var(--mint);
  box-shadow: 0 0 0 0 rgba(77,233,184,0.5);
  animation: landingSignal 1.7s ease-out infinite;
  flex-shrink: 0;
}

.landing-step-dot {
  width: 2rem;
  height: 2rem;
  border-radius: 999px;
  border: 1px solid var(--border-accent);
  background: rgba(142,123,255,0.1);
  color: var(--accent-bright);
  font-size: 0.8rem;
  font-weight: 700;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.landing-reveal {
  opacity: 0;
  transform: translateY(14px) scale(0.992);
  animation: landingReveal 680ms cubic-bezier(0.21, 0.87, 0.25, 1) forwards;
}

.landing-reveal-delay-1 {
  animation-delay: 90ms;
}

.landing-reveal-delay-2 {
  animation-delay: 170ms;
}

@keyframes landingOrbDrift {
  from { transform: translate3d(0, 0, 0) scale(1); }
  to { transform: translate3d(12px, 24px, 0) scale(1.1); }
}

@keyframes landingPulse {
  0%, 100% { box-shadow: 0 0 0 0 rgba(77,233,184,0.25); }
  50% { box-shadow: 0 0 0 10px rgba(77,233,184,0.02); }
}

@keyframes landingSignal {
  0% { box-shadow: 0 0 0 0 rgba(77,233,184,0.5); }
  80% { box-shadow: 0 0 0 9px rgba(77,233,184,0); }
  100% { box-shadow: 0 0 0 0 rgba(77,233,184,0); }
}

@keyframes landingReveal {
  from {
    opacity: 0;
    transform: translateY(14px) scale(0.992);
    filter: blur(8px);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
    filter: blur(0);
  }
}

@media (max-width: 1024px) {
  .landing-orb {
    width: 20rem;
    height: 20rem;
    opacity: 0.38;
  }
}

@media (max-width: 768px) {
  .landing-glass-card {
    padding: 1.2rem;
    border-radius: var(--r-lg);
  }

  .landing-soft-card {
    border-radius: var(--r-md);
    padding: 1rem;
  }

  .landing-btn {
    width: 100%;
  }
}

/* ── RD2 marketing page ───────────────────────────────────────────────────── */
.rd2-root {
  position: relative;
  isolation: isolate;
  background:
    radial-gradient(circle at 0% -10%, rgba(142,123,255,0.16), transparent 38%),
    radial-gradient(circle at 100% 12%, rgba(100,80,232,0.14), transparent 32%),
    radial-gradient(circle at 50% 118%, rgba(77,233,184,0.07), transparent 30%),
    var(--bg-0);
  font-family: var(--font-display);
}

.rd2-grid-overlay {
  position: fixed;
  inset: 0;
  z-index: 0;
  pointer-events: none;
  background:
    linear-gradient(rgba(142,123,255,0.025) 1px, transparent 1px),
    linear-gradient(90deg, rgba(142,123,255,0.025) 1px, transparent 1px);
  background-size: 42px 42px;
  mask-image: radial-gradient(circle at 50% 40%, rgba(0,0,0,0.88), transparent 80%);
}

.rd2-light {
  position: fixed;
  z-index: 0;
  pointer-events: none;
  border-radius: 999px;
  filter: blur(58px);
}

.rd2-light-a {
  width: 26rem;
  height: 26rem;
  top: -12rem;
  left: -9rem;
  background: rgba(142,123,255,0.4);
  animation: rd2LightDrift 16s ease-in-out infinite alternate;
}

.rd2-light-b {
  width: 24rem;
  height: 24rem;
  top: 6rem;
  right: -10rem;
  background: rgba(100,80,232,0.36);
  animation: rd2LightDrift 20s ease-in-out infinite alternate-reverse;
}

.rd2-light-c {
  width: 22rem;
  height: 22rem;
  bottom: -11rem;
  left: 36%;
  background: rgba(77,233,184,0.2);
  animation: rd2LightDrift 22s ease-in-out infinite alternate;
}

.rd2-header {
  background: rgba(8,7,13,0.7);
  backdrop-filter: blur(18px);
}

.rd2-brand-icon {
  width: 2.5rem;
  height: 2.5rem;
  border-radius: var(--r-md);
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 800;
  font-size: 1rem;
  color: #fff;
  background: linear-gradient(140deg, var(--accent-bright), var(--accent-deep));
  box-shadow: var(--glow-accent);
}

.rd2-card {
  border-radius: var(--r-xl);
  border: 1px solid var(--border-2);
  background:
    linear-gradient(160deg, rgba(255,255,255,0.07), rgba(255,255,255,0.02)),
    rgba(13,12,20,0.82);
  box-shadow:
    inset 0 1px 0 rgba(255,255,255,0.07),
    var(--shadow-2);
  backdrop-filter: blur(14px) saturate(1.2);
  padding: 1.45rem;
}

.rd2-soft-card {
  border-radius: var(--r-lg);
  border: 1px solid var(--border-1);
  background:
    linear-gradient(180deg, rgba(255,255,255,0.04), rgba(255,255,255,0.01)),
    rgba(13,12,20,0.76);
  box-shadow:
    inset 0 1px 0 rgba(255,255,255,0.05),
    var(--shadow-1);
  padding: 1.1rem;
}

.rd2-hero-card {
  overflow: hidden;
  position: relative;
}

.rd2-hero-card::after {
  content: "";
  position: absolute;
  inset: auto -25% -42% auto;
  width: 18rem;
  height: 18rem;
  border-radius: 999px;
  background: radial-gradient(circle, rgba(142,123,255,0.18), transparent 68%);
  pointer-events: none;
}

.rd2-pill {
  display: inline-flex;
  align-items: center;
  border: 1px solid var(--border-accent);
  border-radius: 999px;
  padding: 0.35rem 0.76rem;
  color: var(--accent-bright);
  font-size: 0.67rem;
  text-transform: uppercase;
  letter-spacing: 0.24em;
  font-weight: 700;
  background: rgba(142,123,255,0.1);
}

.rd2-hero-title {
  font-size: clamp(2rem, 4.3vw, 4.2rem);
  line-height: 1.04;
  letter-spacing: -0.04em;
  font-weight: 700;
}

.rd2-hero-accent {
  background: linear-gradient(120deg, var(--accent-bright) 0%, var(--accent) 42%, var(--mint) 100%);
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
}

.rd2-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--r-md);
  padding: 0.76rem 1.16rem;
  text-decoration: none;
  font-size: 0.8rem;
  text-transform: uppercase;
  letter-spacing: 0.12em;
  font-weight: 600;
  transition: transform 180ms ease, background 200ms ease, border-color 200ms ease, box-shadow 220ms ease;
}

.rd2-btn:hover {
  transform: translateY(-1px);
}

.rd2-btn-primary {
  color: #fff;
  border: 1px solid rgba(142,123,255,0.7);
  background: linear-gradient(135deg, var(--accent), var(--accent-deep));
  box-shadow: var(--glow-accent);
}

.rd2-btn-primary:hover {
  box-shadow: 0 0 36px rgba(142,123,255,0.4), var(--shadow-2);
}

.rd2-btn-secondary {
  color: var(--text-2);
  border: 1px solid var(--border-2);
  background: rgba(255,255,255,0.03);
}

.rd2-btn-secondary:hover {
  border-color: rgba(255,255,255,0.24);
  background: rgba(255,255,255,0.08);
  color: var(--text-1);
}

.rd2-mini-metric {
  border: 1px solid var(--border-1);
  border-radius: var(--r-md);
  padding: 0.78rem 0.82rem;
  background: rgba(255,255,255,0.025);
}

.rd2-mini-value {
  margin: 0;
  font-size: 1.55rem;
  font-weight: 700;
  letter-spacing: -0.03em;
  color: var(--text-1);
}

.rd2-mini-label {
  margin: 0.3rem 0 0;
  font-size: 0.7rem;
  letter-spacing: 0.13em;
  text-transform: uppercase;
  color: var(--text-2);
}

.rd2-queue-card {
  position: relative;
  overflow: hidden;
}

.rd2-queue-card::before {
  content: "";
  position: absolute;
  inset: 0;
  pointer-events: none;
  background: linear-gradient(180deg, rgba(142,123,255,0.05), transparent 34%);
}

.rd2-live-pill {
  border: 1px solid var(--border-accent);
  border-radius: 999px;
  background: rgba(142,123,255,0.1);
  color: var(--accent-bright);
  font-size: 0.64rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.16em;
  padding: 0.28rem 0.56rem;
  animation: rd2Pulse 1.8s ease-in-out infinite;
}

.rd2-lane {
  display: flex;
  align-items: flex-start;
  gap: 0.58rem;
  border: 1px solid var(--border-1);
  border-radius: var(--r-md);
  padding: 0.62rem 0.74rem;
  margin-top: 0.55rem;
  background: rgba(255,255,255,0.025);
}

.rd2-lane-dot {
  width: 0.5rem;
  height: 0.5rem;
  border-radius: 999px;
  margin-top: 0.24rem;
  background: var(--accent);
  box-shadow: 0 0 0 0 rgba(142,123,255,0.44);
  animation: rd2Signal 1.7s ease-out infinite;
  flex-shrink: 0;
}

.rd2-lane-b .rd2-lane-dot {
  background: var(--accent-bright);
  box-shadow: 0 0 0 0 rgba(171,156,255,0.44);
}

.rd2-lane-c .rd2-lane-dot {
  background: var(--mint);
  box-shadow: 0 0 0 0 rgba(77,233,184,0.44);
}

.rd2-queue-bar {
  width: 100%;
  height: 0.48rem;
  border-radius: 999px;
  background: rgba(255,255,255,0.07);
  overflow: hidden;
}

.rd2-queue-bar-fill {
  width: 58%;
  height: 100%;
  border-radius: inherit;
  background: linear-gradient(90deg, var(--accent-deep) 0%, var(--accent) 50%, var(--mint) 100%);
  animation: rd2BarSlide 2.6s ease-in-out infinite alternate;
}

.rd2-plan-featured {
  border-color: var(--border-accent);
  box-shadow: var(--glow-accent), var(--shadow-2);
}

.rd2-bullet {
  display: flex;
  align-items: flex-start;
  gap: 0.52rem;
  color: var(--text-2);
  font-size: 0.79rem;
  line-height: 1.45;
}

.rd2-bullet-dot {
  width: 0.42rem;
  height: 0.42rem;
  border-radius: 999px;
  margin-top: 0.32rem;
  background: var(--accent);
  flex-shrink: 0;
}

.rd2-cta-card {
  background:
    linear-gradient(145deg, rgba(142,123,255,0.1), rgba(100,80,232,0.07) 45%, rgba(77,233,184,0.05)),
    rgba(13,12,20,0.84);
}

.rd2-reveal {
  opacity: 0;
  transform: translateY(14px) scale(0.992);
  animation: rd2Reveal 640ms cubic-bezier(0.22, 0.85, 0.27, 1) forwards;
}

.rd2-reveal-d1 {
  animation-delay: 80ms;
}

.rd2-reveal-d2 {
  animation-delay: 160ms;
}

@keyframes rd2LightDrift {
  from { transform: translate3d(0, 0, 0) scale(1); }
  to { transform: translate3d(14px, 22px, 0) scale(1.08); }
}

@keyframes rd2Pulse {
  0%, 100% { box-shadow: 0 0 0 0 rgba(142,123,255,0.26); }
  50% { box-shadow: 0 0 0 10px rgba(142,123,255,0.01); }
}

@keyframes rd2Signal {
  0% { box-shadow: 0 0 0 0 rgba(142,123,255,0.42); }
  80% { box-shadow: 0 0 0 8px rgba(142,123,255,0); }
  100% { box-shadow: 0 0 0 0 rgba(142,123,255,0); }
}

@keyframes rd2BarSlide {
  from { transform: translateX(0); }
  to { transform: translateX(16%); }
}

@keyframes rd2Reveal {
  from {
    opacity: 0;
    transform: translateY(14px) scale(0.992);
    filter: blur(8px);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
    filter: blur(0);
  }
}

@media (max-width: 1024px) {
  .rd2-light {
    width: 18rem;
    height: 18rem;
    opacity: 0.4;
  }
}

@media (max-width: 768px) {
  .rd2-card {
    border-radius: var(--r-lg);
    padding: 1rem;
  }

  .rd2-soft-card {
    border-radius: var(--r-md);
    padding: 0.92rem;
  }

  .rd2-btn {
    width: 100%;
  }
}

/* ── Reduced motion: marketing elements ──────────────────────────────────── */
@media (prefers-reduced-motion: reduce) {
  .scroll-reveal,
  .hero-gradient-text,
  .metric-glow,
  .rd2-live-pill,
  .landing-live-dot,
  .rd2-queue-bar-fill,
  .landing-feed-pulse,
  .rd2-lane-dot {
    animation: none !important;
  }
  .hero-gradient-text {
    -webkit-text-fill-color: var(--accent-bright);
  }
  .cta-btn::after,
  .nav-link::after {
    transition-duration: 50ms !important;
  }
}
`
