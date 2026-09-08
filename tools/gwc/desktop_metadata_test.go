package main

import "testing"

func TestValidateScaffoldDesktopMetadataAcceptsIsolatedTarget(parseT *testing.T) {
	parseMetadata := scaffoldDesktopMetadata{
		Version:       1,
		NativeEntry:   "cmd/desktop/main.go",
		FrontendEntry: "frontend/main.go",
		AssetsDir:     "assets",
		OutputPath:    "bin/wails-counter.exe",
		WailsVersion:  "v3.0.0-beta.17",
	}
	if parseErr := validateScaffoldDesktopMetadata(&parseMetadata); parseErr != nil {
		parseT.Fatalf("valid desktop metadata rejected: %v", parseErr)
	}
}

func TestValidateScaffoldDesktopMetadataRejectsMalformedPaths(parseT *testing.T) {
	parseCases := []struct {
		name string
		edit func(*scaffoldDesktopMetadata)
	}{
		{name: "absolute", edit: func(parseMetadata *scaffoldDesktopMetadata) { parseMetadata.NativeEntry = `C:\desktop\main.go` }},
		{name: "traversal", edit: func(parseMetadata *scaffoldDesktopMetadata) { parseMetadata.FrontendEntry = "../frontend/main.go" }},
		{name: "missing version", edit: func(parseMetadata *scaffoldDesktopMetadata) { parseMetadata.Version = 2 }},
		{name: "missing wails version", edit: func(parseMetadata *scaffoldDesktopMetadata) { parseMetadata.WailsVersion = "" }},
	}
	for _, parseCase := range parseCases {
		parseT.Run(parseCase.name, func(parseT *testing.T) {
			parseMetadata := scaffoldDesktopMetadata{Version: 1, NativeEntry: "cmd/desktop/main.go", FrontendEntry: "frontend/main.go", AssetsDir: "assets", OutputPath: "bin/app.exe", WailsVersion: "v3.0.0-beta.17"}
			parseCase.edit(&parseMetadata)
			if parseErr := validateScaffoldDesktopMetadata(&parseMetadata); parseErr == nil {
				parseT.Fatal("malformed desktop metadata was accepted")
			}
		})
	}
}

func TestValidateScaffoldDesktopMetadataRejectsOverlappingOutput(parseT *testing.T) {
	for _, parseOutput := range []string{"assets/app.exe", "assets", "frontend/main.go"} {
		parseMetadata := scaffoldDesktopMetadata{Version: 1, NativeEntry: "cmd/desktop/main.go", FrontendEntry: "frontend/main.go", AssetsDir: "assets", OutputPath: parseOutput, WailsVersion: "v3.0.0-beta.17"}
		if parseErr := validateScaffoldDesktopMetadata(&parseMetadata); parseErr == nil {
			parseT.Fatalf("overlapping output %q was accepted", parseOutput)
		}
	}
}

func TestNormalizeScaffoldMetadataPreservesWebFixtureWithoutDesktop(parseT *testing.T) {
	parseMetadata := scaffoldMetadata{SchemaVersion: currentScaffoldMetadataSchemaVersion, ProjectName: "web-only"}
	if parseErr := normalizeScaffoldMetadata(&parseMetadata); parseErr != nil {
		parseT.Fatalf("web metadata rejected: %v", parseErr)
	}
	if parseMetadata.Desktop != nil || parseMetadata.Ownership.ProjectOwnership != "standalone" || parseMetadata.Ownership.FrameworkSourceMode != "module-proxy" {
		parseT.Fatalf("web metadata normalization changed unexpectedly: %#v", parseMetadata)
	}
}
