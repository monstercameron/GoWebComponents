import { createServer } from 'http';
import { readFile } from 'fs/promises';
import { extname, join } from 'path';
import { fileURLToPath } from 'url';
import { dirname } from 'path';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

const PORT = 8082;
// Serve from the examples directory (parent of test directory)
const ROOT = join(__dirname, '../examples');

const mimeTypes = {
  '.html': 'text/html',
  '.js': 'text/javascript',
  '.wasm': 'application/wasm',
  '.css': 'text/css',
  '.json': 'application/json',
  '.png': 'image/png',
  '.jpg': 'image/jpeg',
  '.jpeg': 'image/jpeg',
  '.svg': 'image/svg+xml',
};

const server = createServer(async (req, res) => {
  // Default to index.html if root is requested, but for examples we usually request specific files
  let filePath = req.url;
  if (filePath === '/') {
      filePath = '/index.html';
  }
  
  // Remove query string
  filePath = filePath.split('?')[0];
  
  filePath = join(ROOT, filePath);
  
  const ext = extname(filePath);
  const contentType = mimeTypes[ext] || 'application/octet-stream';
  
  try {
    const content = await readFile(filePath);
    res.writeHead(200, { 'Content-Type': contentType });
    res.end(content);
  } catch (err) {
    if (err.code === 'ENOENT') {
      console.log(`404: ${req.url}`);
      res.writeHead(404);
      res.end('Not found');
    } else {
      console.error(`500: ${req.url}`, err);
      res.writeHead(500);
      res.end('Server error');
    }
  }
});

server.listen(PORT, () => {
  console.log(`Examples server running at http://localhost:${PORT}`);
  console.log(`Serving from: ${ROOT}`);
});
