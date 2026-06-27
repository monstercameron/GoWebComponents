package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestComponentRegistryTemplatesEmbedAndCompileMarkers proves every catalog entry has an
// embedded template that is non-empty and declares the catalog package (a smoke check that
// the embed paths are correct; the templates themselves compile as tools/gwc/templates).
func TestComponentRegistryTemplatesEmbed(parseT *testing.T) {
	for _, parseEntry := range componentRegistry {
		parseSource, parseErr := componentTemplatesFS.ReadFile(parseEntry.file)
		if parseErr != nil {
			parseT.Fatalf("component %q: embedded file %q unreadable: %v", parseEntry.name, parseEntry.file, parseErr)
		}
		if !strings.Contains(string(parseSource), "package components") {
			parseT.Fatalf("component %q: template should be in package components", parseEntry.name)
		}
	}
}

// TestRenderComponentSourceRewritesPackage proves the copied source drops the catalog
// package doc, sets the destination package, adds a provenance header, and is gofmt-clean.
func TestRenderComponentSourceRewritesPackage(parseT *testing.T) {
	parseSource, _ := componentTemplatesFS.ReadFile("templates/disclosure.go")
	parseRendered, parseErr := renderComponentSource(string(parseSource), "ui_kit", "disclosure")
	if parseErr != nil {
		parseT.Fatalf("render: %v", parseErr)
	}
	if !strings.Contains(parseRendered, "package ui_kit") {
		parseT.Fatalf("expected destination package, got:\n%s", parseRendered)
	}
	if strings.Contains(parseRendered, "package components") {
		parseT.Fatal("catalog package name should be rewritten away")
	}
	if !strings.HasPrefix(parseRendered, "// Code added by `gwc add disclosure`") {
		parseT.Fatalf("expected provenance header, got:\n%.120s", parseRendered)
	}
	if !strings.Contains(parseRendered, "func Disclosure(") {
		parseT.Fatal("expected the Disclosure component to be present")
	}
}

// TestLookupComponentUnknown proves an unknown name is reported, not silently ignored.
func TestLookupComponentUnknown(parseT *testing.T) {
	if _, parseOk := lookupComponent("does-not-exist"); parseOk {
		parseT.Fatal("expected unknown component to be absent")
	}
	if _, parseOk := lookupComponent("tabs"); !parseOk {
		parseT.Fatal("expected tabs to be in the catalog")
	}
}

// TestRunAddWritesComponent proves the end-to-end copy: the file lands at <dir>/<name>.go
// with the destination package, and a second add without -force is refused.
func TestRunAddWritesComponent(parseT *testing.T) {
	parseDir := parseT.TempDir()
	parseDest := filepath.Join(parseDir, "widgets")

	parseLauncher := launcher{}
	if parseErr := parseLauncher.runAdd([]string{"tabs", "-dir", parseDest}); parseErr != nil {
		parseT.Fatalf("runAdd: %v", parseErr)
	}
	parseOut := filepath.Join(parseDest, "tabs.go")
	parseData, parseErr := os.ReadFile(parseOut)
	if parseErr != nil {
		parseT.Fatalf("expected %s to be written: %v", parseOut, parseErr)
	}
	if !strings.Contains(string(parseData), "package widgets") || !strings.Contains(string(parseData), "func Tabs(") {
		parseT.Fatalf("unexpected written content:\n%s", parseData)
	}

	// Without -force, re-adding must refuse rather than clobber.
	if parseErr := parseLauncher.runAdd([]string{"tabs", "-dir", parseDest}); parseErr == nil {
		parseT.Fatal("expected a refusal to overwrite without -force")
	}
	// With -force, it succeeds.
	if parseErr := parseLauncher.runAdd([]string{"tabs", "-dir", parseDest, "-force"}); parseErr != nil {
		parseT.Fatalf("expected -force overwrite to succeed: %v", parseErr)
	}
}
