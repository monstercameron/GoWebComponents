import assert from 'node:assert/strict';
import test from 'node:test';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import {
	loadRunnerOverrides,
	resolveArtifactOutputPath,
	resolveBrowserWorkspace,
	resolveConfiguredPath,
	resolveGoWasmExec,
	resolveLivereloadClientScript,
	resolveLivereloadWorkspace,
	resolveWorkspaceBuildRoot,
} from './runner-paths.mjs';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

test('canonical runner-config example matches JS runner schema', () => {
	const workspace = fs.mkdtempSync(path.join(os.tmpdir(), 'gwc-runner-paths-'));
	const configPath = path.join(workspace, 'gwc-runner.json');
	const repoExamplePath = path.resolve(__dirname, '..', 'docs', 'examples', 'gwc-runner.example.json');
	fs.writeFileSync(configPath, fs.readFileSync(repoExamplePath));

	const overrides = loadRunnerOverrides(workspace);
	assert.equal(overrides.configPath, configPath);
	assert.equal(overrides.paths.generatedProjectRoot, 'generated-projects');
	assert.equal(overrides.paths.artifactRoot, 'enterprise-artifacts');
	assert.equal(overrides.paths.workspaceBuildRoot, 'bin');
	assert.equal(overrides.paths.goWasmExec, 'tools/go_js_wasm_exec.bat');
	assert.equal(overrides.paths.browserWorkspace, 'test');
	assert.equal(overrides.paths.livereloadWorkspace, 'tools/livereload');
	assert.equal(overrides.paths.livereloadClientScript, undefined);
});

test('configured JS runner paths resolve relative to the config file', () => {
	const workspace = fs.mkdtempSync(path.join(os.tmpdir(), 'gwc-runner-paths-'));
	const configDir = path.join(workspace, 'config');
	fs.mkdirSync(configDir, { recursive: true });
	fs.writeFileSync(path.join(configDir, 'gwc-runner.json'), JSON.stringify({
		paths: {
			workspaceBuildRoot: '../custom-bin',
			browserWorkspace: '../browser-suite',
		},
	}, null, 2));

	process.env.GWC_RUNNER_CONFIG = path.join(configDir, 'gwc-runner.json');
	try {
		const overrides = loadRunnerOverrides(workspace);
		assert.equal(resolveConfiguredPath(workspace, overrides.paths.browserWorkspace, overrides.configPath), path.join(workspace, 'browser-suite'));
		assert.equal(resolveWorkspaceBuildRoot(workspace), path.join(workspace, 'custom-bin'));
	} finally {
		delete process.env.GWC_RUNNER_CONFIG;
	}
});

test('artifact-root-aware JS output paths fall back to workspace build roots', () => {
	const workspace = fs.mkdtempSync(path.join(os.tmpdir(), 'gwc-runner-paths-'));

	const fallbackPath = resolveArtifactOutputPath(workspace, 'test-results', 'test');
	assert.equal(fallbackPath, path.join(workspace, 'bin', 'test-results', 'test'));

	fs.writeFileSync(path.join(workspace, 'gwc-runner.json'), JSON.stringify({
		paths: {
			artifactRoot: 'enterprise-artifacts',
		},
	}, null, 2));
	const configuredPath = resolveArtifactOutputPath(workspace, 'test-results', 'test');
	assert.equal(configuredPath, path.join(workspace, 'enterprise-artifacts', path.basename(workspace), 'test-results', 'test'));
});

test('JS workspace resolvers mirror launcher config semantics', () => {
	const repoRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'gwc-runner-paths-'));
	const browserWorkspace = path.join(repoRoot, 'browser-suite');
	const livereloadWorkspace = path.join(repoRoot, 'custom-livereload');
	const clientScript = path.join(repoRoot, 'vendor', 'livereload-client.js');
	const wasmExec = path.join(repoRoot, 'tools', 'go_js_wasm_exec.bat');

	fs.mkdirSync(browserWorkspace, { recursive: true });
	fs.mkdirSync(livereloadWorkspace, { recursive: true });
	fs.mkdirSync(path.dirname(clientScript), { recursive: true });
	fs.mkdirSync(path.dirname(wasmExec), { recursive: true });
	fs.writeFileSync(path.join(browserWorkspace, 'package.json'), '{}\n');
	fs.writeFileSync(clientScript, 'console.log("ok");\n');
	fs.writeFileSync(wasmExec, '@echo off\r\n');
	fs.writeFileSync(path.join(repoRoot, 'gwc-runner.json'), JSON.stringify({
		paths: {
			browserWorkspace: 'browser-suite',
			livereloadWorkspace: 'custom-livereload',
			livereloadClientScript: 'vendor/livereload-client.js',
			goWasmExec: 'tools/go_js_wasm_exec.bat',
		},
	}, null, 2));

	assert.equal(resolveBrowserWorkspace(repoRoot, repoRoot), browserWorkspace);
	assert.equal(resolveLivereloadWorkspace(repoRoot, repoRoot), livereloadWorkspace);
	assert.equal(resolveLivereloadClientScript(repoRoot, repoRoot), clientScript);
	assert.equal(resolveGoWasmExec(repoRoot), wasmExec);
});
