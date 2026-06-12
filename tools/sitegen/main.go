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

// faviconSVG is the generated site favicon: a cyan reactive-ring mark on the
// site's dark surface, matching the boot spinner.
const faviconSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32">
<rect width="32" height="32" rx="7" fill="#0a0f1a"/>
<circle cx="16" cy="16" r="8" fill="none" stroke="#22d3ee" stroke-width="3"/>
<circle cx="16" cy="16" r="2.5" fill="#22d3ee"/>
</svg>`

// ogImageSVG is the generated 1200x630 social/link-preview card.
const ogImageSVG = `<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="630" viewBox="0 0 1200 630">
<rect width="1200" height="630" fill="#0a0f1a"/>
<circle cx="150" cy="170" r="54" fill="none" stroke="#22d3ee" stroke-width="12"/>
<circle cx="150" cy="170" r="16" fill="#22d3ee"/>
<text x="240" y="190" font-family="ui-monospace,Consolas,monospace" font-size="76" font-weight="700" fill="#e2e8f0">GoWebComponents</text>
<text x="80" y="360" font-family="ui-monospace,Consolas,monospace" font-size="42" fill="#94a3b8">Go-native UIs for the browser</text>
<text x="80" y="440" font-family="ui-monospace,Consolas,monospace" font-size="30" fill="#64748b">Fine-grained reactivity · SSR + hydration · crash containment</text>
<text x="80" y="495" font-family="ui-monospace,Consolas,monospace" font-size="30" fill="#64748b">Streaming SSR · realtime · i18n · a11y · PWA · feature flags</text>
<rect x="80" y="556" width="1040" height="3" fill="#1e293b"/>
<text x="80" y="600" font-family="ui-monospace,Consolas,monospace" font-size="26" fill="#22d3ee">pure Go, compiled to WebAssembly</text>
</svg>`

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

	// Brand assets are generated alongside the shell so the deployed artifact is
	// self-contained: the favicon and social/OG preview never 404.
	if parseErr3 := os.WriteFile(filepath.Join(parseOutDir, "favicon.svg"), []byte(faviconSVG), 0o644); parseErr3 != nil {
		return fmt.Errorf("write favicon: %w", parseErr3)
	}
	if parseErr4 := os.WriteFile(filepath.Join(parseOutDir, "og-image.svg"), []byte(ogImageSVG), 0o644); parseErr4 != nil {
		return fmt.Errorf("write og image: %w", parseErr4)
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
<link rel="icon" type="image/svg+xml" href="favicon.svg">
<meta property="og:type" content="website">
<meta property="og:title" content="GoWebComponents — Go-native UIs for the browser">
<meta property="og:description" content="Build React-class web UIs in pure Go: fine-grained reactivity, SSR with hydration, and crash containment built in.">
<meta property="og:image" content="og-image.svg">
<meta name="twitter:card" content="summary_large_image">
<meta name="twitter:title" content="GoWebComponents — Go-native UIs for the browser">
<meta name="twitter:description" content="Build React-class web UIs in pure Go: fine-grained reactivity, SSR with hydration, and crash containment built in.">
<meta name="twitter:image" content="og-image.svg">
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
	var FAIL_KEY = "gwc.boot.fails";
	var FAIL_LIMIT = 3;
	function fails() { try { return parseInt(localStorage.getItem(FAIL_KEY) || "0", 10) || 0; } catch (e) { return 0; } }
	function setFails(n) { try { localStorage.setItem(FAIL_KEY, String(n)); } catch (e) {} }
	function purgeAndRetry() {
		setFails(0);
		if (window.caches && caches.keys) {
			caches.keys().then(function (ks) { return Promise.all(ks.map(function (k) { return caches.delete(k); })); })
				.then(function () { location.reload(); }, function () { location.reload(); });
		} else { location.reload(); }
	}
	// Crash-loop safe mode: after FAIL_LIMIT consecutive failed boots, show a
	// minimal diagnostics view (no wasm) instead of re-running a build that just
	// crashes again. The view needs no wasm so it always renders.
	if (fails() >= FAIL_LIMIT) {
		document.getElementById("app").innerHTML =
			'<div style="max-width:640px;margin:12vh auto;padding:0 20px;line-height:1.6;color:#94a3b8">' +
			'<h1 style="color:#e2e8f0">Safe mode</h1>' +
			'<p>This app failed to start ' + fails() + ' times in a row, so it stopped auto-retrying to avoid a crash loop. ' +
			'This is usually a stale cached build.</p>' +
			'<button id="gwc-purge" style="background:#22d3ee;color:#0a0f1a;border:0;border-radius:6px;padding:10px 16px;font:inherit;cursor:pointer">Purge caches and retry</button>' +
			'</div>';
		document.getElementById("gwc-purge").addEventListener("click", purgeAndRetry);
		return;
	}
	// Count this boot attempt up front; a healthy run clears it after a liveness
	// window, so only repeated failures accumulate toward safe mode.
	setFails(fails() + 1);
	var go = new Go();
	var load = ("instantiateStreaming" in WebAssembly)
		? WebAssembly.instantiateStreaming(fetch("site.wasm"), go.importObject)
		: fetch("site.wasm").then(function (resp) { return resp.arrayBuffer(); })
			.then(function (bytes) { return WebAssembly.instantiate(bytes, go.importObject); });
	load.then(function (result) {
			var boot = document.getElementById("boot");
			if (boot && boot.parentNode) boot.parentNode.removeChild(boot);
			go.run(result.instance);
			// Survived the liveness window => boot is healthy, reset the counter.
			setTimeout(function () { setFails(0); }, 4000);
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
