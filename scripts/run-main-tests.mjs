import { spawn } from 'node:child_process';
import { access, readdir } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const repoRoot = path.resolve(__dirname, '..');
const npmCommand = process.platform === 'win32' ? 'npm.cmd' : 'npm';
const npmExecPath = process.env.npm_execpath;

const requestedLanes = parseRequestedLanes(process.argv.slice(2));

const lanes = [
	{
		id: 'go-native',
		title: 'Root native Go tests',
		run: () => runCommand('go', ['test', './...'], { env: buildNativeGoEnv() }),
	},
	{
		id: 'go-wasm',
		title: 'Root js/wasm Go tests',
		run: async () => {
			const packages = await collectWasmTestPackages(repoRoot);
			if (packages.length === 0) {
				console.log('No js/wasm test packages found.');
				return;
			}
			const wasmExec = await resolveWasmExec(repoRoot);
			await runCommand('go', ['test', '-exec', wasmExec, ...packages], {
				env: buildWasmGoEnv(),
			});
		},
	},
	{
		id: 'go-livereload',
		title: 'Nested tools/livereload Go tests',
		run: () => runCommand('go', ['test', './...'], {
			cwd: path.join(repoRoot, 'tools', 'livereload'),
			env: buildNativeGoEnv(),
		}),
	},
	{
		id: 'browser',
		title: 'Main Playwright workspace',
		run: () => runNpmCommand(['--prefix', 'test', 'test', '--', '--reporter=list'], {
			env: buildBrowserTestEnv(),
		}),
	},
	{
		id: 'examples',
		title: 'Examples Playwright suites',
		run: () => runNpmCommand(['--prefix', 'examples', 'run', 'test:all'], {
			env: buildBrowserTestEnv(),
		}),
	},
];

const selectedLanes = requestedLanes.length === 0
	? lanes
	: lanes.filter((lane) => requestedLanes.includes(lane.id));

if (selectedLanes.length === 0) {
	console.error(`No matching lanes found for: ${requestedLanes.join(', ')}`);
	console.error(`Available lanes: ${lanes.map((lane) => lane.id).join(', ')}`);
	process.exit(1);
}

try {
	for (const lane of selectedLanes) {
		console.log(`\n=== ${lane.title} (${lane.id}) ===`);
		await lane.run();
	}
	console.log('\nAll requested test lanes passed.');
} catch (error) {
	process.exitCode = typeof error?.code === 'number' ? error.code : 1;
	if (error instanceof Error && error.message) {
		console.error(`\nTest runner stopped: ${error.message}`);
	}
}

function parseRequestedLanes(args) {
	const requested = [];
	for (let index = 0; index < args.length; index++) {
		const current = args[index];
		if (current === '--lane' && index + 1 < args.length) {
			requested.push(args[index + 1]);
			index++;
		}
	}
	return requested;
}

function runCommand(command, args, options = {}) {
	return new Promise((resolve, reject) => {
		const child = spawn(command, args, {
			cwd: options.cwd ?? repoRoot,
			env: options.env ?? process.env,
			stdio: 'inherit',
			shell: false,
		});
		child.on('error', reject);
		child.on('exit', (code, signal) => {
			if (code === 0) {
				resolve();
				return;
			}
			reject(new Error(`${command} ${args.join(' ')} failed with code ${code ?? 'null'}${signal ? ` (signal ${signal})` : ''}`));
		});
	});
}

function runNpmCommand(args, options = {}) {
	if (npmExecPath) {
		return runCommand(process.execPath, [npmExecPath, ...args], options);
	}
	return runCommand(npmCommand, args, options);
}

function buildNativeGoEnv() {
	const env = { ...process.env };
	delete env.GOOS;
	delete env.GOARCH;
	return env;
}

function buildWasmGoEnv() {
	return {
		...buildNativeGoEnv(),
		GOOS: 'js',
		GOARCH: 'wasm',
	};
}

function buildBrowserTestEnv() {
	return {
		...buildNativeGoEnv(),
		PLAYWRIGHT_WORKERS: process.env.PLAYWRIGHT_WORKERS || '4',
	};
}

async function collectWasmTestPackages(root) {
	const packageDirs = new Set();
	await walk(root, async (entryPath, entry) => {
		if (!entry.isFile() || !entry.name.endsWith('_wasm_test.go')) {
			return;
		}
		packageDirs.add(`./${path.relative(root, path.dirname(entryPath)).replace(/\\/g, '/')}`);
	});
	return Array.from(packageDirs).sort();
}

async function walk(currentPath, visitor) {
	const entries = await readdir(currentPath, { withFileTypes: true });
	for (const entry of entries) {
		if (shouldSkip(entry.name)) {
			continue;
		}
		const entryPath = path.join(currentPath, entry.name);
		if (entry.isDirectory()) {
			await walk(entryPath, visitor);
			continue;
		}
		await visitor(entryPath, entry);
	}
}

function shouldSkip(name) {
	return name === '.git' ||
		name === 'node_modules' ||
		name === 'test-results' ||
		name === 'playwright-report' ||
		name === 'dist' ||
		name === 'coverage';
}

async function resolveWasmExec(root) {
	if (process.env.GO_WASM_EXEC) {
		return process.env.GO_WASM_EXEC;
	}
	if (process.platform === 'win32') {
		return path.join(root, 'tools', 'go_js_wasm_exec.bat');
	}
	throw new Error('GO_WASM_EXEC is not set and the repo only ships a Windows go_js_wasm_exec helper. Set GO_WASM_EXEC to a valid js/wasm executor for this platform.');
}

await access(path.join(repoRoot, 'package.json'));