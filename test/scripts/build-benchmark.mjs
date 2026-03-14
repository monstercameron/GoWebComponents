import { mkdir } from 'fs/promises';
import { dirname, join } from 'path';
import { fileURLToPath } from 'url';
import { spawn } from 'child_process';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const testsDir = dirname(__dirname);
const repoRoot = dirname(testsDir);
const benchmarkDir = join(testsDir, 'benchmark');
const outDir = join(benchmarkDir, 'bin');
const outFile = join(outDir, 'benchmark.wasm');

await mkdir(outDir, { recursive: true });

const child = spawn(
  'go',
  ['build', '-o', outFile, './test/benchmark'],
  {
    cwd: repoRoot,
    stdio: 'inherit',
    env: {
      ...process.env,
      GOOS: 'js',
      GOARCH: 'wasm',
    },
  },
);

const exitCode = await new Promise((resolve, reject) => {
  child.on('error', reject);
  child.on('close', resolve);
});

if (exitCode !== 0) {
  process.exit(exitCode ?? 1);
}
