package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var runFilesCommand = func(l launcher, args []string) error {
	return l.runFiles(args)
}

type filesConfig struct {
	rootPath    string
	extensions  []string
	excludeDirs []string
	json        bool
}

type filesReport struct {
	OK          bool     `json:"ok"`
	Root        string   `json:"root"`
	Extensions  []string `json:"extensions,omitempty"`
	ExcludeDirs []string `json:"excludeDirs,omitempty"`
	Count       int      `json:"count"`
	Files       []string `json:"files"`
}

type filesListFlag struct {
	values []string
}

func (parseF *filesListFlag) String() string {
	return strings.Join(parseF.values, ",")
}

func (parseF *filesListFlag) Set(parseValue string) error {
	parseTrimmed := strings.TrimSpace(parseValue)
	if parseTrimmed == "" {
		return nil
	}
	parseF.values = append(parseF.values, parseTrimmed)
	return nil
}

func (parseL launcher) runFiles(parseArgs []string) error {
	parseFs := flag.NewFlagSet("files", flag.ContinueOnError)
	parseFs.SetOutput(os.Stdout)
	parseRoot := parseFs.String("root", "", "Root directory to inspect; defaults to the current working directory")
	parseJsonOutput := parseFs.Bool("json", false, "Emit a machine-readable JSON report")
	var parseExtensions filesListFlag
	var parseExcludeDirs filesListFlag
	parseFs.Var(&parseExtensions, "ext", "File extension filter, with or without a leading dot; repeatable")
	parseFs.Var(&parseExcludeDirs, "exclude-dir", "Directory name to skip anywhere in the walk; repeatable (.git is always skipped)")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}

	parseConfig, parseErr2 := resolveFilesConfig(filesConfig{
		rootPath:    *parseRoot,
		extensions:  parseExtensions.values,
		excludeDirs: parseExcludeDirs.values,
		json:        *parseJsonOutput,
	})
	if parseErr2 != nil {
		return parseErr2
	}
	parseReport, parseErr2 := collectFilesReport(parseConfig)
	if parseErr2 != nil {
		return parseErr2
	}
	if parseConfig.json {
		parseEncoder := json.NewEncoder(os.Stdout)
		parseEncoder.SetIndent("", "  ")
		return parseEncoder.Encode(parseReport)
	}
	for _, parsePath := range parseReport.Files {
		fmt.Println(parsePath)
	}
	return nil
}

func resolveFilesConfig(parseConfig filesConfig) (filesConfig, error) {
	parseRootPath := strings.TrimSpace(parseConfig.rootPath)
	if parseRootPath == "" {
		parseCwd, parseErr := os.Getwd()
		if parseErr != nil {
			return filesConfig{}, fmt.Errorf("resolve files root from cwd: %w", parseErr)
		}
		parseRootPath = parseCwd
	}
	parseAbsRoot, parseErr2 := filepath.Abs(parseRootPath)
	if parseErr2 != nil {
		return filesConfig{}, fmt.Errorf("resolve files root: %w", parseErr2)
	}
	parseInfo, parseErr2 := os.Stat(parseAbsRoot)
	if parseErr2 != nil {
		return filesConfig{}, fmt.Errorf("stat files root: %w", parseErr2)
	}
	if !parseInfo.IsDir() {
		return filesConfig{}, fmt.Errorf("files root is not a directory: %s", parseAbsRoot)
	}

	parseExtensions, parseErr2 := normalizeFilesExtensions(parseConfig.extensions)
	if parseErr2 != nil {
		return filesConfig{}, parseErr2
	}
	parseExcludeDirs, parseErr2 := normalizeFilesExcludeDirs(parseConfig.excludeDirs)
	if parseErr2 != nil {
		return filesConfig{}, parseErr2
	}

	return filesConfig{
		rootPath:    parseAbsRoot,
		extensions:  parseExtensions,
		excludeDirs: parseExcludeDirs,
		json:        parseConfig.json,
	}, nil
}

func normalizeFilesExtensions(parseValues []string) ([]string, error) {
	if len(parseValues) == 0 {
		return nil, nil
	}
	parseNormalized := map[string]struct{}{}
	for _, parseValue := range parseValues {
		parseTrimmed := strings.TrimSpace(parseValue)
		if parseTrimmed == "" {
			continue
		}
		if strings.ContainsAny(parseTrimmed, `/\`) {
			return nil, fmt.Errorf("extension filters must be plain file extensions, got %q", parseValue)
		}
		if !strings.HasPrefix(parseTrimmed, ".") {
			parseTrimmed = "." + parseTrimmed
		}
		if parseTrimmed == "." {
			return nil, fmt.Errorf("extension filters must not be empty")
		}
		parseNormalized[strings.ToLower(parseTrimmed)] = struct{}{}
	}
	if len(parseNormalized) == 0 {
		return nil, nil
	}
	parseResult := make([]string, 0, len(parseNormalized))
	for parseValue2 := range parseNormalized {
		parseResult = append(parseResult, parseValue2)
	}
	sort.Strings(parseResult)
	return parseResult, nil
}

func normalizeFilesExcludeDirs(parseValues []string) ([]string, error) {
	parseNormalized := map[string]struct{}{
		".git": {},
	}
	for _, parseValue := range parseValues {
		parseTrimmed := strings.TrimSpace(parseValue)
		if parseTrimmed == "" {
			continue
		}
		if strings.ContainsAny(parseTrimmed, `/\`) {
			return nil, fmt.Errorf("exclude-dir values must be single directory names, got %q", parseValue)
		}
		parseCleaned := strings.ToLower(filepath.Clean(parseTrimmed))
		switch parseCleaned {
		case "", ".", "..":
			return nil, fmt.Errorf("exclude-dir values must be concrete directory names, got %q", parseValue)
		}
		parseNormalized[parseCleaned] = struct{}{}
	}
	parseResult := make([]string, 0, len(parseNormalized))
	for parseValue2 := range parseNormalized {
		parseResult = append(parseResult, parseValue2)
	}
	sort.Strings(parseResult)
	return parseResult, nil
}

func collectFilesReport(parseConfig filesConfig) (filesReport, error) {
	parseExcludeDirs := make(map[string]struct{}, len(parseConfig.excludeDirs))
	for _, parseDirName := range parseConfig.excludeDirs {
		parseExcludeDirs[strings.ToLower(strings.TrimSpace(parseDirName))] = struct{}{}
	}
	parseExtensions := make(map[string]struct{}, len(parseConfig.extensions))
	for _, parseExtension := range parseConfig.extensions {
		parseExtensions[strings.ToLower(strings.TrimSpace(parseExtension))] = struct{}{}
	}

	parseFiles := []string{}
	parseErr := filepath.WalkDir(parseConfig.rootPath, func(parsePath string, parseEntry fs.DirEntry, parseErr2 error) error {
		if parseErr2 != nil {
			return parseErr2
		}
		if parsePath == parseConfig.rootPath {
			return nil
		}
		if parseEntry.IsDir() {
			if _, parseSkip := parseExcludeDirs[strings.ToLower(parseEntry.Name())]; parseSkip {
				return filepath.SkipDir
			}
			return nil
		}
		if len(parseExtensions) > 0 {
			if _, parseOk := parseExtensions[strings.ToLower(filepath.Ext(parseEntry.Name()))]; !parseOk {
				return nil
			}
		}
		parseRelPath, parseErr2 := filepath.Rel(parseConfig.rootPath, parsePath)
		if parseErr2 != nil {
			return fmt.Errorf("resolve relative path for %s: %w", parsePath, parseErr2)
		}
		parseFiles = append(parseFiles, filepath.ToSlash(parseRelPath))
		return nil
	})
	if parseErr != nil {
		return filesReport{}, fmt.Errorf("walk files under %s: %w", parseConfig.rootPath, parseErr)
	}
	sort.Strings(parseFiles)
	return filesReport{
		OK:          true,
		Root:        parseConfig.rootPath,
		Extensions:  append([]string(nil), parseConfig.extensions...),
		ExcludeDirs: append([]string(nil), parseConfig.excludeDirs...),
		Count:       len(parseFiles),
		Files:       parseFiles,
	}, nil
}
