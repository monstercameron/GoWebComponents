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
func ParseLoad(parseRelativePath string) (string, error) {
	parseNormalizedPath := filepath.ToSlash(strings.TrimSpace(parseRelativePath))
	if parseNormalizedPath == "" {
		return "", errors.New("sql path is empty")
	}
	if parseCachedValue, parseOk := cache.Load(parseNormalizedPath); parseOk {
		return parseCachedValue.(string), nil
	}
	parseCandidateRoots, parseErr := parseExampleRoots()
	if parseErr != nil {
		return "", parseErr
	}
	parseTriedPaths := make([]string, 0, len(parseCandidateRoots))
	for _, parseRoot := range parseCandidateRoots {
		parseSqlPath := filepath.Join(parseRoot, "sql", filepath.FromSlash(parseNormalizedPath))
		parseTriedPaths = append(parseTriedPaths, parseSqlPath)
		parseContents, parseReadErr := os.ReadFile(parseSqlPath)
		if parseReadErr == nil {
			parseSqlText := string(parseContents)
			cache.Store(parseNormalizedPath, parseSqlText)
			return parseSqlText, nil
		}
		if !errors.Is(parseReadErr, os.ErrNotExist) {
			return "", parseReadErr
		}
	}
	return "", fmt.Errorf("sql file not found: %s (tried %s)", parseNormalizedPath, strings.Join(parseTriedPaths, ", "))
}

func parseExampleRoots() ([]string, error) {
	rootOnce.Do(func() {
		parseSeen := map[string]struct{}{}
		parseAddRoot := func(parsePath string) {
			if strings.TrimSpace(parsePath) == "" {
				return
			}
			parseCleanPath := filepath.Clean(parsePath)
			if _, parseExists := parseSeen[parseCleanPath]; parseExists {
				return
			}
			parseSeen[parseCleanPath] = struct{}{}
			roots = append(roots, parseCleanPath)
		}

		if parseConfiguredRoot := os.Getenv("CHAT_WIZARD_ROOT"); parseConfiguredRoot != "" {
			parseAddRoot(parseConfiguredRoot)
		}
		if parseWorkingDir, parseErr := os.Getwd(); parseErr == nil {
			parseAddRoot(parseWorkingDir)
			parseAddRoot(filepath.Join(parseWorkingDir, "examples", "100-ai-chat-wizard"))
		}
		if parseExecutablePath, parseErr2 := os.Executable(); parseErr2 == nil {
			parseCurrentDir := filepath.Dir(parseExecutablePath)
			for range 5 {
				parseAddRoot(parseCurrentDir)
				parseAddRoot(filepath.Join(parseCurrentDir, "examples", "100-ai-chat-wizard"))
				parseParentDir := filepath.Dir(parseCurrentDir)
				if parseParentDir == parseCurrentDir {
					break
				}
				parseCurrentDir = parseParentDir
			}
		}
		if _, parseSourceFile, _, parseOk := runtime.Caller(0); parseOk {
			parseAddRoot(filepath.Join(filepath.Dir(parseSourceFile), "..", ".."))
		}
		if len(roots) == 0 {
			rootErr = errors.New("could not determine chat wizard sql root")
		}
	})
	return roots, rootErr
}
