package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// runExamplesBuildPublicSite builds the public examples site wasm binary and stages embedded example binaries for local example browsing.
func (parseL launcher) runExamplesBuildPublicSite(parseArgs []string) error {
	buildFs := flag.NewFlagSet("examples build-public-site", flag.ContinueOnError)
	buildFs.SetOutput(os.Stdout)
	buildProfile := buildFs.String("profile", "release", "Build profile: development, ci, benchmark, release, or tinygo")
	if buildErr := buildFs.Parse(parseArgs); buildErr != nil {
		if errors.Is(buildErr, flag.ErrHelp) {
			return nil
		}
		return buildErr
	}

	buildConfigs, buildErr := buildExamplesPublicSiteConfigs(parseL, *buildProfile)
	if buildErr != nil {
		return buildErr
	}

	for _, buildConfig := range buildConfigs {
		buildSummary, buildErr := executeExamplesBuild(buildConfig)
		if buildErr != nil {
			return buildErr
		}
		printBuildSummary(buildSummary)
	}

	buildStagedExampleCount, buildErr := stageExamplesPublicSiteBinaries(parseL, buildConfigs)
	if buildErr != nil {
		return buildErr
	}
	if buildErr := stageExamplesPublicSitePreviewRoots(parseL, buildConfigs); buildErr != nil {
		return buildErr
	}
	if copyErr := copyExamplesEnsureLoggerScript(filepath.Join(parseL.staticDir, "script", "example-logger.js")); copyErr != nil {
		return copyErr
	}
	if copyErr := syncExamplesPublicCodeMirrors(parseL); copyErr != nil {
		return copyErr
	}
	if copyErr := syncExamplesPublicSiteCatalog(parseL); copyErr != nil {
		return copyErr
	}
	fmt.Printf("Staged public site wasm to %s\n", filepath.Join(parseL.staticDir, "bin", "public-examples-site.wasm"))
	fmt.Printf("Staged %d public example wasm binaries to %s\n", buildStagedExampleCount, filepath.Join(parseL.examplesDir, "public-examples-site", "assets", "bins"))
	fmt.Printf("Staged runnable public example previews to %s\n", filepath.Join(parseL.examplesDir, "public-examples-site", "assets", "examples"))
	fmt.Printf("Mirrored public example source files to %s\n", filepath.Join(parseL.examplesDir, "public-examples-site", "assets", "code"))
	fmt.Printf("Regenerated public examples catalog at %s\n", filepath.Join(parseL.examplesDir, "public-examples-site", "assets", "data", "catalog.json"))
	return nil
}

// buildExamplesPublicSiteConfigs resolves the launcher-owned build configs for the public examples site shell and every top-level public example binary.
func buildExamplesPublicSiteConfigs(parseL launcher, buildProfile string) ([]buildConfig, error) {
	buildWasmDir := strings.TrimSpace(parseL.resolvedExamplesWasmDir())
	if buildWasmDir == "" {
		return nil, errors.New("examples wasm directory is required")
	}

	buildRawConfigs := []buildConfig{
		{
			appPath:    filepath.Join(parseL.examplesDir, "public-examples-site", "main.go"),
			rootPath:   filepath.Join(parseL.examplesDir, "public-examples-site"),
			outputPath: filepath.Join(buildWasmDir, "public-examples-site.wasm"),
			profile:    buildProfile,
		},
	}
	buildExampleDirectories, buildErr := listExamplesPublicSiteDirectories(filepath.Join(parseL.examplesDir, "public"))
	if buildErr != nil {
		return nil, buildErr
	}
	for _, buildExampleDirectory := range buildExampleDirectories {
		buildExampleSlug := filepath.Base(buildExampleDirectory)
		buildRawConfigs = append(buildRawConfigs, buildConfig{
			appPath:    filepath.Join(buildExampleDirectory, "main.go"),
			rootPath:   buildExampleDirectory,
			outputPath: filepath.Join(buildWasmDir, buildExampleSlug+".wasm"),
			profile:    buildProfile,
		})
	}

	buildResolvedConfigs := make([]buildConfig, 0, len(buildRawConfigs))
	for _, buildRawConfig := range buildRawConfigs {
		buildResolvedConfig, buildErr := resolveBuildConfig(buildRawConfig)
		if buildErr != nil {
			return nil, buildErr
		}
		buildResolvedConfigs = append(buildResolvedConfigs, buildResolvedConfig)
	}
	return buildResolvedConfigs, nil
}

// stageExamplesPublicSiteBinaries mirrors the built shell and example wasm files into the static and embedded public-site asset directories.
func stageExamplesPublicSiteBinaries(parseL launcher, buildConfigs []buildConfig) (int, error) {
	buildExampleBinsDir := filepath.Join(parseL.examplesDir, "public-examples-site", "assets", "bins")
	if buildErr := os.RemoveAll(buildExampleBinsDir); buildErr != nil {
		return 0, fmt.Errorf("clear embedded example wasm directory: %w", buildErr)
	}
	if buildErr := os.MkdirAll(buildExampleBinsDir, 0755); buildErr != nil {
		return 0, fmt.Errorf("create embedded example wasm directory: %w", buildErr)
	}

	buildExampleCount := 0
	for _, buildConfig := range buildConfigs {
		buildBinaryName := filepath.Base(buildConfig.outputPath)
		if buildErr := copyExamplesBuildBinary(buildConfig.outputPath, filepath.Join(parseL.staticDir, "bin", buildBinaryName)); buildErr != nil {
			return 0, buildErr
		}
		if buildBinaryName == "public-examples-site.wasm" {
			continue
		}
		if buildErr := copyExamplesBuildBinary(buildConfig.outputPath, filepath.Join(buildExampleBinsDir, buildBinaryName)); buildErr != nil {
			return 0, buildErr
		}
		buildExampleCount++
	}
	return buildExampleCount, nil
}

// copyExamplesBuildBinary copies one built wasm artifact into the embedded public examples site asset directory.
func copyExamplesBuildBinary(copySourcePath string, copyTargetPath string) error {
	copyInputFile, copyErr := os.Open(copySourcePath)
	if copyErr != nil {
		return fmt.Errorf("open built wasm binary: %w", copyErr)
	}
	defer copyInputFile.Close()

	if copyErr := os.MkdirAll(filepath.Dir(copyTargetPath), 0755); copyErr != nil {
		return fmt.Errorf("create embedded wasm directory: %w", copyErr)
	}

	copyOutputFile, copyErr := os.Create(copyTargetPath)
	if copyErr != nil {
		return fmt.Errorf("create embedded wasm binary: %w", copyErr)
	}

	if _, copyErr := io.Copy(copyOutputFile, copyInputFile); copyErr != nil {
		_ = copyOutputFile.Close()
		return fmt.Errorf("copy embedded wasm binary: %w", copyErr)
	}
	if copyErr := copyOutputFile.Close(); copyErr != nil {
		return fmt.Errorf("flush embedded wasm binary: %w", copyErr)
	}
	return nil
}

// copyExamplesEnsureLoggerScript writes the small static logger helper expected by direct-file example previews.
func copyExamplesEnsureLoggerScript(copyTargetPath string) error {
	const copyLoggerScript = "(function(){\n" +
		"  if (window.__gwcExampleLogger) return;\n" +
		"  const copyPrefix = '[gwc example]';\n" +
		"  window.__gwcExampleLogger = {\n" +
		"    log: (...copyArgs) => console.log(copyPrefix, ...copyArgs),\n" +
		"    info: (...copyArgs) => console.info(copyPrefix, ...copyArgs),\n" +
		"    warn: (...copyArgs) => console.warn(copyPrefix, ...copyArgs),\n" +
		"    error: (...copyArgs) => console.error(copyPrefix, ...copyArgs)\n" +
		"  };\n" +
		"})();\n"

	if copyErr := os.MkdirAll(filepath.Dir(copyTargetPath), 0755); copyErr != nil {
		return fmt.Errorf("create example logger directory: %w", copyErr)
	}
	if copyErr := os.WriteFile(copyTargetPath, []byte(copyLoggerScript), 0644); copyErr != nil {
		return fmt.Errorf("write example logger script: %w", copyErr)
	}
	return nil
}

// syncExamplesPublicCodeMirrors refreshes the public examples source mirror used by the public examples site code viewer.
func syncExamplesPublicCodeMirrors(parseL launcher) error {
	copySourceRoot := filepath.Join(parseL.examplesDir, "public")
	copyTargetRoot := filepath.Join(parseL.examplesDir, "public-examples-site", "assets", "code")

	copyInfo, copyErr := os.Stat(copySourceRoot)
	if copyErr != nil {
		return fmt.Errorf("inspect public examples source directory: %w", copyErr)
	}
	if !copyInfo.IsDir() {
		return fmt.Errorf("public examples source path is not a directory: %s", copySourceRoot)
	}

	if copyErr := os.RemoveAll(copyTargetRoot); copyErr != nil {
		return fmt.Errorf("clear public examples code mirror: %w", copyErr)
	}
	if copyErr := copyExamplesDirectoryTree(copySourceRoot, copyTargetRoot); copyErr != nil {
		return copyErr
	}
	return nil
}

// copyExamplesDirectoryTree recursively copies one directory tree into another path.
func copyExamplesDirectoryTree(copySourceRoot string, copyTargetRoot string) error {
	return filepath.WalkDir(copySourceRoot, func(copySourcePath string, copyDirEntry os.DirEntry, copyErr error) error {
		if copyErr != nil {
			return copyErr
		}

		copyRelativePath, copyErr := filepath.Rel(copySourceRoot, copySourcePath)
		if copyErr != nil {
			return fmt.Errorf("resolve mirrored relative path: %w", copyErr)
		}
		copyTargetPath := filepath.Join(copyTargetRoot, copyRelativePath)

		if copyDirEntry.IsDir() {
			if copyErr := os.MkdirAll(copyTargetPath, 0755); copyErr != nil {
				return fmt.Errorf("create mirrored code directory: %w", copyErr)
			}
			return nil
		}

		return copyExamplesFile(copySourcePath, copyTargetPath)
	})
}

// copyExamplesFile copies one source file into the mirrored public examples code tree.
func copyExamplesFile(copySourcePath string, copyTargetPath string) error {
	copyInputFile, copyErr := os.Open(copySourcePath)
	if copyErr != nil {
		return fmt.Errorf("open source code mirror file: %w", copyErr)
	}
	defer copyInputFile.Close()

	if copyErr := os.MkdirAll(filepath.Dir(copyTargetPath), 0755); copyErr != nil {
		return fmt.Errorf("create source code mirror directory: %w", copyErr)
	}

	copyOutputFile, copyErr := os.Create(copyTargetPath)
	if copyErr != nil {
		return fmt.Errorf("create source code mirror file: %w", copyErr)
	}

	if _, copyErr := io.Copy(copyOutputFile, copyInputFile); copyErr != nil {
		_ = copyOutputFile.Close()
		return fmt.Errorf("copy source code mirror file: %w", copyErr)
	}
	if copyErr := copyOutputFile.Close(); copyErr != nil {
		return fmt.Errorf("flush source code mirror file: %w", copyErr)
	}
	return nil
}
