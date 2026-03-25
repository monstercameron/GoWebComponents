package sqlfiles

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

var (
	rootOnce sync.Once
	roots    []string
	rootErr  error
	cache    sync.Map
)

// Load reads one SQL file from the example's sql directory and caches the result.
func Load(relativePath string) (string, error) {
	normalizedPath := filepath.ToSlash(strings.TrimSpace(relativePath))
	if normalizedPath == "" {
		return "", errors.New("sql path is empty")
	}
	if cachedValue, ok := cache.Load(normalizedPath); ok {
		return cachedValue.(string), nil
	}
	candidateRoots, err := exampleRoots()
	if err != nil {
		return "", err
	}
	triedPaths := make([]string, 0, len(candidateRoots))
	for _, root := range candidateRoots {
		sqlPath := filepath.Join(root, "sql", filepath.FromSlash(normalizedPath))
		triedPaths = append(triedPaths, sqlPath)
		contents, readErr := os.ReadFile(sqlPath)
		if readErr == nil {
			sqlText := string(contents)
			cache.Store(normalizedPath, sqlText)
			return sqlText, nil
		}
		if !errors.Is(readErr, os.ErrNotExist) {
			return "", readErr
		}
	}
	return "", fmt.Errorf("sql file not found: %s (tried %s)", normalizedPath, strings.Join(triedPaths, ", "))
}

func exampleRoots() ([]string, error) {
	rootOnce.Do(func() {
		seen := map[string]struct{}{}
		addRoot := func(path string) {
			if strings.TrimSpace(path) == "" {
				return
			}
			cleanPath := filepath.Clean(path)
			if _, exists := seen[cleanPath]; exists {
				return
			}
			seen[cleanPath] = struct{}{}
			roots = append(roots, cleanPath)
		}

		if configuredRoot := os.Getenv("CHAT_WIZARD_ROOT"); configuredRoot != "" {
			addRoot(configuredRoot)
		}
		if workingDir, err := os.Getwd(); err == nil {
			addRoot(workingDir)
			addRoot(filepath.Join(workingDir, "examples", "100-ai-chat-wizard"))
		}
		if executablePath, err := os.Executable(); err == nil {
			currentDir := filepath.Dir(executablePath)
			for i := 0; i < 5; i++ {
				addRoot(currentDir)
				addRoot(filepath.Join(currentDir, "examples", "100-ai-chat-wizard"))
				parentDir := filepath.Dir(currentDir)
				if parentDir == currentDir {
					break
				}
				currentDir = parentDir
			}
		}
		if _, sourceFile, _, ok := runtime.Caller(0); ok {
			addRoot(filepath.Join(filepath.Dir(sourceFile), "..", ".."))
		}
		if len(roots) == 0 {
			rootErr = errors.New("could not determine chat wizard sql root")
		}
	})
	return roots, rootErr
}
