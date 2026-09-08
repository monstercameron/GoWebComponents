package main

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
)

// validateScaffoldDesktopMetadata validates an opt-in desktop target without
// requiring generated files to exist yet.
func validateScaffoldDesktopMetadata(parseMetadata *scaffoldDesktopMetadata) error {
	if parseMetadata == nil {
		return nil
	}
	if parseMetadata.Version != 1 {
		return fmt.Errorf("unsupported desktop metadata version %d", parseMetadata.Version)
	}
	if strings.TrimSpace(parseMetadata.WailsVersion) == "" {
		return fmt.Errorf("desktop wailsVersion is required")
	}
	parsePaths := []struct {
		name string
		path string
	}{
		{name: "nativeEntry", path: parseMetadata.NativeEntry},
		{name: "frontendEntry", path: parseMetadata.FrontendEntry},
		{name: "assetsDir", path: parseMetadata.AssetsDir},
		{name: "outputPath", path: parseMetadata.OutputPath},
	}
	for _, parseItem := range parsePaths {
		if parseErr := validateDesktopRelativePath(parseItem.name, parseItem.path); parseErr != nil {
			return parseErr
		}
	}
	parseInputPaths := []string{parseMetadata.NativeEntry, parseMetadata.FrontendEntry, parseMetadata.AssetsDir}
	for _, parseInputPath := range parseInputPaths {
		if desktopPathsOverlap(parseMetadata.OutputPath, parseInputPath) {
			return fmt.Errorf("desktop outputPath %q overlaps input path %q", parseMetadata.OutputPath, parseInputPath)
		}
	}
	return nil
}

// validateDesktopRelativePath rejects traversal and Windows volume/stream aliases on every platform.
func validateDesktopRelativePath(parseName string, parsePath string) error {
	if parsePath != strings.TrimSpace(parsePath) || strings.ContainsAny(parsePath, ":\x00") {
		return fmt.Errorf("desktop %s has an invalid path: %q", parseName, parsePath)
	}
	parsePath = strings.TrimSpace(parsePath)
	if parsePath == "" {
		return fmt.Errorf("desktop %s is required", parseName)
	}
	if filepath.IsAbs(parsePath) || filepath.VolumeName(parsePath) != "" || strings.HasPrefix(parsePath, "/") || strings.HasPrefix(parsePath, "\\") {
		return fmt.Errorf("desktop %s must be relative to the module root: %q", parseName, parsePath)
	}
	parseParts := strings.FieldsFunc(parsePath, func(parseRune rune) bool { return parseRune == '/' || parseRune == '\\' })
	for _, parsePart := range parseParts {
		if strings.TrimRight(parsePart, " .") != parsePart && parsePart != "." {
			return fmt.Errorf("desktop %s contains an ambiguous Windows path component: %q", parseName, parsePart)
		}
		if parsePart == ".." {
			return fmt.Errorf("desktop %s must not contain parent traversal: %q", parseName, parsePath)
		}
	}
	if filepath.Clean(parsePath) == "." {
		return fmt.Errorf("desktop %s must name a path: %q", parseName, parsePath)
	}
	return nil
}

// desktopPathsOverlap compares slash-normalized paths using Windows case folding on all hosts.
func desktopPathsOverlap(parseLeft string, parseRight string) bool {
	parseLeft = strings.ToLower(path.Clean(strings.ReplaceAll(parseLeft, "\\", "/")))
	parseRight = strings.ToLower(path.Clean(strings.ReplaceAll(parseRight, "\\", "/")))
	return desktopPathWithin(parseLeft, parseRight) || desktopPathWithin(parseRight, parseLeft)
}

// desktopPathWithin reports equality or a component-boundary descendant.
func desktopPathWithin(parseChild string, parseParent string) bool {
	parseRelative, parseErr := filepath.Rel(parseParent, parseChild)
	if parseErr != nil {
		return false
	}
	return parseRelative == "." || (parseRelative != ".." && !strings.HasPrefix(parseRelative, ".."+string(filepath.Separator)))
}
