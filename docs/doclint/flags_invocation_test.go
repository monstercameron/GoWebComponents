package doclint

import "testing"

// TestScanFileFlagsRecognizesLauncherFormsWithoutGoToolFalsePositives pins invocation boundaries and separates Go flags from launcher flags.
func TestScanFileFlagsRecognizesLauncherFormsWithoutGoToolFalsePositives(parseTest *testing.T) {
	for _, parseCommand := range []string{
		"gwc build -removed-flag",
		"./gwc build -removed-flag",
		"gwc.exe build -removed-flag",
		`.\gwc.exe build -removed-flag`,
		"$ go run ./tools/gwc build -removed-flag",
		`PS> go run .\tools\gwc build -removed-flag`,
		"go run -tags gwc_desktop ./tools/gwc build -removed-flag",
		"go run -tags=gwc_desktop -ldflags '-s -w' ./tools/gwc build -removed-flag",
	} {
		parseTest.Run(parseCommand, func(parseTest *testing.T) {
			parseRefs := scanFileFlags("```powershell\n"+parseCommand+"\n```\n", "fixture.md", map[string]bool{})
			if len(parseRefs) != 1 || parseRefs[0].Flag != "removed-flag" || parseRefs[0].Line != 2 {
				parseTest.Fatalf("expected only launcher flag, got %+v", parseRefs)
			}
		})
	}
	for _, parseCommand := range []string{
		"go test ./tools/gwc -count=1 -v",
		"go vet ./tools/gwc -fake",
		"echo 'gwc -fake'",
		"go run ./tools/gwc-extra -fake",
		"go run ./another ./tools/gwc -fake",
		"go run -tags playwrightgo ./tools/gwc build -app main.go",
	} {
		parseRefs := scanFileFlags("```sh\n"+parseCommand+"\n```\n", "fixture.md", map[string]bool{"app": true})
		if len(parseRefs) != 0 {
			parseTest.Errorf("non-launcher flag reported for %q: %+v", parseCommand, parseRefs)
		}
	}
}
