package runnerconfig

import "testing"

func BenchmarkResolveValueRelative(parseB *testing.B) {
	parseB.ReportAllocs()
	parseConfigPath := `C:\repo\gwc-runner.json`
	for parseB.Loop() {
		parseValue, parseErr := ResolveValue(parseConfigPath, `tools\go_js_wasm_exec.bat`)
		if parseErr != nil {
			parseB.Fatalf("ResolveValue: %v", parseErr)
		}
		if parseValue == "" {
			parseB.Fatal("ResolveValue returned empty path")
		}
	}
}

func BenchmarkArtifactNamespace(parseB *testing.B) {
	parseB.ReportAllocs()
	parseRoot := `C:\Users\Cam\Desktop\GoWebComponents`
	for parseB.Loop() {
		parseValue := GetArtifactNamespace(parseRoot)
		if parseValue == "" {
			parseB.Fatal("GetArtifactNamespace returned empty namespace")
		}
	}
}
