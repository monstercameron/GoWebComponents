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