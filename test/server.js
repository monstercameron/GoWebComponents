import { createServer } from 'http';
import { readFile } from 'fs/promises';
import { extname, join, normalize } from 'path';
import { fileURLToPath } from 'url';
import { dirname } from 'path';

import { resolveWorkspaceBuildPath } from '../scripts/runner-paths.mjs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const PORT = Number(process.env.PORT || '8083');
const ROOT = join(__dirname, 'testapp');
const WASM_PATH = resolveWorkspaceBuildPath(dirname(__dirname), 'test', 'testapp', 'main.wasm');

const mimeTypes = {
  '.html': 'text/html',
  '.js': 'text/javascript',
  '.wasm': 'application/wasm',
  '.css': 'text/css',
  '.json': 'application/json',
};

const userResponse = JSON.stringify({
  id: 123,
  name: 'Ada Lovelace',
  role: 'runtime-test-user',
});

const server = createServer(async (req, res) => {
  const requestUrl = new URL(req.url, `http://${req.headers.host || '127.0.0.1'}`);

  if (requestUrl.pathname === '/healthz') {
    res.writeHead(200, { 'Content-Type': 'application/json', 'Cache-Control': 'no-store' });
    res.end(JSON.stringify({ ok: true }));
    return;
  }

  if (requestUrl.pathname === '/api/user/123') {
    res.writeHead(200, {
      'Content-Type': 'application/json',
      'Cache-Control': 'no-store',
    });
    res.end(userResponse);
    return;
  }

  let filePath = requestUrl.pathname === '/' ? '/index.html' : requestUrl.pathname;

  if (requestUrl.pathname === '/main.wasm') {
  filePath = WASM_PATH;
  const contentType = mimeTypes['.wasm'];
  try {
    const content = await readFile(filePath);
    res.writeHead(200, {
      'Content-Type': contentType,
      'Cache-Control': 'no-store',
    });
    res.end(content);
  } catch (err) {
    if (err.code === 'ENOENT') {
      res.writeHead(404);
      res.end('Not found');
    } else {
      res.writeHead(500);
      res.end('Server error');
    }
  }
  return;
  }

  filePath = normalize(join(ROOT, filePath));

  if (!filePath.startsWith(ROOT)) {
    res.writeHead(403);
    res.end('Forbidden');
    return;
  }
  
  const ext = extname(filePath);
  const contentType = mimeTypes[ext] || 'application/octet-stream';
  
  try {
    const content = await readFile(filePath);
    res.writeHead(200, {
      'Content-Type': contentType,
      'Cache-Control': ext === '.wasm' ? 'no-store' : 'no-cache',
    });
    res.end(content);
  } catch (err) {
    if (err.code === 'ENOENT') {
      res.writeHead(404);
      res.end('Not found');
    } else {
      res.writeHead(500);
      res.end('Server error');
    }
  }
});

server.listen(PORT, () => {
  console.log(`Server running at http://localhost:${PORT}`);
});
