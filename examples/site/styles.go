//go:build js && wasm
// +build js,wasm

package main

// siteCSS is the entire design system, authored in Go per the no-.css-file
// goal for the docs site app. It is injected as a <style> node at mount.
const siteCSS = `/* GoWebComponents docs site — refined dark design system.
   One background, one accent, no gradients, no glass. */

:root {
  --bg: #0a0f1a;
  --bg-raised: #0e1524;
  --bg-inset: #070b13;
  --border: rgba(148, 163, 184, 0.14);
  --border-strong: rgba(148, 163, 184, 0.28);
  --text: #e2e8f0;
  --text-muted: #94a3b8;
  --text-faint: #64748b;
  --accent: #22d3ee;
  --accent-dim: rgba(34, 211, 238, 0.12);
  --accent-border: rgba(34, 211, 238, 0.35);
  --ok: #34d399;
  --warn: #fbbf24;
  --mono: ui-monospace, "Cascadia Code", "JetBrains Mono", Menlo, Consolas, monospace;
  --sans: ui-sans-serif, system-ui, "Segoe UI", Roboto, "Helvetica Neue", sans-serif;
  --radius: 10px;
  --content-width: 1080px;
}

* { box-sizing: border-box; }

html {
  scrollbar-gutter: stable;
  scroll-behavior: smooth;
}

body {
  margin: 0;
  background: var(--bg);
  color: var(--text);
  font-family: var(--sans);
  font-size: 16px;
  line-height: 1.6;
  -webkit-font-smoothing: antialiased;
}

a { color: var(--accent); text-decoration: none; }
a:hover { text-decoration: underline; text-underline-offset: 3px; }

code, pre, kbd { font-family: var(--mono); }

::selection { background: var(--accent-dim); }

/* ---------- chrome ---------- */

.site-header {
  position: sticky;
  top: 0;
  z-index: 50;
  background: rgba(10, 15, 26, 0.88);
  backdrop-filter: blur(8px);
  border-bottom: 1px solid var(--border);
}

.site-header-inner {
  max-width: var(--content-width);
  margin: 0 auto;
  padding: 0 24px;
  height: 56px;
  display: flex;
  align-items: center;
  gap: 28px;
}

.site-logo {
  font-weight: 700;
  font-size: 15px;
  letter-spacing: 0.02em;
  color: var(--text);
}
.site-logo:hover { text-decoration: none; }
.site-logo .logo-accent { color: var(--accent); }

.site-nav {
  display: flex;
  gap: 22px;
  font-size: 14px;
  margin-left: auto;
  align-items: center;
}
.site-nav a { color: var(--text-muted); }
.site-nav a:hover { color: var(--text); text-decoration: none; }
.site-nav a[data-active="true"] { color: var(--text); font-weight: 600; }

.search-button {
  display: flex;
  align-items: center;
  gap: 10px;
  border: 1px solid var(--border);
  background: var(--bg-raised);
  color: var(--text-faint);
  border-radius: 8px;
  padding: 5px 10px;
  font-size: 13px;
  font-family: var(--sans);
  cursor: pointer;
}
.search-button:hover { border-color: var(--border-strong); color: var(--text-muted); }
.search-button kbd {
  border: 1px solid var(--border);
  border-radius: 4px;
  padding: 0 5px;
  font-size: 11px;
  color: var(--text-faint);
  background: var(--bg-inset);
}

.site-footer {
  border-top: 1px solid var(--border);
  margin-top: 96px;
}
.site-footer-inner {
  max-width: var(--content-width);
  margin: 0 auto;
  padding: 32px 24px;
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  justify-content: space-between;
  color: var(--text-faint);
  font-size: 13px;
}
.site-footer a { color: var(--text-muted); }

.container {
  max-width: var(--content-width);
  margin: 0 auto;
  padding: 0 24px;
}

/* ---------- landing ---------- */

.hero {
  padding: 88px 0 56px;
}
.hero h1 {
  font-size: clamp(2.4rem, 6vw, 3.6rem);
  line-height: 1.08;
  letter-spacing: -0.025em;
  margin: 0 0 18px;
  font-weight: 800;
}
.hero h1 .hero-accent { color: var(--accent); }
.hero .hero-sub {
  max-width: 560px;
  color: var(--text-muted);
  font-size: 1.12rem;
  margin: 0 0 30px;
}
.hero-ctas { display: flex; gap: 12px; flex-wrap: wrap; }

.button-primary {
  background: var(--accent);
  color: #06222a;
  font-weight: 700;
  padding: 10px 20px;
  border-radius: 8px;
  font-size: 14.5px;
}
.button-primary:hover { text-decoration: none; filter: brightness(1.1); }
.button-secondary {
  border: 1px solid var(--border-strong);
  color: var(--text);
  padding: 10px 20px;
  border-radius: 8px;
  font-size: 14.5px;
  font-weight: 600;
}
.button-secondary:hover { text-decoration: none; border-color: var(--accent-border); }

.hero-demo {
  display: grid;
  grid-template-columns: minmax(0, 1.15fr) minmax(0, 0.85fr);
  gap: 18px;
  margin: 56px 0 0;
}
@media (max-width: 820px) { .hero-demo { grid-template-columns: 1fr; } }

.demo-pane {
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--bg-raised);
  overflow: hidden;
  display: flex;
  flex-direction: column;
}
.demo-pane-title {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 9px 14px;
  font-size: 12.5px;
  font-family: var(--mono);
  color: var(--text-faint);
  border-bottom: 1px solid var(--border);
}
.demo-pane-title .live-dot {
  width: 7px; height: 7px;
  border-radius: 999px;
  background: var(--ok);
  display: inline-block;
}
.demo-pane pre {
  margin: 0;
  padding: 16px 18px;
  font-size: 13px;
  line-height: 1.55;
  overflow-x: auto;
  color: var(--text);
  flex: 1;
}
.demo-pane iframe {
  border: 0;
  width: 100%;
  min-height: 320px;
  flex: 1;
  background: var(--bg-inset);
}

.stats-strip {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 14px;
  margin: 64px 0 0;
}
.stat-card {
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--bg-raised);
  padding: 18px 20px;
}
.stat-value {
  font-size: 1.7rem;
  font-weight: 800;
  letter-spacing: -0.02em;
  color: var(--accent);
  font-family: var(--mono);
}
.stat-label { color: var(--text-muted); font-size: 13px; margin-top: 4px; }
.stat-note { color: var(--text-faint); font-size: 11.5px; margin-top: 6px; }

.section-title {
  font-size: 1.6rem;
  font-weight: 750;
  letter-spacing: -0.02em;
  margin: 88px 0 8px;
}
.section-sub { color: var(--text-muted); margin: 0 0 28px; max-width: 620px; }

.feature-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 14px;
}
.feature-card {
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--bg-raised);
  padding: 20px;
}
.feature-card h3 { margin: 0 0 8px; font-size: 15.5px; }
.feature-card p { margin: 0; color: var(--text-muted); font-size: 14px; }
.feature-card .feature-link { display: inline-block; margin-top: 12px; font-size: 13.5px; }

/* ---------- cards & galleries ---------- */

.card-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 14px;
}

.catalog-card {
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--bg-raised);
  padding: 18px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  transition: border-color 120ms ease;
}
.catalog-card:hover { border-color: var(--accent-border); }
.catalog-card h3 { margin: 0; font-size: 15.5px; }
.catalog-card h3 a { color: var(--text); }
.catalog-card p { margin: 0; color: var(--text-muted); font-size: 13.5px; flex: 1; }
.card-meta { display: flex; flex-wrap: wrap; gap: 6px; }
.chip {
  border: 1px solid var(--border);
  color: var(--text-faint);
  border-radius: 999px;
  font-size: 11.5px;
  padding: 1px 9px;
}
.chip-level { color: var(--accent); border-color: var(--accent-border); }
.card-actions { display: flex; gap: 14px; font-size: 13.5px; }

.filter-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin: 0 0 24px;
}
.filter-chip {
  border: 1px solid var(--border);
  background: var(--bg-raised);
  color: var(--text-muted);
  border-radius: 999px;
  padding: 5px 14px;
  font-size: 13px;
  cursor: pointer;
  font-family: var(--sans);
}
.filter-chip[data-active="true"] {
  border-color: var(--accent-border);
  color: var(--accent);
  background: var(--accent-dim);
}

.page-title { font-size: 2rem; font-weight: 800; letter-spacing: -0.02em; margin: 48px 0 8px; }
.page-sub { color: var(--text-muted); margin: 0 0 32px; max-width: 640px; }

/* ---------- learn layout ---------- */

.learn-layout {
  display: grid;
  grid-template-columns: 250px minmax(0, 1fr);
  gap: 48px;
  padding-top: 40px;
}
@media (max-width: 900px) { .learn-layout { grid-template-columns: 1fr; } }

.learn-sidebar {
  position: sticky;
  top: 80px;
  align-self: start;
  font-size: 13.5px;
  max-height: calc(100vh - 110px);
  overflow-y: auto;
}
@media (max-width: 900px) { .learn-sidebar { position: static; max-height: none; } }
.learn-sidebar .sidebar-group {
  text-transform: uppercase;
  letter-spacing: 0.08em;
  font-size: 11px;
  color: var(--text-faint);
  margin: 18px 0 8px;
}
.learn-sidebar a {
  display: block;
  color: var(--text-muted);
  padding: 4px 10px;
  border-left: 2px solid transparent;
  border-radius: 0 6px 6px 0;
}
.learn-sidebar a:hover { color: var(--text); text-decoration: none; background: var(--bg-raised); }
.learn-sidebar a[data-active="true"] {
  color: var(--accent);
  border-left-color: var(--accent);
  background: var(--accent-dim);
}

/* ---------- prose (markdown / articles) ---------- */

.prose { max-width: 760px; padding-bottom: 48px; }
.prose h1 { font-size: 2rem; font-weight: 800; letter-spacing: -0.02em; line-height: 1.15; margin: 8px 0 16px; }
.prose h2 {
  font-size: 1.35rem; font-weight: 700; letter-spacing: -0.01em;
  margin: 44px 0 12px; padding-top: 18px;
  border-top: 1px solid var(--border);
}
.prose h3 { font-size: 1.08rem; font-weight: 700; margin: 30px 0 10px; }
.prose h4, .prose h5, .prose h6 { font-size: 0.95rem; font-weight: 700; margin: 22px 0 8px; }
.prose p { margin: 0 0 14px; color: var(--text-muted); }
.prose li { color: var(--text-muted); margin: 4px 0; }
.prose ul, .prose ol { padding-left: 24px; margin: 0 0 14px; }
.prose strong { color: var(--text); }
.prose blockquote {
  border-left: 3px solid var(--accent-border);
  margin: 0 0 14px;
  padding: 2px 0 2px 16px;
  color: var(--text-muted);
}
.prose hr { border: 0; border-top: 1px solid var(--border); margin: 32px 0; }
.prose table { border-collapse: collapse; width: 100%; margin: 0 0 18px; font-size: 14px; }
.prose th, .prose td { border: 1px solid var(--border); padding: 7px 12px; text-align: left; }
.prose th { color: var(--text); background: var(--bg-raised); }
.prose td { color: var(--text-muted); }
.prose code {
  background: var(--bg-raised);
  border: 1px solid var(--border);
  border-radius: 5px;
  padding: 1px 6px;
  font-size: 0.86em;
  color: var(--accent);
}
.prose pre {
  background: var(--bg-inset);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 16px 18px;
  overflow-x: auto;
  margin: 0 0 18px;
  position: relative;
  font-size: 13px;
  line-height: 1.55;
}
.prose pre code {
  background: none; border: 0; padding: 0;
  color: var(--text); font-size: inherit;
}

.copy-button {
  position: absolute;
  top: 8px; right: 8px;
  border: 1px solid var(--border);
  background: var(--bg-raised);
  color: var(--text-faint);
  border-radius: 6px;
  font-size: 11.5px;
  padding: 3px 9px;
  cursor: pointer;
  opacity: 0;
  transition: opacity 100ms ease;
  font-family: var(--sans);
}
pre:hover .copy-button { opacity: 1; }
.copy-button:hover { color: var(--text); }

.chapter-pager {
  display: flex;
  justify-content: space-between;
  gap: 14px;
  margin-top: 48px;
  max-width: 760px;
}
.chapter-pager a {
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 12px 16px;
  flex: 1;
  color: var(--text-muted);
  font-size: 13.5px;
}
.chapter-pager a:hover { border-color: var(--accent-border); text-decoration: none; color: var(--text); }
.chapter-pager .pager-label { display: block; color: var(--text-faint); font-size: 11px; text-transform: uppercase; letter-spacing: 0.08em; }
.chapter-pager .pager-next { text-align: right; }

/* ---------- search modal ---------- */

.search-overlay {
  position: fixed;
  inset: 0;
  background: rgba(4, 7, 13, 0.7);
  z-index: 100;
  display: none;
  padding: 12vh 16px 0;
}
.search-overlay[data-open="true"] { display: block; }
.search-panel {
  max-width: 560px;
  margin: 0 auto;
  background: var(--bg-raised);
  border: 1px solid var(--border-strong);
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 24px 64px rgba(0, 0, 0, 0.5);
}
.search-input {
  width: 100%;
  background: transparent;
  border: 0;
  border-bottom: 1px solid var(--border);
  color: var(--text);
  font-size: 15px;
  padding: 14px 18px;
  outline: none;
  font-family: var(--sans);
}
.search-results { max-height: 50vh; overflow-y: auto; }
.search-result {
  display: block;
  padding: 10px 18px;
  border-bottom: 1px solid var(--border);
  color: var(--text);
  font-size: 14px;
}
.search-result:hover, .search-result[data-selected="true"] {
  background: var(--accent-dim);
  text-decoration: none;
}
.search-result .result-kind {
  color: var(--text-faint);
  font-size: 11.5px;
  text-transform: uppercase;
  letter-spacing: 0.07em;
  margin-right: 8px;
}
.search-result .result-snippet {
  display: block;
  color: var(--text-faint);
  font-size: 12.5px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.search-empty { padding: 22px 18px; color: var(--text-faint); font-size: 14px; }

/* ---------- wasm app boot ---------- */

.boot-state {
  min-height: 60vh;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-faint);
  font-family: var(--mono);
  font-size: 14px;
}
.spinner {
  width: 16px; height: 16px;
  border: 2px solid var(--border-strong);
  border-top-color: var(--accent);
  border-radius: 999px;
  margin-right: 10px;
  animation: site-spin 700ms linear infinite;
}
@keyframes site-spin { to { transform: rotate(360deg); } }

/* ---------- source viewer & chroma tokens ---------- */

.source-header {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 14px;
  margin: 48px 0 16px;
}
.source-header h1 { margin: 0; font-size: 1.5rem; font-weight: 800; letter-spacing: -0.02em; }
.source-header .source-path { font-family: var(--mono); font-size: 12.5px; color: var(--text-faint); }
.source-actions { margin-left: auto; display: flex; gap: 12px; font-size: 13.5px; align-items: center; }

.source-pane {
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--bg-inset);
  overflow: hidden;
  margin-bottom: 64px;
}
.source-pane pre {
  margin: 0;
  padding: 18px 20px;
  overflow-x: auto;
  font-size: 13px;
  line-height: 1.6;
}

.tok-kw { color: #f472b6; }
.tok-str { color: #a5d6a7; }
.tok-com { color: #64748b; font-style: italic; }
.tok-num { color: #fbbf24; }
.tok-fn { color: #93c5fd; }
.tok-typ { color: #5eead4; }
.tok-op { color: #94a3b8; }
`
