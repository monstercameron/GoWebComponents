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
`
