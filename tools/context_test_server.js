const http = require('http');
const fs = require('fs');
const path = require('path');
const { buildWasm } = require('./build_wasm');

const repoRoot = path.resolve(__dirname, '..');
const host = '127.0.0.1';
const port = 8081;

const mimeTypes = {
  '.html': 'text/html; charset=utf-8',
  '.js': 'application/javascript; charset=utf-8',
  '.json': 'application/json; charset=utf-8',
  '.css': 'text/css; charset=utf-8',
  '.wasm': 'application/wasm',
  '.ico': 'image/x-icon',
  '.png': 'image/png',
  '.jpg': 'image/jpeg',
  '.jpeg': 'image/jpeg',
  '.svg': 'image/svg+xml',
  '.webmanifest': 'application/manifest+json',
};

function resolveFilePath(urlPath) {
  const sanitizedPath = decodeURIComponent(urlPath.split('?')[0]);
  const requestPath = sanitizedPath === '/' ? '/static/index.html' : sanitizedPath;
  const absolutePath = path.resolve(repoRoot, `.${requestPath}`);

  if (!absolutePath.startsWith(repoRoot)) {
    return null;
  }

  return absolutePath;
}

function sendFile(res, filePath) {
  const extension = path.extname(filePath).toLowerCase();
  const contentType = mimeTypes[extension] || 'application/octet-stream';

  res.writeHead(200, {
    'Content-Type': contentType,
    'Cache-Control': 'no-store',
  });

  fs.createReadStream(filePath).pipe(res);
}

buildWasm();

const server = http.createServer((req, res) => {
  const filePath = resolveFilePath(req.url || '/');
  if (!filePath) {
    res.writeHead(403);
    res.end('Forbidden');
    return;
  }

  fs.stat(filePath, (err, stat) => {
    if (err) {
      res.writeHead(404);
      res.end('Not Found');
      return;
    }

    if (stat.isDirectory()) {
      const indexPath = path.join(filePath, 'index.html');
      fs.stat(indexPath, indexErr => {
        if (indexErr) {
          res.writeHead(404);
          res.end('Not Found');
          return;
        }

        sendFile(res, indexPath);
      });
      return;
    }

    sendFile(res, filePath);
  });
});

server.listen(port, host, () => {
  console.log(`Context test server listening on http://${host}:${port}`);
});

function shutdown() {
  server.close(() => process.exit(0));
}

process.on('SIGINT', shutdown);
process.on('SIGTERM', shutdown);