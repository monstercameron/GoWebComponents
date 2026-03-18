import express from "express";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const repoRoot = path.resolve(__dirname, "..", "..");
const examplesDir = path.join(repoRoot, "examples");
const staticDir = path.join(examplesDir, "static");
const atlasEntryPath = "/86-atlas-commerce-os/atlas-commerce-os.html";
const host = process.env.HOST || "127.0.0.1";
const port = Number(process.env.PORT || "8085");

const app = express();

app.disable("x-powered-by");
app.use((req, res, next) => {
  if (req.path.endsWith(".wasm")) {
    res.type("application/wasm");
  }
  res.setHeader("Cache-Control", "no-store, no-cache, must-revalidate");
  next();
});

app.get("/healthz", (req, res) => {
  res.json({
    ok: true,
    service: "atlas-demo-server",
    entry: atlasEntryPath,
    time: new Date().toISOString()
  });
});

app.get("/", (req, res) => {
  res.redirect(atlasEntryPath);
});

app.get("/examples/static/index.html", (req, res) => {
  res.redirect(atlasEntryPath);
});

app.use("/static", express.static(staticDir, { extensions: ["html"] }));
app.use("/examples/static", express.static(staticDir, { extensions: ["html"] }));
app.use("/examples", express.static(examplesDir, { extensions: ["html"] }));
app.use("/", express.static(examplesDir, { extensions: ["html"] }));

app.use((req, res) => {
  res.status(404).json({
    error: "not_found",
    path: req.originalUrl,
    hint: atlasEntryPath
  });
});

app.listen(port, host, () => {
  console.log(`Atlas demo server listening on http://${host}:${port}`);
  console.log(`Atlas: http://${host}:${port}${atlasEntryPath}`);
});
