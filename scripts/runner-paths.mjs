import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';

export function resolveRunnerConfigPath(root) {
	const explicit = process.env.GWC_RUNNER_CONFIG?.trim();
	if (explicit) {
		return path.isAbsolute(explicit) ? path.normalize(explicit) : path.resolve(root, explicit);
	}
	const localConfig = path.join(root, 'gwc-runner.json');
	if (fs.existsSync(localConfig)) {
		return localConfig;
	}
	const homeConfig = path.join(os.homedir(), '.gwc', 'runner.json');
	if (fs.existsSync(homeConfig)) {
		return homeConfig;
	}
	return '';
}

export function loadRunnerOverrides(root) {
	const configPath = resolveRunnerConfigPath(root);
	if (!configPath) {
		return { configPath: '', paths: {} };
	}
	const content = fs.readFileSync(configPath, 'utf8');
	const parsed = JSON.parse(content);
	return { configPath, paths: parsed?.paths ?? {} };
}

export function resolveConfiguredPath(root, raw, configPath = '') {
	if (!raw || !String(raw).trim()) {
		return '';
	}
	if (path.isAbsolute(raw)) {
		return path.normalize(raw);
	}
	const baseDir = configPath ? path.dirname(configPath) : root;
	return path.resolve(baseDir, raw);
}

export function resolveWorkspaceBuildRoot(root) {
	const overrides = loadRunnerOverrides(root);
	const configured = resolveConfiguredPath(root, overrides.paths?.workspaceBuildRoot, overrides.configPath);
	if (configured) {
		return configured;
	}
	return path.join(root, 'bin');
}

export function resolveWorkspaceBuildPath(root, ...segments) {
	return path.join(resolveWorkspaceBuildRoot(root), ...segments);
}

export function artifactNamespace(root) {
	const cleaned = path.normalize(path.resolve(root));
	const base = path.basename(cleaned);
	return base && base !== path.sep ? base : 'workspace';
}

export function resolveArtifactRoot(root) {
	const overrides = loadRunnerOverrides(root);
	return resolveConfiguredPath(root, overrides.paths?.artifactRoot, overrides.configPath);
}

export function resolveArtifactOutputPath(root, ...segments) {
	const configured = resolveArtifactRoot(root);
	if (configured) {
		return path.join(configured, artifactNamespace(root), ...segments);
	}
	return resolveWorkspaceBuildPath(root, ...segments);
}

export function resolveGoWasmExec(root) {
	const overrides = loadRunnerOverrides(root);
	const configured = resolveConfiguredPath(root, overrides.paths?.goWasmExec, overrides.configPath);
	if (configured) {
		if (!fs.existsSync(configured)) {
			throw new Error(`configured goWasmExec path does not exist: ${configured}`);
		}
		return configured;
	}
	if (process.env.GO_WASM_EXEC) {
		return process.env.GO_WASM_EXEC;
	}
	if (process.platform === 'win32') {
		return path.join(root, 'tools', 'go_js_wasm_exec.bat');
	}
	throw new Error('GO_WASM_EXEC is not set and the repo only ships a Windows go_js_wasm_exec helper. Set GO_WASM_EXEC to a valid js/wasm executor for this platform.');
}

export function resolveBrowserWorkspace(root, workspaceRoot = root) {
	const overrides = loadRunnerOverrides(workspaceRoot);
	const configured = resolveConfiguredPath(workspaceRoot, overrides.paths?.browserWorkspace, overrides.configPath);
	if (configured) {
		if (!fs.existsSync(configured) || !fs.statSync(configured).isDirectory()) {
			throw new Error(`configured browserWorkspace path does not exist: ${configured}`);
		}
		if (!hasPlaywrightGoSuite(configured)) {
			throw new Error(`configured browserWorkspace does not contain a Playwright-Go suite: ${configured}`);
		}
		return configured;
	}
	if (hasPlaywrightGoSuite(workspaceRoot)) {
		return workspaceRoot;
	}
	const repoWorkspace = path.join(root, 'test');
	if (hasPlaywrightGoSuite(repoWorkspace)) {
		return repoWorkspace;
	}
	return '';
}

function hasPlaywrightGoSuite(workspace) {
	return fs.existsSync(path.join(workspace, 'playwrightgo')) ||
		fs.existsSync(path.join(workspace, 'test', 'playwrightgo'));
}

export function resolveLivereloadWorkspace(root, workspaceRoot = root) {
	const overrides = loadRunnerOverrides(workspaceRoot);
	const configured = resolveConfiguredPath(workspaceRoot, overrides.paths?.livereloadWorkspace, overrides.configPath);
	if (configured) {
		if (!fs.existsSync(configured) || !fs.statSync(configured).isDirectory()) {
			throw new Error(`configured livereloadWorkspace path does not exist: ${configured}`);
		}
		return configured;
	}
	return path.join(root, 'tools', 'livereload');
}

export function resolveLivereloadClientScript(root, workspaceRoot = root) {
	const overrides = loadRunnerOverrides(workspaceRoot);
	const configured = resolveConfiguredPath(workspaceRoot, overrides.paths?.livereloadClientScript, overrides.configPath);
	if (configured) {
		if (!fs.existsSync(configured)) {
			throw new Error(`configured livereloadClientScript path does not exist: ${configured}`);
		}
		return configured;
	}
	return '';
}
