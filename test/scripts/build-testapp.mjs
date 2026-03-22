import { mkdir } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { spawn } from 'node:child_process';

import { resolveWorkspaceBuildPath } from '../../scripts/runner-paths.mjs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const testsDir = dirname(__dirname);
const repoRoot = dirname(testsDir);
const testAppDir = join(testsDir, 'testapp');
const outFile = resolveWorkspaceBuildPath(repoRoot, 'test', 'testapp', 'main.wasm');

await mkdir(dirname(outFile), { recursive: true });

const child = spawn(
	'go',
	['build', '-o', outFile, '.'],
	{
		cwd: testAppDir,
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