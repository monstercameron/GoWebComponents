package main

import (
	"strings"
	"testing"
)

// TestBuildWasmGoEnvDisablesJSONV2 checks the measured wasm size setting while
// preserving unrelated experiments selected by the caller.
func TestBuildWasmGoEnvDisablesJSONV2(parseT *testing.T) {
	parseT.Setenv("GOEXPERIMENT", "jsonv2,simd")
	parseEnv := buildWasmGoEnv()
	parseSeen := 0
	for _, parseEntry := range parseEnv {
		if !strings.HasPrefix(parseEntry, "GOEXPERIMENT=") {
			continue
		}
		parseSeen++
		if parseEntry != "GOEXPERIMENT=simd,nojsonv2" {
			parseT.Errorf("wasm experiments = %q, want simd,nojsonv2", parseEntry)
		}
	}
	if parseSeen != 1 {
		parseT.Errorf("GOEXPERIMENT entries = %d, want 1", parseSeen)
	}
}
