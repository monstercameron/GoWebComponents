package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/monstercameron/GoWebComponents/tools/runnerconfig"
)

const launcherOverrideEnvVar = runnerconfig.OverrideEnvVar

type launcherOverridePaths = runnerconfig.Paths

type launcherOverrides = runnerconfig.Overrides

var launcherConfigGetwd = os.Getwd

var launcherConfigUserHomeDir = os.UserHomeDir

func launcherRunnerConfigFS() runnerconfig.FS {
	return runnerconfig.FS{
		Getwd:       launcherConfigGetwd,
		UserHomeDir: launcherConfigUserHomeDir,
		ReadFile:    os.ReadFile,
		Stat:        os.Stat,
	}
}

func loadLauncherOverrides(cwd string) (launcherOverrides, string, bool, error) {
	return runnerconfig.Load(cwd, launcherRunnerConfigFS())
}

func loadLauncherOverridesForCurrentContext() (launcherOverrides, string, bool, error) {
	cwd, err := launcherConfigGetwd()
	if err != nil {
		cwd = ""
	}
	return loadLauncherOverrides(cwd)
}

func resolveLauncherOverridePath(cwd string) (string, error) {
	return runnerconfig.ResolveConfigPath(cwd, launcherRunnerConfigFS())
}

func findLauncherOverrideInParents(cwd string) string {
	return runnerconfig.FindConfigInParents(cwd, launcherRunnerConfigFS())
}

func resolveLauncherOverrideValue(configPath string, raw string) (string, error) {
	return runnerconfig.ResolveValue(configPath, raw)
}

func resolveLauncherConfiguredPath(rootPath string, selector func(launcherOverridePaths) string, label string) (string, bool, error) {
	return runnerconfig.ResolveConfiguredPath(rootPath, func(paths runnerconfig.Paths) string {
		return selector(paths)
	}, label, launcherRunnerConfigFS())
}

func resolveLauncherArtifactRoot(rootPath string) (string, bool, error) {
	return runnerconfig.ResolveArtifactRoot(rootPath, launcherRunnerConfigFS())
}

func launcherArtifactNamespace(rootPath string) string {
	return runnerconfig.ArtifactNamespace(rootPath)
}

func resolveLauncherArtifactPath(rootPath string, segments ...string) (string, bool, error) {
	return runnerconfig.ResolveArtifactPath(rootPath, launcherRunnerConfigFS(), segments...)
}

func resolveLauncherWorkspaceBuildPath(rootPath string, segments ...string) (string, error) {
	return runnerconfig.ResolveWorkspaceBuildPath(rootPath, launcherRunnerConfigFS(), segments...)
}

func resolveLauncherExamplesWasmDir(repoRoot string, staticDir string) string {
	if strings.TrimSpace(repoRoot) != "" {
		resolved, err := resolveLauncherWorkspaceBuildPath(repoRoot, "examples")
		if err == nil && strings.TrimSpace(resolved) != "" {
			if info, statErr := os.Stat(resolved); statErr == nil && info.IsDir() {
				return resolved
			}
		}
	}
	if strings.TrimSpace(staticDir) != "" {
		return filepath.Join(staticDir, "bin")
	}
	return ""
}

func resolveLauncherDefaultBuildOutput(rootPath string) (string, string, error) {
	if artifactPath, ok, err := resolveLauncherArtifactPath(rootPath, scaffoldWASMOutputPath()); err != nil {
		return "", "", err
	} else if ok {
		return artifactPath, "gwc-runner.json paths.artifactRoot", nil
	}
	return filepath.Join(rootPath, scaffoldWASMOutputPath()), "convention fallback", nil
}

func resolveLauncherDefaultReleaseOutDir(rootPath string) (string, string, error) {
	if artifactPath, ok, err := resolveLauncherArtifactPath(rootPath, "wasm-release"); err != nil {
		return "", "", err
	} else if ok {
		return artifactPath, "gwc-runner.json paths.artifactRoot", nil
	}
	return filepath.Join(rootPath, defaultScaffoldReleaseOutDir()), "convention fallback", nil
}

func resolveLauncherTempRoot(rootPath string) (string, string, error) {
	if artifactPath, ok, err := resolveLauncherArtifactPath(rootPath, "tmp"); err != nil {
		return "", "", err
	} else if ok {
		return artifactPath, "gwc-runner.json paths.artifactRoot", nil
	}
	return filepath.Join(rootPath, "bin", "tmp"), "convention fallback", nil
}
