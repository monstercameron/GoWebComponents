// Command sitegen builds the GoWebComponents docs site.
//
// The site itself is a pure GWC application (examples/site): every page,
// style, and behavior is Go compiled to wasm. This tool compiles that app
// and generates the single boot shell — the only HTML in the deployed
// artifact, and it is generated here, never authored.
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	parseRepoRoot := flag.String("root", ".", "repository root")
	parseOutDir := flag.String("out", "examples/site-dist", "output directory for the built site")
	parseServeAddr := flag.String("serve", "", "after building, serve the site locally on this address (e.g. 127.0.0.1:8090)")
	flag.Parse()

	if parseErr := generateSite(*parseRepoRoot, *parseOutDir); parseErr != nil {
		fmt.Fprintln(os.Stderr, "sitegen:", parseErr)
		os.Exit(1)
	}
	if *parseServeAddr != "" {
		if parseErr := servePreview(*parseRepoRoot, *parseOutDir, *parseServeAddr); parseErr != nil {
			fmt.Fprintln(os.Stderr, "sitegen:", parseErr)
			os.Exit(1)
		}
	}
}

// generateSite compiles the docs site wasm app and writes the boot shell.
func generateSite(parseRepoRoot string, parseOutDir string) error {
	if parseErr := os.MkdirAll(parseOutDir, 0o755); parseErr != nil {
		return fmt.Errorf("create output dir: %w", parseErr)
	}

	parseWasmPath := filepath.Join(parseOutDir, "site.wasm")
	parseCmd := exec.Command("go", "build", "-trimpath", "-ldflags", "-s -w", "-o", parseWasmPath, "./examples/site")
	parseCmd.Dir = parseRepoRoot
	parseCmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	if parseOutput, parseErr := parseCmd.CombinedOutput(); parseErr != nil {
		return fmt.Errorf("build site wasm: %w\n%s", parseErr, parseOutput)
	}

	parseShell, parseErr := buildBootShell()
	if parseErr != nil {
		return parseErr
	}
	if parseErr2 := os.WriteFile(filepath.Join(parseOutDir, "index.html"), []byte(parseShell), 0o644); parseErr2 != nil {
		return fmt.Errorf("write boot shell: %w", parseErr2)
	}

	parseInfo, _ := os.Stat(parseWasmPath)
	fmt.Printf("sitegen: built site.wasm (%.1f MB) and generated boot shell in %s\n",
		float64(parseInfo.Size())/(1024*1024), parseOutDir)
	return nil
}

// buildBootShell generates the single HTML document that boots the wasm app.
// The Go runtime loader (wasm_exec.js) is inlined from the local toolchain so
// the deployed artifact is exactly one shell plus one wasm binary.
func buildBootShell() (string, error) {
	parseGoroot, parseErr := exec.Command("go", "env", "GOROOT").Output()
	if parseErr != nil {
		return "", fmt.Errorf("resolve GOROOT: %w", parseErr)
	}
	parseExecPath := filepath.Join(strings.TrimSpace(string(parseGoroot)), "lib", "wasm", "wasm_exec.js")
	parseExecRaw, parseErr2 := os.ReadFile(parseExecPath)
	if parseErr2 != nil {
		return "", fmt.Errorf("read wasm_exec.js: %w", parseErr2)
	}

	return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="description" content="Build React-class web UIs in pure Go: fine-grained reactivity, SSR with hydration, and crash containment built in.">
<title>GoWebComponents — Go-native UIs for the browser</title>
<style>
html,body{margin:0;background:#0a0f1a;color:#64748b;font:14px ui-monospace,Consolas,monospace}
#boot{min-height:100vh;display:flex;align-items:center;justify-content:center;gap:10px}
#boot .dot{width:14px;height:14px;border:2px solid rgba(148,163,184,.3);border-top-color:#22d3ee;border-radius:999px;animation:r .7s linear infinite}
@keyframes r{to{transform:rotate(360deg)}}
</style>
</head>
<body>
<div id="app"><div id="boot"><span class="dot"></span>booting Go…</div></div>
<script>
` + string(parseExecRaw) + `
(function () {
	var go = new Go();
	var load = ("instantiateStreaming" in WebAssembly)
		? WebAssembly.instantiateStreaming(fetch("site.wasm"), go.importObject)
		: fetch("site.wasm").then(function (resp) { return resp.arrayBuffer(); })
			.then(function (bytes) { return WebAssembly.instantiate(bytes, go.importObject); });
	load.then(function (result) {
			var boot = document.getElementById("boot");
			if (boot && boot.parentNode) boot.parentNode.removeChild(boot);
			go.run(result.instance);
		})
		.catch(function (err) {
			var boot = document.getElementById("boot");
			if (boot) boot.textContent = "failed to start: " + err;
		});
})();
</script>
</body>
</html>
`, nil
}

// servePreview hosts the built site with the same path layout as the GitHub
// Pages deployment: the catalog app and shared static assets are mounted
// beside the wasm app shell.
func servePreview(parseRepoRoot string, parseOutDir string, parseAddr string) error {
	parseMux := http.NewServeMux()
	parseMux.Handle("/public-examples-site/", http.StripPrefix("/public-examples-site/",
		http.FileServer(http.Dir(filepath.Join(parseRepoRoot, "examples", "public-examples-site")))))
	parseMux.Handle("/static/", http.StripPrefix("/static/",
		http.FileServer(http.Dir(filepath.Join(parseRepoRoot, "examples", "static")))))
	parseMux.Handle("/", http.FileServer(http.Dir(parseOutDir)))
	fmt.Printf("sitegen: serving site at http://%s/\n", parseAddr)
	return http.ListenAndServe(parseAddr, parseMux)
}
