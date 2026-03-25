import { mkdirSync, readFileSync, readdirSync, statSync, writeFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const scriptDir = path.dirname(fileURLToPath(import.meta.url));
const staticDir = path.resolve(scriptDir, "..");
const examplesDir = path.resolve(staticDir, "..");
const outputDir = path.join(staticDir, "generated");
const outputPath = path.join(outputDir, "tailwind-manifest.html");

const sourceExtensions = new Set([".go", ".html", ".js", ".ts", ".tsx"]);
const skippedDirs = new Set([
  ".git",
  ".npm-cache",
  "bin",
  "css",
  "generated",
  "images",
  "modules",
  "node_modules",
]);

function walk(dirPath, files) {
  for (const entry of readdirSync(dirPath, { withFileTypes: true })) {
    if (entry.isDirectory()) {
      if (!skippedDirs.has(entry.name)) {
        walk(path.join(dirPath, entry.name), files);
      }
      continue;
    }
    if (sourceExtensions.has(path.extname(entry.name))) {
      files.push(path.join(dirPath, entry.name));
    }
  }
}

function escapeHTML(text) {
  return text
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}

function normalizeLiteral(literal) {
  return literal
    .replace(/\\(["\\`])/g, "$1")
    .replace(/\\n/g, " ")
    .replace(/\\r/g, " ")
    .replace(/\\t/g, " ")
    .replace(/\s+/g, " ")
    .trim();
}

function extractLiterals(sourceText) {
  const literals = [];
  const stringPattern = /"(?:\\.|[^"\\])*"|`[^`]*`/gs;
  for (const match of sourceText.matchAll(stringPattern)) {
    const raw = match[0];
    const body = raw.slice(1, -1);
    const normalized = normalizeLiteral(body);
    if (normalized !== "") {
      literals.push(normalized);
    }
  }
  return literals;
}

function buildManifestRows() {
  const files = [];
  walk(examplesDir, files);

  const rows = [];
  for (const filePath of files) {
    const sourceText = readFileSync(filePath, "utf8");
    const relativePath = path.relative(examplesDir, filePath).replaceAll("\\", "/");
    const literals = extractLiterals(sourceText);
    if (literals.length === 0) {
      continue;
    }
    rows.push(`<!-- ${escapeHTML(relativePath)} -->`);
    for (const literal of literals) {
      rows.push(`<div class="${escapeHTML(literal)}"></div>`);
    }
  }
  return rows;
}

mkdirSync(outputDir, { recursive: true });

const rows = buildManifestRows();
const manifest = [
  "<!DOCTYPE html>",
  "<html>",
  "<body>",
  ...rows,
  "</body>",
  "</html>",
  "",
].join("\n");

writeFileSync(outputPath, manifest, "utf8");

const manifestStat = statSync(outputPath);
console.log(`generated ${path.relative(staticDir, outputPath).replaceAll("\\", "/")} (${manifestStat.size} bytes)`);
