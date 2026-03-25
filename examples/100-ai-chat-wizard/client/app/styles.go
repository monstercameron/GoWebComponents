//go:build js && wasm

package app

const chatWizardStyles = `
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
  0% { transform: translateX(-55%); }
  50% { transform: translateX(150%); }
  100% { transform: translateX(-55%); }
}

#chat-input {
  field-sizing: content;
  max-height: 14rem;
  scrollbar-width: thin;
  scrollbar-color: rgba(128, 255, 206, 0.22) rgba(255,255,255,0.04);
}

#chat-input::-webkit-scrollbar {
  width: 0.8rem;
  height: 0.8rem;
}

#chat-input::-webkit-scrollbar-track {
  background: linear-gradient(180deg, rgba(255,255,255,0.045), rgba(255,255,255,0.02));
  border-radius: 999px;
  border: 1px solid rgba(255,255,255,0.035);
}

#chat-input::-webkit-scrollbar-thumb {
  border-radius: 999px;
  border: 2px solid transparent;
  background-clip: padding-box;
  background: linear-gradient(180deg, rgba(141,255,216,0.28), rgba(25,195,125,0.5));
  box-shadow: inset 0 1px 0 rgba(255,255,255,0.14), 0 0 0 1px rgba(15,23,42,0.1);
  transition: background 160ms ease, box-shadow 160ms ease;
}

#chat-input::-webkit-scrollbar-thumb:hover {
  background: linear-gradient(180deg, rgba(171,255,227,0.44), rgba(25,195,125,0.72));
  box-shadow: inset 0 1px 0 rgba(255,255,255,0.2), 0 0 18px rgba(25,195,125,0.18);
}

#chat-input::-webkit-scrollbar-corner {
  background: transparent;
}

.chat-scrollbar {
  scrollbar-width: thin;
  scrollbar-color: rgba(128, 255, 206, 0.22) rgba(255,255,255,0.04);
}

.chat-scrollbar::-webkit-scrollbar {
  width: 0.8rem;
  height: 0.8rem;
}

.chat-scrollbar::-webkit-scrollbar-track {
  background: linear-gradient(180deg, rgba(255,255,255,0.035), rgba(255,255,255,0.015));
  border-radius: 999px;
  border: 1px solid rgba(255,255,255,0.035);
}

.chat-scrollbar::-webkit-scrollbar-thumb {
  border-radius: 999px;
  border: 2px solid transparent;
  background-clip: padding-box;
  background: linear-gradient(180deg, rgba(141,255,216,0.28), rgba(25,195,125,0.5));
  box-shadow: inset 0 1px 0 rgba(255,255,255,0.14), 0 0 0 1px rgba(15,23,42,0.1);
  transition: background 160ms ease, box-shadow 160ms ease;
}

.chat-scrollbar::-webkit-scrollbar-thumb:hover {
  background: linear-gradient(180deg, rgba(171,255,227,0.44), rgba(25,195,125,0.72));
  box-shadow: inset 0 1px 0 rgba(255,255,255,0.2), 0 0 18px rgba(25,195,125,0.18);
}

.chat-scrollbar::-webkit-scrollbar-corner {
  background: transparent;
}

.chat-scrollbar--sidebar::-webkit-scrollbar-track {
  background: linear-gradient(180deg, rgba(255,255,255,0.022), rgba(255,255,255,0.008));
}

.chat-scrollbar--sidebar::-webkit-scrollbar-thumb {
  background: linear-gradient(180deg, rgba(255,255,255,0.12), rgba(126,255,203,0.3));
}

.chat-scrollbar--sidebar::-webkit-scrollbar-thumb:hover {
  background: linear-gradient(180deg, rgba(255,255,255,0.18), rgba(126,255,203,0.44));
}

.chat-scrollbar--panel::-webkit-scrollbar-track {
  background: linear-gradient(180deg, rgba(255,255,255,0.045), rgba(255,255,255,0.02));
}

@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0; }
}

.cursor::after {
  content: "\258D";
  animation: blink 0.9s step-start infinite;
}

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
  background: rgba(255,255,255,0.55);
  animation: thinking-bounce 1.3s ease-in-out infinite;
}

.thinking-dots span:nth-child(2) {
  animation-delay: 0.18s;
}

.thinking-dots span:nth-child(3) {
  animation-delay: 0.36s;
}

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
  will-change: opacity, transform, filter;
}

.thought-section-card:nth-child(2) .thought-section-heading-enter {
  animation-delay: 45ms;
}

.thought-section-card:nth-child(3) .thought-section-heading-enter {
  animation-delay: 90ms;
}

.thought-section-card:nth-child(4) .thought-section-heading-enter {
  animation-delay: 135ms;
}

.thought-section-card:nth-child(5) .thought-section-heading-enter {
  animation-delay: 180ms;
}

@keyframes thought-heading-flicker {
  0%, 100% {
    opacity: 1;
    text-shadow: 0 0 0 rgba(154,247,208,0);
  }
  33% {
    opacity: 0.94;
    text-shadow: 0 0 8px rgba(154,247,208,0.1);
  }
  58% {
    opacity: 0.985;
    text-shadow: 0 0 12px rgba(154,247,208,0.14);
  }
  74% {
    opacity: 0.96;
    text-shadow: 0 0 6px rgba(154,247,208,0.08);
  }
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

@media (prefers-reduced-motion: reduce) {
  .thought-section-heading-enter,
  .thought-section-heading-streaming .thought-section-title,
  .thought-section-live {
    animation: none !important;
  }
}

.prose {
  line-height: 1.7;
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
  font-weight: 600;
  margin: 1em 0 0.4em;
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
  background: #1e1e1e;
  color: #e5e7eb;
  border-radius: 0.5rem;
  padding: 0.9em 1.1em;
  overflow-x: auto;
  font-size: 0.85em;
  margin: 0;
  white-space: pre;
  font-family: ui-monospace, "Cascadia Code", Menlo, Consolas, monospace;
}

.prose code {
  background: rgba(0,0,0,0.12);
  border-radius: 0.25rem;
  padding: 0.15em 0.35em;
  font-size: 0.87em;
  font-family: ui-monospace, "Cascadia Code", Menlo, Consolas, monospace;
}

.prose pre code {
  background: none;
  padding: 0;
}

.prose blockquote {
  border-left: 3px solid #6b7280;
  padding-left: 0.9em;
  color: #6b7280;
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
  border: 1px solid #374151;
  padding: 0.4em 0.7em;
}

.prose th {
  background: #1f2937;
}

.prose hr {
  border: none;
  border-top: 1px solid #374151;
  margin: 1em 0;
}

.prose a {
  color: #818cf8;
  text-decoration: underline;
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
  border-radius: 1rem;
  border: 1px solid rgba(255,255,255,0.08);
  background: linear-gradient(180deg, rgba(255,255,255,0.045), rgba(255,255,255,0.02));
  box-shadow: inset 0 1px 0 rgba(255,255,255,0.05), 0 14px 34px rgba(0,0,0,0.12);
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
  background: rgba(255,255,255,0.18);
}

.prose .katex-display::-webkit-scrollbar-track {
  background: transparent;
}

.prose-pre-wrap {
  position: relative;
  margin: 0.75em 0;
  border-radius: 0.5rem;
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
  background: rgba(255,255,255,0.07);
  color: rgba(229,231,235,0.65);
  border: 1px solid rgba(255,255,255,0.1);
  border-radius: 0.3rem;
  padding: 0.15em 0.55em;
  font-size: 0.75em;
  cursor: pointer;
  line-height: 1.5;
  transition: background 0.15s, color 0.15s;
}

.prose-copy-btn:hover {
  background: rgba(255,255,255,0.14);
  color: #e5e7eb;
}

.prose-copy-btn.copied {
  color: #34d399;
  border-color: rgba(52,211,153,0.4);
}

.prose-mermaid-wrap {
  position: relative;
  margin: 0.75em 0;
  border-radius: 0.8rem;
  overflow: hidden;
  border: 1px solid rgba(255,255,255,0.08);
  background: linear-gradient(180deg, rgba(255,255,255,0.03), rgba(255,255,255,0.015));
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

.sidebar {
  transition: width 250ms ease, min-width 250ms ease;
}

.sidebar-open {
  width: 260px;
  min-width: 260px;
}

.sidebar-closed {
  width: 0;
  min-width: 0;
}

@keyframes msg-in {
  from { opacity: 0; transform: translateY(8px); }
  to { opacity: 1; transform: translateY(0); }
}

.msg-bubble {
  animation: msg-in 200ms ease both;
}

@keyframes overlay-fade-in {
  from { opacity: 0; }
  to { opacity: 1; }
}

@keyframes modal-slide-in {
  from { opacity: 0; transform: scale(0.96) translateY(6px); }
  to { opacity: 1; transform: scale(1) translateY(0); }
}

.overlay-in {
  animation: overlay-fade-in 180ms ease both;
}

.modal-in {
  animation: modal-slide-in 220ms ease both;
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
  animation: screen-fade-in 260ms cubic-bezier(0.2, 0.8, 0.2, 1) both;
  transform-origin: center top;
}

.thread-screen-ghost {
  pointer-events: none;
  z-index: 30;
}

.control-group-card {
  border: 1px solid rgba(255,255,255,0.08);
  background:
    linear-gradient(180deg, rgba(255,255,255,0.05), rgba(255,255,255,0.02)),
    rgba(47,47,47,0.78);
  box-shadow: inset 0 1px 0 rgba(255,255,255,0.04);
  animation: controlStripIn 220ms ease-out both;
}

.control-group-mobile {
  animation: controlStripIn 220ms ease-out both;
}

.control-group-label {
  color: rgba(255,255,255,0.42);
  font-size: 0.62rem;
  font-weight: 600;
  letter-spacing: 0.16em;
  text-transform: uppercase;
}

.control-group-shell {
  border: 1px solid rgba(255,255,255,0.08);
  background: linear-gradient(180deg, rgba(255,255,255,0.045), rgba(255,255,255,0.018));
  box-shadow: inset 0 1px 0 rgba(255,255,255,0.03);
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
    0 0 0 rgba(25,195,125,0);
}

.toolbar-select:hover:not(:disabled) {
  transform: translateY(-1px);
  border-color: rgba(131,255,210,0.24);
  background: linear-gradient(180deg, rgba(73,73,73,0.98), rgba(47,47,47,0.98));
  box-shadow:
    inset 0 1px 0 rgba(255,255,255,0.06),
    0 10px 24px rgba(0,0,0,0.16),
    0 0 0 1px rgba(25,195,125,0.06);
}

.toolbar-select:focus,
.toolbar-select:focus-visible {
  border-color: rgba(122,255,204,0.44);
  box-shadow:
    inset 0 1px 0 rgba(255,255,255,0.06),
    0 0 0 1px rgba(122,255,204,0.2),
    0 0 0 4px rgba(25,195,125,0.12),
    0 12px 28px rgba(0,0,0,0.2);
}

.toolbar-select:active:not(:disabled) {
  transform: translateY(1px) scale(0.992);
  background: linear-gradient(180deg, rgba(40,40,40,0.98), rgba(54,54,54,0.98));
  box-shadow:
    inset 0 2px 4px rgba(0,0,0,0.24),
    0 4px 12px rgba(0,0,0,0.14),
    0 0 0 1px rgba(25,195,125,0.1);
}

.toolbar-select-option {
  background: #2f2f2f;
  color: rgba(255,255,255,0.86);
  transition: background 140ms ease, color 140ms ease;
}

.toolbar-select-option:hover {
  background: rgba(25,195,125,0.18);
  color: #ffffff;
}

.toolbar-select-option:checked {
  background: linear-gradient(180deg, rgba(88,255,194,0.28), rgba(25,195,125,0.22));
  color: #f2fff9;
}

.control-chip {
  border: 1px solid transparent;
  color: rgba(255,255,255,0.58);
  transition:
    color 180ms ease,
    background 180ms ease,
    border-color 180ms ease,
    transform 180ms ease,
    box-shadow 180ms ease;
}

.control-chip:hover {
  color: rgba(255,255,255,0.88);
  background: rgba(255,255,255,0.08);
  transform: translateY(-1px);
}

.control-chip-active-default {
  color: #050505;
  background: linear-gradient(180deg, rgba(255,255,255,0.98), rgba(242,242,242,0.96));
  border-color: rgba(255,255,255,0.98);
  box-shadow: 0 8px 20px rgba(0,0,0,0.18);
}

.control-chip-active-accent {
  color: #052016;
  background: linear-gradient(180deg, rgba(104,255,197,0.98), rgba(25,195,125,0.94));
  border-color: rgba(120,255,204,0.8);
  box-shadow: 0 10px 24px rgba(25,195,125,0.2);
}

.control-chip-idle {
  background: transparent;
}

@keyframes controlStripIn {
  from { opacity: 0; transform: translateY(-5px); }
  to { opacity: 1; transform: translateY(0); }
}

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
  border: 1px solid rgba(25,195,125,0.22);
  background: linear-gradient(180deg, rgba(15,28,23,0.96), rgba(10,20,17,0.94));
  box-shadow: 0 16px 42px rgba(0,0,0,0.32), 0 0 0 1px rgba(255,255,255,0.04) inset;
  animation: quoteSelectionSpinnerIn 140ms ease-out both;
}

.quote-selection-spinner-dot {
  width: 0.95rem;
  height: 0.95rem;
  border-radius: 999px;
  border: 2px solid rgba(184,255,224,0.18);
  border-top-color: rgba(184,255,224,0.95);
  animation: quoteSelectionSpin 720ms linear infinite;
}

.quote-selection-chip {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.6rem 0.95rem;
  border-radius: 999px;
  border: 1px solid rgba(25,195,125,0.26);
  background: linear-gradient(180deg, rgba(19,37,31,0.98), rgba(11,20,17,0.96));
  color: #d8fff1;
  font-size: 0.92rem;
  font-weight: 600;
  letter-spacing: 0.01em;
  box-shadow: 0 18px 48px rgba(0,0,0,0.34), 0 0 0 1px rgba(255,255,255,0.04) inset;
  transition: transform 180ms ease, border-color 180ms ease, background 180ms ease;
  animation: quoteSelectionChipIn 220ms cubic-bezier(0.16, 1, 0.3, 1) both;
}

.quote-selection-chip:hover {
  transform: translate(-50%, -125%) scale(1.03);
  border-color: rgba(131,255,210,0.42);
  background: linear-gradient(180deg, rgba(25,51,42,0.98), rgba(13,24,20,0.96));
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

.landing-root {
  position: relative;
  isolation: isolate;
  background:
    radial-gradient(circle at 9% 0%, rgba(38,230,166,0.2), transparent 36%),
    radial-gradient(circle at 92% 12%, rgba(111,190,255,0.18), transparent 28%),
    radial-gradient(circle at 50% 120%, rgba(252,211,77,0.12), transparent 34%),
    linear-gradient(180deg, #060a13 0%, #070d17 48%, #050b15 100%);
  font-family: "Space Grotesk", "Avenir Next", "Segoe UI", "Helvetica Neue", sans-serif;
}

.landing-shell {
  position: relative;
  z-index: 2;
}

.landing-header {
  background: rgba(8, 14, 25, 0.58);
  backdrop-filter: blur(18px);
}

.landing-brand-badge {
  width: 2.6rem;
  height: 2.6rem;
  border-radius: 0.95rem;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 800;
  letter-spacing: 0.08em;
  color: #041018;
  background: linear-gradient(135deg, #75ffd0 0%, #26dca4 54%, #a6d8ff 100%);
  box-shadow: 0 10px 40px rgba(38, 220, 164, 0.36);
}

.landing-backdrop-grid {
  position: fixed;
  inset: 0;
  pointer-events: none;
  z-index: 0;
  background:
    linear-gradient(rgba(255,255,255,0.02) 1px, transparent 1px),
    linear-gradient(90deg, rgba(255,255,255,0.02) 1px, transparent 1px);
  background-size: 38px 38px;
  mask-image: radial-gradient(circle at 50% 40%, rgba(0,0,0,0.9), rgba(0,0,0,0.2) 70%, transparent);
}

.landing-orb {
  position: fixed;
  width: 26rem;
  height: 26rem;
  border-radius: 999px;
  filter: blur(48px);
  opacity: 0.5;
  pointer-events: none;
  z-index: 0;
}

.landing-orb-a {
  top: -11rem;
  left: -7rem;
  background: rgba(61, 250, 182, 0.42);
  animation: landingOrbDrift 15s ease-in-out infinite alternate;
}

.landing-orb-b {
  top: 10rem;
  right: -11rem;
  background: rgba(116, 180, 255, 0.34);
  animation: landingOrbDrift 18s ease-in-out infinite alternate-reverse;
}

.landing-orb-c {
  bottom: -12rem;
  left: 30%;
  background: rgba(252, 211, 77, 0.22);
  animation: landingOrbDrift 22s ease-in-out infinite alternate;
}

.landing-glass-card {
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 1.8rem;
  padding: 1.6rem;
  background:
    linear-gradient(170deg, rgba(255,255,255,0.08), rgba(255,255,255,0.025)),
    rgba(10, 17, 31, 0.78);
  box-shadow:
    inset 0 1px 0 rgba(255,255,255,0.1),
    0 26px 70px rgba(0,0,0,0.36);
  backdrop-filter: blur(12px);
}

.landing-soft-card {
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 1.4rem;
  padding: 1.25rem;
  background:
    linear-gradient(180deg, rgba(255,255,255,0.05), rgba(255,255,255,0.02)),
    rgba(7, 14, 27, 0.78);
  box-shadow:
    inset 0 1px 0 rgba(255,255,255,0.06),
    0 16px 46px rgba(0,0,0,0.26);
}

.landing-plan-highlight {
  border-color: rgba(108, 255, 204, 0.4);
  box-shadow:
    inset 0 1px 0 rgba(255,255,255,0.1),
    0 18px 52px rgba(22, 178, 129, 0.22);
}

.landing-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  border-radius: 999px;
  padding: 0.38rem 0.78rem;
  border: 1px solid rgba(129, 255, 211, 0.25);
  background: rgba(129, 255, 211, 0.08);
  color: rgba(194, 255, 230, 0.94);
  font-size: 0.69rem;
  text-transform: uppercase;
  letter-spacing: 0.24em;
  font-weight: 700;
}

.landing-hero-title {
  font-size: clamp(2.05rem, 4.6vw, 4.15rem);
  line-height: 0.98;
  letter-spacing: -0.045em;
  font-weight: 700;
}

.landing-hero-emphasis {
  background: linear-gradient(120deg, #d4ffe9 0%, #79ffd4 40%, #96c8ff 100%);
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
}

.landing-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 0.95rem;
  padding: 0.78rem 1.2rem;
  font-size: 0.84rem;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  text-decoration: none;
  transition: transform 200ms ease, border-color 200ms ease, box-shadow 200ms ease, background 220ms ease, color 220ms ease;
}

.landing-btn:hover {
  transform: translateY(-1px);
}

.landing-btn-primary {
  color: #031914;
  border: 1px solid rgba(128, 255, 205, 0.9);
  background: linear-gradient(125deg, #a4ffe2 0%, #50e8b6 52%, #8fc8ff 100%);
  box-shadow: 0 14px 40px rgba(46, 213, 158, 0.36);
}

.landing-btn-primary:hover {
  box-shadow: 0 18px 48px rgba(46, 213, 158, 0.46);
}

.landing-btn-secondary {
  color: rgba(255,255,255,0.86);
  border: 1px solid rgba(255,255,255,0.14);
  background: rgba(255,255,255,0.04);
}

.landing-btn-secondary:hover {
  border-color: rgba(255,255,255,0.28);
  background: rgba(255,255,255,0.08);
}

.landing-chip {
  border-radius: 999px;
  border: 1px solid rgba(255,255,255,0.12);
  background: rgba(255,255,255,0.05);
  color: rgba(255,255,255,0.75);
  padding: 0.34rem 0.7rem;
  font-size: 0.7rem;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}

.landing-live-dot {
  display: inline-flex;
  align-items: center;
  border-radius: 999px;
  border: 1px solid rgba(94, 255, 188, 0.32);
  background: rgba(94, 255, 188, 0.12);
  color: rgba(167, 255, 223, 0.95);
  font-size: 0.66rem;
  font-weight: 700;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  padding: 0.3rem 0.62rem;
  animation: landingPulse 2s ease-in-out infinite;
}

.landing-kpi-card {
  border: 1px solid rgba(255,255,255,0.1);
  border-radius: 1rem;
  background: rgba(255,255,255,0.04);
  padding: 0.9rem 1rem;
}

.landing-kpi-value {
  margin: 0;
  font-size: clamp(1.5rem, 2.2vw, 2.1rem);
  font-weight: 700;
  letter-spacing: -0.03em;
  color: #ffffff;
}

.landing-kpi-label {
  margin: 0.32rem 0 0;
  font-size: 0.72rem;
  text-transform: uppercase;
  letter-spacing: 0.16em;
  color: rgba(186, 255, 226, 0.8);
}

.landing-feed-item {
  display: flex;
  align-items: flex-start;
  gap: 0.58rem;
  border: 1px solid rgba(255,255,255,0.1);
  border-radius: 0.9rem;
  padding: 0.62rem 0.74rem;
  background: rgba(255,255,255,0.04);
  color: rgba(255,255,255,0.76);
  font-size: 0.77rem;
  line-height: 1.45;
}

.landing-feed-pulse {
  width: 0.48rem;
  height: 0.48rem;
  margin-top: 0.26rem;
  border-radius: 999px;
  background: #52efbc;
  box-shadow: 0 0 0 0 rgba(82, 239, 188, 0.52);
  animation: landingSignal 1.7s ease-out infinite;
  flex-shrink: 0;
}

.landing-step-dot {
  width: 2rem;
  height: 2rem;
  border-radius: 999px;
  border: 1px solid rgba(124, 255, 208, 0.34);
  background: rgba(124, 255, 208, 0.12);
  color: #9dffe1;
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
  from {
    transform: translate3d(0, 0, 0) scale(1);
  }
  to {
    transform: translate3d(12px, 24px, 0) scale(1.1);
  }
}

@keyframes landingPulse {
  0%, 100% { box-shadow: 0 0 0 0 rgba(94,255,188,0.25); }
  50% { box-shadow: 0 0 0 10px rgba(94,255,188,0.02); }
}

@keyframes landingSignal {
  0% { box-shadow: 0 0 0 0 rgba(82,239,188,0.52); }
  80% { box-shadow: 0 0 0 9px rgba(82,239,188,0); }
  100% { box-shadow: 0 0 0 0 rgba(82,239,188,0); }
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
    opacity: 0.42;
  }
}

@media (max-width: 768px) {
  .landing-glass-card {
    padding: 1.2rem;
    border-radius: 1.35rem;
  }

  .landing-soft-card {
    border-radius: 1.15rem;
    padding: 1rem;
  }

  .landing-btn {
    width: 100%;
  }
}

.rd2-root {
  position: relative;
  isolation: isolate;
  background:
    radial-gradient(circle at 0% -10%, rgba(64, 228, 186, 0.22), transparent 38%),
    radial-gradient(circle at 100% 10%, rgba(107, 157, 255, 0.2), transparent 33%),
    radial-gradient(circle at 48% 118%, rgba(245, 186, 76, 0.16), transparent 30%),
    linear-gradient(180deg, #060911 0%, #070d17 44%, #040914 100%);
  font-family: "Space Grotesk", "Sora", "Avenir Next", "Segoe UI", sans-serif;
}

.rd2-grid-overlay {
  position: fixed;
  inset: 0;
  z-index: 0;
  pointer-events: none;
  background:
    linear-gradient(rgba(255,255,255,0.02) 1px, transparent 1px),
    linear-gradient(90deg, rgba(255,255,255,0.02) 1px, transparent 1px);
  background-size: 42px 42px;
  mask-image: radial-gradient(circle at 50% 40%, rgba(0,0,0,0.9), transparent 80%);
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
  background: rgba(53, 234, 178, 0.44);
  animation: rd2LightDrift 16s ease-in-out infinite alternate;
}

.rd2-light-b {
  width: 24rem;
  height: 24rem;
  top: 6rem;
  right: -10rem;
  background: rgba(123, 167, 255, 0.38);
  animation: rd2LightDrift 20s ease-in-out infinite alternate-reverse;
}

.rd2-light-c {
  width: 22rem;
  height: 22rem;
  bottom: -11rem;
  left: 36%;
  background: rgba(245, 186, 76, 0.28);
  animation: rd2LightDrift 22s ease-in-out infinite alternate;
}

.rd2-header {
  background: rgba(7, 13, 25, 0.62);
  backdrop-filter: blur(18px);
}

.rd2-brand-icon {
  width: 2.5rem;
  height: 2.5rem;
  border-radius: 0.88rem;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 800;
  font-size: 1rem;
  color: #04211a;
  background: linear-gradient(140deg, #afffe6 0%, #3adcad 53%, #9bc8ff 100%);
  box-shadow: 0 16px 40px rgba(57, 220, 173, 0.32);
}

.rd2-card {
  border-radius: 1.55rem;
  border: 1px solid rgba(255,255,255,0.12);
  background:
    linear-gradient(160deg, rgba(255,255,255,0.09), rgba(255,255,255,0.02)),
    rgba(9, 16, 30, 0.8);
  box-shadow:
    inset 0 1px 0 rgba(255,255,255,0.08),
    0 24px 70px rgba(0,0,0,0.34);
  backdrop-filter: blur(11px);
  padding: 1.45rem;
}

.rd2-soft-card {
  border-radius: 1.2rem;
  border: 1px solid rgba(255,255,255,0.1);
  background:
    linear-gradient(180deg, rgba(255,255,255,0.05), rgba(255,255,255,0.015)),
    rgba(9, 16, 30, 0.76);
  box-shadow:
    inset 0 1px 0 rgba(255,255,255,0.06),
    0 16px 44px rgba(0,0,0,0.25);
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
  background: radial-gradient(circle, rgba(151, 233, 255, 0.2), transparent 68%);
  pointer-events: none;
}

.rd2-pill {
  display: inline-flex;
  align-items: center;
  border: 1px solid rgba(140, 255, 215, 0.26);
  border-radius: 999px;
  padding: 0.35rem 0.76rem;
  color: rgba(189, 255, 228, 0.92);
  font-size: 0.67rem;
  text-transform: uppercase;
  letter-spacing: 0.24em;
  font-weight: 700;
  background: rgba(140, 255, 215, 0.08);
}

.rd2-hero-title {
  font-size: clamp(2rem, 4.3vw, 4.2rem);
  line-height: 0.96;
  letter-spacing: -0.05em;
  font-weight: 700;
}

.rd2-hero-accent {
  background: linear-gradient(120deg, #d6ffe9 0%, #88ffd6 42%, #9dc6ff 100%);
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
}

.rd2-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 0.92rem;
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
  color: #031d15;
  border: 1px solid rgba(160, 255, 221, 0.88);
  background: linear-gradient(125deg, #b6ffe9 0%, #4de9b8 54%, #9ec6ff 100%);
  box-shadow: 0 16px 42px rgba(57, 220, 173, 0.34);
}

.rd2-btn-primary:hover {
  box-shadow: 0 18px 52px rgba(57, 220, 173, 0.45);
}

.rd2-btn-secondary {
  color: rgba(255,255,255,0.86);
  border: 1px solid rgba(255,255,255,0.18);
  background: rgba(255,255,255,0.04);
}

.rd2-btn-secondary:hover {
  border-color: rgba(255,255,255,0.3);
  background: rgba(255,255,255,0.1);
}

.rd2-mini-metric {
  border: 1px solid rgba(255,255,255,0.1);
  border-radius: 0.95rem;
  padding: 0.78rem 0.82rem;
  background: rgba(255,255,255,0.03);
}

.rd2-mini-value {
  margin: 0;
  font-size: 1.55rem;
  font-weight: 700;
  letter-spacing: -0.03em;
  color: #ffffff;
}

.rd2-mini-label {
  margin: 0.3rem 0 0;
  font-size: 0.7rem;
  letter-spacing: 0.13em;
  text-transform: uppercase;
  color: rgba(181, 255, 226, 0.78);
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
  background: linear-gradient(180deg, rgba(95, 142, 255, 0.06), transparent 34%);
}

.rd2-live-pill {
  border: 1px solid rgba(92, 255, 188, 0.35);
  border-radius: 999px;
  background: rgba(92, 255, 188, 0.13);
  color: rgba(178, 255, 227, 0.94);
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
  border: 1px solid rgba(255,255,255,0.1);
  border-radius: 0.92rem;
  padding: 0.62rem 0.74rem;
  margin-top: 0.55rem;
  background: rgba(255,255,255,0.04);
}

.rd2-lane-dot {
  width: 0.5rem;
  height: 0.5rem;
  border-radius: 999px;
  margin-top: 0.24rem;
  background: #4de9b8;
  box-shadow: 0 0 0 0 rgba(77,233,184,0.44);
  animation: rd2Signal 1.7s ease-out infinite;
  flex-shrink: 0;
}

.rd2-lane-b .rd2-lane-dot {
  background: #8fb5ff;
  box-shadow: 0 0 0 0 rgba(143,181,255,0.44);
}

.rd2-lane-c .rd2-lane-dot {
  background: #f8bf66;
  box-shadow: 0 0 0 0 rgba(248,191,102,0.44);
}

.rd2-queue-bar {
  width: 100%;
  height: 0.48rem;
  border-radius: 999px;
  background: rgba(255,255,255,0.08);
  overflow: hidden;
}

.rd2-queue-bar-fill {
  width: 58%;
  height: 100%;
  border-radius: inherit;
  background: linear-gradient(90deg, #58f0bf 0%, #8bb7ff 48%, #f3be6d 100%);
  animation: rd2BarSlide 2.6s ease-in-out infinite alternate;
}

.rd2-plan-featured {
  border-color: rgba(126, 255, 207, 0.4);
  box-shadow:
    inset 0 1px 0 rgba(255,255,255,0.1),
    0 20px 56px rgba(30, 186, 136, 0.22);
}

.rd2-bullet {
  display: flex;
  align-items: flex-start;
  gap: 0.52rem;
  color: rgba(255,255,255,0.74);
  font-size: 0.79rem;
  line-height: 1.45;
}

.rd2-bullet-dot {
  width: 0.42rem;
  height: 0.42rem;
  border-radius: 999px;
  margin-top: 0.32rem;
  background: #5ef0bf;
  flex-shrink: 0;
}

.rd2-cta-card {
  background:
    linear-gradient(145deg, rgba(87, 239, 188, 0.11), rgba(103, 156, 255, 0.08) 45%, rgba(255, 205, 120, 0.07)),
    rgba(9, 16, 30, 0.84);
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
  0%, 100% { box-shadow: 0 0 0 0 rgba(92,255,188,0.26); }
  50% { box-shadow: 0 0 0 10px rgba(92,255,188,0.01); }
}

@keyframes rd2Signal {
  0% { box-shadow: 0 0 0 0 rgba(77,233,184,0.42); }
  80% { box-shadow: 0 0 0 8px rgba(77,233,184,0); }
  100% { box-shadow: 0 0 0 0 rgba(77,233,184,0); }
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
    opacity: 0.44;
  }
}

@media (max-width: 768px) {
  .rd2-card {
    border-radius: 1.2rem;
    padding: 1rem;
  }

  .rd2-soft-card {
    border-radius: 1rem;
    padding: 0.92rem;
  }

  .rd2-btn {
    width: 100%;
  }
}
`
