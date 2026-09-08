package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/monstercameron/GoWebComponents/v6/tools/runnerconfig"
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

func loadLauncherOverrides(parseCwd string) (launcherOverrides, string, bool, error) {
	return runnerconfig.Load(parseCwd, launcherRunnerConfigFS())
}

func loadLauncherOverridesForCurrentContext() (launcherOverrides, string, bool, error) {
	parseCwd, parseErr := launcherConfigGetwd()
	if parseErr != nil {
		parseCwd = ""
	}
	return loadLauncherOverrides(parseCwd)
}

func resolveLauncherOverridePath(parseCwd string) (string, error) {
	return runnerconfig.ResolveConfigPath(parseCwd, launcherRunnerConfigFS())
}

func findLauncherOverrideInParents(parseCwd string) string {
	return runnerconfig.LocateConfigInParents(parseCwd, launcherRunnerConfigFS())
}

func resolveLauncherOverrideValue(parseConfigPath string, parseRaw string) (string, error) {
	return runnerconfig.ResolveValue(parseConfigPath, parseRaw)
}

func resolveLauncherConfiguredPath(parseRootPath string, parseSelector func(launcherOverridePaths) string, parseLabel string) (string, bool, error) {
	return runnerconfig.ResolveConfiguredPath(parseRootPath, func(parsePaths runnerconfig.Paths) string {
		return parseSelector(parsePaths)
	}, parseLabel, launcherRunnerConfigFS())
}

func resolveLauncherArtifactRoot(parseRootPath string) (string, bool, error) {
	return runnerconfig.ResolveArtifactRoot(parseRootPath, launcherRunnerConfigFS())
}

func resolveLauncherArtifactPath(parseRootPath string, parseSegments ...string) (string, bool, error) {
	return runnerconfig.ResolveArtifactPath(parseRootPath, launcherRunnerConfigFS(), parseSegments...)
}

func resolveLauncherWorkspaceBuildPath(parseRootPath string, parseSegments ...string) (string, error) {
	return runnerconfig.ResolveWorkspaceBuildPath(parseRootPath, launcherRunnerConfigFS(), parseSegments...)
}

func resolveLauncherExamplesWasmDir(parseRepoRoot string, parseStaticDir string) string {
	if strings.TrimSpace(parseRepoRoot) != "" {
		parseResolved, parseErr := resolveLauncherWorkspaceBuildPath(parseRepoRoot, "examples")
		if parseErr == nil && strings.TrimSpace(parseResolved) != "" {
			if parseInfo, parseStatErr := os.Stat(parseResolved); parseStatErr == nil && parseInfo.IsDir() {
				return parseResolved
			}
		}
	}
	if strings.TrimSpace(parseStaticDir) != "" {
		return filepath.Join(parseStaticDir, "bin")
	}
	return ""
}

func resolveLauncherDefaultBuildOutput(parseRootPath string) (string, string, error) {
	if parseArtifactPath, parseOk, parseErr := resolveLauncherArtifactPath(parseRootPath, scaffoldWASMOutputPath()); parseErr != nil {
		return "", "", parseErr
	} else if parseOk {
		return parseArtifactPath, "gwc-runner.json paths.artifactRoot", nil
	}
	return filepath.Join(parseRootPath, scaffoldWASMOutputPath()), "convention fallback", nil
}

func resolveLauncherDefaultReleaseOutDir(parseRootPath string) (string, string, error) {
	if parseArtifactPath, parseOk, parseErr := resolveLauncherArtifactPath(parseRootPath, "wasm-release"); parseErr != nil {
		return "", "", parseErr
	} else if parseOk {
		return parseArtifactPath, "gwc-runner.json paths.artifactRoot", nil
	}
	return filepath.Join(parseRootPath, defaultScaffoldReleaseOutDir()), "convention fallback", nil
}

func resolveLauncherTempRoot(parseRootPath string) (string, string, error) {
	if parseArtifactPath, parseOk, parseErr := resolveLauncherArtifactPath(parseRootPath, "tmp"); parseErr != nil {
		return "", "", parseErr
	} else if parseOk {
		return parseArtifactPath, "gwc-runner.json paths.artifactRoot", nil
	}
	return filepath.Join(parseRootPath, "bin", "tmp"), "convention fallback", nil
}
