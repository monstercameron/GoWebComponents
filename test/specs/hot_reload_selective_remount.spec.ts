import { test, expect } from '@playwright/test';
import { spawn, spawnSync } from 'node:child_process';
import { readFile, rm, writeFile } from 'node:fs/promises';
import http from 'node:http';
import net from 'node:net';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

import { resolveWorkspaceBuildPath } from '../../scripts/runner-paths.mjs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const repoRoot = path.resolve(__dirname, '..', '..');
const exampleChangedPath = path.join(repoRoot, 'examples', '98-hot-reload', 'changed_panel.go');
const exampleHTMLPath = path.join(repoRoot, 'examples', '98-hot-reload', 'hot-reload.html');
const exampleRoot = path.join(repoRoot, 'examples', '98-hot-reload');
const exampleWasmPath = resolveWorkspaceBuildPath(repoRoot, 'examples', '98-hot-reload', 'main.wasm');

const selectiveVersionConstV1 = 'const changedSubtreeVersion = "v1"';
const selectiveVersionConstV2 = 'const changedSubtreeVersion = "v2"';
const selectiveVersionTextV1 = 'Changed subtree version: v1';
const selectiveVersionTextV2 = 'Changed subtree version: v2';

test.describe.configure({ mode: 'serial' });

test.describe('GoWebComponents selective hot reload remounts', () => {
	let serverProcess;
	let serverOutput = '';
	let originalChangedSource = '';
	let port;
	let pageUrl;

	test.beforeAll(async () => {
		test.setTimeout(150000);

		originalChangedSource = await readFile(exampleChangedPath, 'utf8');
		if (!originalChangedSource.includes(selectiveVersionConstV1)) {
			throw new Error('hot reload selective remount example is not at the expected baseline');
		}
		await rm(exampleWasmPath, { force: true });

		port = await findAvailablePort();
		pageUrl = `http://127.0.0.1:${port}/hot-reload.html`;
		const wasmUrl = `http://127.0.0.1:${port}/main.wasm`;
		serverProcess = startStandaloneDevServer(port, (output) => {
			serverOutput = output;
		});
		await waitForServer([pageUrl, wasmUrl], 120000, serverProcess, () => serverOutput);
	});

	test.afterAll(async () => {
		if (originalChangedSource) {
			await writeFile(exampleChangedPath, originalChangedSource, 'utf8');
		}
		await stopStandaloneDevServer(serverProcess);
	});

	test('preserves unchanged sibling state while remounting the changed subtree after an edit', async ({ page }) => {
		test.setTimeout(90000);

		await page.goto(pageUrl, { waitUntil: 'networkidle', timeout: 30000 });

		await expect(page.locator('#changed-version')).toHaveCount(1, { timeout: 45000 });
		await expect(page.locator('#stable-count')).toHaveText('Stable count: 0');
		await expect(page.locator('#changed-count')).toHaveText('Changed count: 0');
		await expect(page.locator('#changed-version')).toHaveText(selectiveVersionTextV1, { timeout: 45000 });

		const changedPanelHandle = await page.locator('#changed-panel').elementHandle();

		await page.locator('#stable-increment').click();
		await page.locator('#stable-increment').click();
		await page.locator('#changed-increment').click();

		await expect(page.locator('#stable-count')).toHaveText('Stable count: 2');
		await expect(page.locator('#changed-count')).toHaveText('Changed count: 1');

		const updatedSource = originalChangedSource.replace(selectiveVersionConstV1, selectiveVersionConstV2);
		if (updatedSource === originalChangedSource) {
			throw new Error('failed to create changed-component edit for selective hot reload test');
		}
		await writeFile(exampleChangedPath, updatedSource, 'utf8');

		await expect(page.locator('#changed-version')).toHaveCount(1, { timeout: 45000 });
		await expect(page.locator('#changed-version')).toHaveText(selectiveVersionTextV2, { timeout: 45000 });
		await expect(page.locator('#stable-count')).toHaveText('Stable count: 2', { timeout: 15000 });
		await expect(page.locator('#changed-count')).toHaveText('Changed count: 0', { timeout: 15000 });

		const changedNodeRemounted = await page.evaluate((node) => node !== document.querySelector('#changed-panel'), changedPanelHandle);
		expect(changedNodeRemounted).toBe(true);
	});
});

function startStandaloneDevServer(port, onOutput) {
	if (process.platform === 'win32') {
		const processHandle = spawn('powershell', [
			'-NoProfile',
			'-ExecutionPolicy',
			'Bypass',
			'-File',
			path.join(repoRoot, 'tools', 'dev.ps1'),
			'-Main', path.join(repoRoot, 'examples', '98-hot-reload', 'main.go'),
			'-Root', exampleRoot,
			'-Index', exampleHTMLPath,
			'-Output', exampleWasmPath,
			'-Port', String(port),
		], {
			cwd: repoRoot,
			stdio: ['ignore', 'pipe', 'pipe'],
		});
		drainProcessOutput(processHandle, onOutput);
		return processHandle;
	}

	const processHandle = spawn('bash', [
		path.join(repoRoot, 'tools', 'dev.sh'),
		path.join(repoRoot, 'examples', '98-hot-reload', 'main.go'),
		exampleRoot,
		exampleHTMLPath,
		exampleWasmPath,
	], {
		cwd: repoRoot,
		stdio: ['ignore', 'pipe', 'pipe'],
		env: {
			...process.env,
			PORT: String(port),
			HOST: '127.0.0.1',
		},
	});
	drainProcessOutput(processHandle, onOutput);
	return processHandle;
}

function drainProcessOutput(processHandle, onOutput) {
	if (!processHandle) {
		return;
	}
	const lines = [];
	const record = (chunk) => {
		const text = chunk.toString();
		for (const line of text.split(/\r?\n/)) {
			const trimmed = line.trim();
			if (!trimmed) {
				continue;
			}
			lines.push(trimmed);
			if (lines.length > 40) {
				lines.shift();
			}
		}
		onOutput?.(lines.join('\n'));
	};
	processHandle.stdout?.on('data', record);
	processHandle.stderr?.on('data', record);
}

async function stopStandaloneDevServer(serverProcess) {
	if (!serverProcess || serverProcess.killed) {
		return;
	}

	if (process.platform === 'win32') {
		spawnSync('taskkill', ['/pid', String(serverProcess.pid), '/t', '/f'], { stdio: 'ignore' });
		return;
	}

	serverProcess.kill('SIGTERM');
	await new Promise((resolve) => {
		serverProcess.once('exit', resolve);
		setTimeout(resolve, 5000);
	});
}

async function waitForServer(urls, timeoutMs, serverProcess, getOutput) {
	const deadline = Date.now() + timeoutMs;
	let lastError = null;
	const pendingUrls = Array.isArray(urls) ? urls : [urls];

	while (Date.now() < deadline) {
		if (serverProcess && serverProcess.exitCode !== null) {
			const output = getOutput ? getOutput() : '';
			const detail = output ? `\n${output}` : '';
			throw new Error(`standalone dev server exited before becoming ready (exit ${serverProcess.exitCode})${detail}`);
		}
		let allReady = true;
		for (const url of pendingUrls) {
			try {
				const status = await requestStatus(url);
				if (status === 200) {
					continue;
				}
				allReady = false;
				lastError = new Error(`unexpected status ${status} for ${url}`);
				break;
			} catch (error) {
				allReady = false;
				lastError = error;
				break;
			}
		}
		if (allReady) {
			return;
		}
		await new Promise((resolve) => setTimeout(resolve, 500));
	}

	throw lastError || new Error(`timed out waiting for ${pendingUrls.join(', ')}`);
}

function findAvailablePort() {
	return new Promise((resolve, reject) => {
		const server = net.createServer();
		server.unref();
		server.on('error', reject);
		server.listen(0, '127.0.0.1', () => {
			const address = server.address();
			if (!address || typeof address === 'string') {
				server.close(() => reject(new Error('failed to resolve an available port')));
				return;
			}
			server.close((error) => {
				if (error) {
					reject(error);
					return;
				}
				resolve(address.port);
			});
		});
	});
}

function requestStatus(url) {
	return new Promise((resolve, reject) => {
		const request = http.get(url, (response) => {
			response.resume();
			resolve(response.statusCode || 0);
		});
		request.on('error', reject);
		request.setTimeout(5000, () => {
			request.destroy(new Error(`timed out requesting ${url}`));
		});
	});
}