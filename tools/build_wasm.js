const fs = require('fs');
const path = require('path');
const { spawnSync } = require('child_process');

function buildWasm() {
  const repoRoot = path.resolve(__dirname, '..');
  const outputDir = path.join(repoRoot, 'static', 'bin');
  const outputFile = path.join(outputDir, 'main.wasm');

  fs.mkdirSync(outputDir, { recursive: true });

  const result = spawnSync('go', ['build', '-o', outputFile], {
    cwd: repoRoot,
    stdio: 'inherit',
    env: {
      ...process.env,
      GOOS: 'js',
      GOARCH: 'wasm',
    },
  });

  if (result.status !== 0) {
    process.exit(result.status || 1);
  }
}

if (require.main === module) {
  buildWasm();
}

module.exports = { buildWasm };