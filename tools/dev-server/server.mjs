import express from "express";
import path from "node:path";
import { fileURLToPath } from "node:url";
import fs from "node:fs";
import fsPromises from "node:fs/promises";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const repoRoot = path.resolve(__dirname, "..", "..");
const examplesDir = path.join(repoRoot, "examples");
const staticDir = path.join(examplesDir, "static");

const app = express();
const host = process.env.HOST || "127.0.0.1";
const port = Number(process.env.PORT || "8090");

function requestLogger(req, res, next) {
  const start = Date.now();
  res.on("finish", () => {
    const elapsed = Date.now() - start;
    // Keep logging compact but readable for iterative dev loops.
    console.log(`${res.statusCode} ${req.method} ${req.originalUrl} ${elapsed}ms`);
  });
  next();
}

function devHeaders(req, res, next) {
  if (req.path.endsWith(".wasm")) {
    res.type("application/wasm");
  }
  // Disable cache in dev to avoid stale wasm/js while iterating quickly.
  res.setHeader("Cache-Control", "no-store, no-cache, must-revalidate");
  next();
}

app.disable("x-powered-by");
app.use(requestLogger);
app.use(devHeaders);

if (!fs.existsSync(examplesDir)) {
  console.error(`Examples directory not found: ${examplesDir}`);
  process.exit(1);
}

// Health endpoint for scripts and CI diagnostics.
app.get("/healthz", (req, res) => {
  res.json({
    ok: true,
    service: "gowebcomponents-dev-server",
    time: new Date().toISOString(),
    root: repoRoot
  });
});

async function buildExamplesListing() {
  const entries = await fsPromises.readdir(examplesDir, { withFileTypes: true });
  const exampleDirs = entries
    .filter((e) => e.isDirectory() && /^\d{2}-/.test(e.name))
    .map((e) => e.name)
    .sort();

  const links = [];
  for (const dirName of exampleDirs) {
    const dirPath = path.join(examplesDir, dirName);
    const files = await fsPromises.readdir(dirPath, { withFileTypes: true });
    const html = files.find((f) => f.isFile() && f.name.toLowerCase().endsWith(".html"));
    if (html) {
      links.push({
        name: dirName,
        href: `/examples/${dirName}/${html.name}`
      });
    }
  }
  return links;
}

app.get("/examples/list", async (req, res) => {
  const links = await buildExamplesListing();
  const listItems = links
    .map((l) => `<li><a href="${l.href}">${l.name}</a></li>`)
    .join("");

  const html = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>GoWebComponents Examples</title>
  <style>
    body { font-family: Segoe UI, Arial, sans-serif; margin: 2rem; line-height: 1.5; }
    h1 { margin-top: 0; }
    ul { padding-left: 1.2rem; }
    li { margin: 0.35rem 0; }
    a { color: #0b63ce; text-decoration: none; }
    a:hover { text-decoration: underline; }
    .meta { margin-bottom: 1rem; color: #555; }
  </style>
</head>
<body>
  <h1>GoWebComponents Examples</h1>
  <p class="meta">Generated from example folders under <code>/examples</code>.</p>
  <p><a href="/examples/static/index.html">Open styled showcase page</a></p>
  <ul>${listItems}</ul>
</body>
</html>`;

  res.status(200).type("text/html").send(html);
});

app.get("/examples", (req, res) => {
  res.redirect("/examples/static/index.html");
});

app.get("/examples/", (req, res) => {
  res.redirect("/examples/static/index.html");
});

// Primary static mounts.
app.use("/examples", express.static(examplesDir, { extensions: ["html"] }));
app.use("/", express.static(examplesDir, { extensions: ["html"] }));
app.use("/static", express.static(staticDir, { extensions: ["html"] }));

app.get("/", (req, res) => {
  res.redirect("/examples/static/index.html");
});

app.use((req, res) => {
  res.status(404).json({
    error: "not_found",
    path: req.originalUrl,
    hint: "Try /examples/static/index.html or /examples/01-counter/counter.html"
  });
});

const server = app.listen(port, host, () => {
  console.log(`GoWebComponents dev server listening on http://${host}:${port}`);
  console.log(`Examples: http://${host}:${port}/examples/static/index.html`);
  console.log(`Counter:  http://${host}:${port}/examples/01-counter/counter.html`);
});

server.on("error", (err) => {
  if (err && err.code === "EADDRINUSE") {
    console.error(`Port ${port} is already in use on ${host}.`);
    console.error("Stop the other process or run with a different PORT.");
    process.exit(1);
  }
  console.error("Server failed to start:", err);
  process.exit(1);
});

function shutdown(signal) {
  console.log(`Received ${signal}, shutting down...`);
  server.close(() => {
    console.log("Server stopped.");
    process.exit(0);
  });
}

process.on("SIGINT", () => shutdown("SIGINT"));
process.on("SIGTERM", () => shutdown("SIGTERM"));
