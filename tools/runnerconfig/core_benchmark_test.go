package runnerconfig

import "testing"

func BenchmarkResolveValueRelative(b *testing.B) {
	b.ReportAllocs()
	configPath := `C:\repo\gwc-runner.json`
	for b.Loop() {
		value, err := ResolveValue(configPath, `tools\go_js_wasm_exec.bat`)
		if err != nil {
			b.Fatalf("ResolveValue: %v", err)
		}
		if value == "" {
			b.Fatal("ResolveValue returned empty path")
		}
	}
}

func BenchmarkArtifactNamespace(b *testing.B) {
	b.ReportAllocs()
	root := `C:\Users\Cam\Desktop\GoWebComponents`
	for b.Loop() {
		value := GetArtifactNamespace(root)
		if value == "" {
			b.Fatal("GetArtifactNamespace returned empty namespace")
		}
	}
}

