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

func (f *filesListFlag) String() string {
	return strings.Join(f.values, ",")
}

func (f *filesListFlag) Set(value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	f.values = append(f.values, trimmed)
	return nil
}

func (l launcher) runFiles(args []string) error {
	fs := flag.NewFlagSet("files", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	root := fs.String("root", "", "Root directory to inspect; defaults to the current working directory")
	jsonOutput := fs.Bool("json", false, "Emit a machine-readable JSON report")
	var extensions filesListFlag
	var excludeDirs filesListFlag
	fs.Var(&extensions, "ext", "File extension filter, with or without a leading dot; repeatable")
	fs.Var(&excludeDirs, "exclude-dir", "Directory name to skip anywhere in the walk; repeatable (.git is always skipped)")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	config, err := resolveFilesConfig(filesConfig{
		rootPath:    *root,
		extensions:  extensions.values,
		excludeDirs: excludeDirs.values,
		json:        *jsonOutput,
	})
	if err != nil {
		return err
	}
	report, err := collectFilesReport(config)
	if err != nil {
		return err
	}
	if config.json {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(report)
	}
	for _, path := range report.Files {
		fmt.Println(path)
	}
	return nil
}

func resolveFilesConfig(config filesConfig) (filesConfig, error) {
	rootPath := strings.TrimSpace(config.rootPath)
	if rootPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return filesConfig{}, fmt.Errorf("resolve files root from cwd: %w", err)
		}
		rootPath = cwd
	}
	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		return filesConfig{}, fmt.Errorf("resolve files root: %w", err)
	}
	info, err := os.Stat(absRoot)
	if err != nil {
		return filesConfig{}, fmt.Errorf("stat files root: %w", err)
	}
	if !info.IsDir() {
		return filesConfig{}, fmt.Errorf("files root is not a directory: %s", absRoot)
	}

	extensions, err := normalizeFilesExtensions(config.extensions)
	if err != nil {
		return filesConfig{}, err
	}
	excludeDirs, err := normalizeFilesExcludeDirs(config.excludeDirs)
	if err != nil {
		return filesConfig{}, err
	}

	return filesConfig{
		rootPath:    absRoot,
		extensions:  extensions,
		excludeDirs: excludeDirs,
		json:        config.json,
	}, nil
}

func normalizeFilesExtensions(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	normalized := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if strings.ContainsAny(trimmed, `/\`) {
			return nil, fmt.Errorf("extension filters must be plain file extensions, got %q", value)
		}
		if !strings.HasPrefix(trimmed, ".") {
			trimmed = "." + trimmed
		}
		if trimmed == "." {
			return nil, fmt.Errorf("extension filters must not be empty")
		}
		normalized[strings.ToLower(trimmed)] = struct{}{}
	}
	if len(normalized) == 0 {
		return nil, nil
	}
	result := make([]string, 0, len(normalized))
	for value := range normalized {
		result = append(result, value)
	}
	sort.Strings(result)
	return result, nil
}

func normalizeFilesExcludeDirs(values []string) ([]string, error) {
	normalized := map[string]struct{}{
		".git": {},
	}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if strings.ContainsAny(trimmed, `/\`) {
			return nil, fmt.Errorf("exclude-dir values must be single directory names, got %q", value)
		}
		cleaned := strings.ToLower(filepath.Clean(trimmed))
		switch cleaned {
		case "", ".", "..":
			return nil, fmt.Errorf("exclude-dir values must be concrete directory names, got %q", value)
		}
		normalized[cleaned] = struct{}{}
	}
	result := make([]string, 0, len(normalized))
	for value := range normalized {
		result = append(result, value)
	}
	sort.Strings(result)
	return result, nil
}

func collectFilesReport(config filesConfig) (filesReport, error) {
	excludeDirs := make(map[string]struct{}, len(config.excludeDirs))
	for _, dirName := range config.excludeDirs {
		excludeDirs[strings.ToLower(strings.TrimSpace(dirName))] = struct{}{}
	}
	extensions := make(map[string]struct{}, len(config.extensions))
	for _, extension := range config.extensions {
		extensions[strings.ToLower(strings.TrimSpace(extension))] = struct{}{}
	}

	files := []string{}
	err := filepath.WalkDir(config.rootPath, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == config.rootPath {
			return nil
		}
		if entry.IsDir() {
			if _, skip := excludeDirs[strings.ToLower(entry.Name())]; skip {
				return filepath.SkipDir
			}
			return nil
		}
		if len(extensions) > 0 {
			if _, ok := extensions[strings.ToLower(filepath.Ext(entry.Name()))]; !ok {
				return nil
			}
		}
		relPath, err := filepath.Rel(config.rootPath, path)
		if err != nil {
			return fmt.Errorf("resolve relative path for %s: %w", path, err)
		}
		files = append(files, filepath.ToSlash(relPath))
		return nil
	})
	if err != nil {
		return filesReport{}, fmt.Errorf("walk files under %s: %w", config.rootPath, err)
	}
	sort.Strings(files)
	return filesReport{
		OK:          true,
		Root:        config.rootPath,
		Extensions:  append([]string(nil), config.extensions...),
		ExcludeDirs: append([]string(nil), config.excludeDirs...),
		Count:       len(files),
		Files:       files,
	}, nil
}
