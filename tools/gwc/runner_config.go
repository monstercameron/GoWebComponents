package main

import (
	"os"

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
