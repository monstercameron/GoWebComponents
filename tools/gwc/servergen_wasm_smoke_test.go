package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestGeneratedClientStubCompilesForWasm is the FB1 audit smoke: it generates the client
// stub from a //gwc:server function and proves the result actually compiles for the browser
// target (GOOS=js GOARCH=wasm) with the shared types present — catching any drift that makes
// generated code valid-looking but non-building. It builds inside the repo module (so the
// serverfn import resolves from the existing go.sum) in a unique temp package that is removed
// afterward. Skipped in -short (spawns a wasm build).
func TestGeneratedClientStubCompilesForWasm(parseT *testing.T) {
	if testing.Short() {
		parseT.Skip("skipping wasm-compile smoke in -short")
	}
	parseRepoRoot, parseErr := filepath.Abs(filepath.Join("..", ".."))
	if parseErr != nil {
		parseT.Fatalf("resolve repo root: %v", parseErr)
	}

	parsePkgRel := "internal/gwcfb1smoke"
	parseDir := filepath.Join(parseRepoRoot, parsePkgRel)
	if parseErr := os.MkdirAll(parseDir, 0755); parseErr != nil {
		parseT.Fatalf("mkdir smoke pkg: %v", parseErr)
	}
	parseT.Cleanup(func() { _ = os.RemoveAll(parseDir) })

	writeFile(parseT, parseDir, "types.go", "package gwcfb1smoke\n\ntype PingReq struct {\n\tName string `json:\"name\"`\n}\n\ntype PingResp struct {\n\tReply string `json:\"reply\"`\n}\n")
	writeFile(parseT, parseDir, "fn.go", "//go:build !js || !wasm\n\npackage gwcfb1smoke\n\nimport \"context\"\n\n//gwc:server\nfunc Ping(ctx context.Context, req PingReq) (PingResp, error) {\n\treturn PingResp{Reply: \"hi \" + req.Name}, nil\n}\n")

	parsePkgName, parseFuncs, parseErr := collectServerFuncs(parseDir)
	if parseErr != nil {
		parseT.Fatalf("collectServerFuncs: %v", parseErr)
	}
	parseClient, parseErr := generateServerClientFile(parsePkgName, parseFuncs)
	if parseErr != nil {
		parseT.Fatalf("generate client: %v", parseErr)
	}
	parseServer, parseErr := generateServerRegistrationFile(parsePkgName, parseFuncs)
	if parseErr != nil {
		parseT.Fatalf("generate server: %v", parseErr)
	}
	writeFile(parseT, parseDir, serverClientGenFile, parseClient)
	writeFile(parseT, parseDir, serverServerGenFile, parseServer)

	// Browser target: the generated client stub + shared types must compile (the server fn
	// is excluded by its build tag, so the stub does not collide).
	parseCmd := exec.Command("go", "build", "./"+parsePkgRel)
	parseCmd.Dir = parseRepoRoot
	parseCmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	if parseOut, parseBuildErr := parseCmd.CombinedOutput(); parseBuildErr != nil {
		parseT.Fatalf("generated client stub failed to compile for wasm:\n%s", parseOut)
	}
}

func writeFile(parseT *testing.T, parseDir, parseName, parseContent string) {
	parseT.Helper()
	if parseErr := os.WriteFile(filepath.Join(parseDir, parseName), []byte(parseContent), 0644); parseErr != nil {
		parseT.Fatalf("write %s: %v", parseName, parseErr)
	}
}
