// Command sitegen builds the GoWebComponents docs site.
//
// The site itself is a pure GWC application (examples/site): every page,
// style, and behavior is Go compiled to wasm. This tool compiles that app
// and generates the single boot shell — the only HTML in the deployed
// artifact, and it is generated here, never authored. It also emits a
// Web App Manifest (manifest.json) and a versioned service worker (sw.js)
// so the site is installable and works offline after the first load.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	htmlpkg "html"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/monstercameron/GoWebComponents/v6/pwa"
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

// generateSite compiles the docs site wasm app and writes the boot shell,
// brand assets, Web App Manifest, and service worker.
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

	// Read the wasm once: its full SHA-256 is embedded in the shell for an
	// integrity (SRI-style) check before execution, and its 12-hex prefix names
	// the service-worker cache below.
	parseWasmBytes, parseErr0 := os.ReadFile(parseWasmPath)
	if parseErr0 != nil {
		return fmt.Errorf("read site.wasm: %w", parseErr0)
	}
	parseWasmSHA := fullSHA256Hex(parseWasmBytes)

	parseShell, parseErr := buildBootShell(parseWasmSHA)
	if parseErr != nil {
		return parseErr
	}
	if parseErr2 := os.WriteFile(filepath.Join(parseOutDir, "index.html"), []byte(parseShell), 0o644); parseErr2 != nil {
		return fmt.Errorf("write boot shell: %w", parseErr2)
	}
	parseVersionJSON := buildVersionJSON(parseWasmSHA)
	if parseErrVersion := os.WriteFile(filepath.Join(parseOutDir, "version.json"), []byte(parseVersionJSON), 0o644); parseErrVersion != nil {
		return fmt.Errorf("write version.json: %w", parseErrVersion)
	}

	// Brand assets are generated alongside the shell so the deployed artifact is
	// self-contained: the favicon and social/OG preview never 404.
	if parseErr3 := os.WriteFile(filepath.Join(parseOutDir, "favicon.svg"), []byte(faviconSVG), 0o644); parseErr3 != nil {
		return fmt.Errorf("write favicon: %w", parseErr3)
	}
	if parseErr4 := os.WriteFile(filepath.Join(parseOutDir, "og-image.svg"), []byte(ogImageSVG), 0o644); parseErr4 != nil {
		return fmt.Errorf("write og image: %w", parseErr4)
	}

	// The cache-busting version is the 12-hex prefix of the wasm SHA already
	// computed above, so a new build automatically evicts stale SW caches.
	parseCacheVersion := parseWasmSHA[:12]

	// Web App Manifest — makes the site installable.
	parseManifestJSON, parseErr6 := buildManifestJSON()
	if parseErr6 != nil {
		return fmt.Errorf("build manifest.json: %w", parseErr6)
	}
	if parseErr7 := os.WriteFile(filepath.Join(parseOutDir, "manifest.json"), parseManifestJSON, 0o644); parseErr7 != nil {
		return fmt.Errorf("write manifest.json: %w", parseErr7)
	}

	// Service worker — offline-capable cache-first shell with versioned eviction.
	parseSWAssets := []string{"index.html", "site.wasm", "favicon.svg", "og-image.svg", "manifest.json", "version.json"}
	parseSWJS := buildServiceWorkerJS(parseCacheVersion, parseSWAssets)
	if parseErr8 := os.WriteFile(filepath.Join(parseOutDir, "sw.js"), []byte(parseSWJS), 0o644); parseErr8 != nil {
		return fmt.Errorf("write sw.js: %w", parseErr8)
	}

	// Use the bytes already read above rather than re-Stat'ing the file (avoids a
	// redundant syscall and a nil-deref panic if the file vanishes mid-run).
	fmt.Printf("sitegen: built site.wasm (%.1f MB) and generated boot shell in %s\n",
		float64(len(parseWasmBytes))/(1024*1024), parseOutDir)
	return nil
}

// wasmCacheVersion returns a 12-hex-character prefix of the SHA-256 of the
// wasm binary. A new build always yields a new cache name, which triggers the
// service worker's activate handler to delete all previous caches.
func wasmCacheVersion(parseWasmBytes []byte) string {
	return fullSHA256Hex(parseWasmBytes)[:12]
}

// fullSHA256Hex returns the full lowercase hex SHA-256 of the bytes, used as the
// integrity digest embedded in the boot shell.
func fullSHA256Hex(parseBytes []byte) string {
	parseSum := sha256.Sum256(parseBytes)
	return hex.EncodeToString(parseSum[:])
}

func buildVersionJSON(parseWasmSHA string) string {
	parseBuildID := strings.TrimSpace(parseWasmSHA)
	if len(parseBuildID) > 12 {
		parseBuildID = parseBuildID[:12]
	}
	return fmt.Sprintf("{\"buildId\":\"%s\",\"sha256\":\"%s\"}\n", parseBuildID, strings.TrimSpace(parseWasmSHA))
}

// buildManifestJSON returns indented JSON for the site's Web App Manifest
// using the canonical pwa.Manifest type and pwa.MarshalManifestJSONIndented.
func buildManifestJSON() ([]byte, error) {
	parseManifest := pwa.Manifest{
		Name:            "GoWebComponents",
		ShortName:       "GWC",
		StartURL:        ".",
		Display:         pwa.ManifestDisplayStandalone,
		ThemeColor:      "#0a0f1a",
		BackgroundColor: "#22d3ee",
		Icons: []pwa.ManifestImage{
			{
				Src:     "favicon.svg",
				Type:    "image/svg+xml",
				Sizes:   "any",
				Purpose: "any maskable",
			},
		},
	}
	return pwa.MarshalManifestJSONIndented(parseManifest, "", "  ")
}

// buildServiceWorkerJS generates a vanilla-JS service worker that precaches
// the shell artifacts under a versioned cache name. On install it fetches all
// assets into the cache; on fetch it serves cache-first with a network
// fallback; on activate it deletes every cache whose name starts with the
// shared prefix but does not match the current version, so stale builds are
// evicted automatically.
//
// parseCacheVersion is a short hash derived from the wasm binary (see
// wasmCacheVersion). parseAssets is the list of URLs to precache relative to
// the service worker scope.
func buildServiceWorkerJS(parseCacheVersion string, parseAssets []string) string {
	// Build the JS array literal for the precache list.
	var parsePrecache strings.Builder
	parsePrecache.WriteString("[")
	for parseI, parseAsset := range parseAssets {
		if parseI > 0 {
			parsePrecache.WriteString(", ")
		}
		parsePrecache.WriteString("\"")
		parsePrecache.WriteString(parseAsset)
		parsePrecache.WriteString("\"")
	}
	parsePrecache.WriteString("]")

	// The service worker is assembled with double-quoted Go strings so it can
	// safely live inside a file separate from the backtick boot-shell template.
	var parseSW strings.Builder
	parseSW.WriteString("// Generated by sitegen — do not edit.\n")
	parseSW.WriteString("var CACHE_PREFIX = \"gwc-shell\";\n")
	parseSW.WriteString("var CACHE_VERSION = \"gwc-shell-" + parseCacheVersion + "\";\n")
	parseSW.WriteString("var PRECACHE_ASSETS = " + parsePrecache.String() + ";\n")
	parseSW.WriteString("\n")
	parseSW.WriteString("self.addEventListener(\"install\", function(event) {\n")
	parseSW.WriteString("  event.waitUntil(\n")
	parseSW.WriteString("    caches.open(CACHE_VERSION).then(function(cache) {\n")
	parseSW.WriteString("      return cache.addAll(PRECACHE_ASSETS);\n")
	parseSW.WriteString("    }).then(function() {\n")
	parseSW.WriteString("      return self.skipWaiting();\n")
	parseSW.WriteString("    })\n")
	parseSW.WriteString("  );\n")
	parseSW.WriteString("});\n")
	parseSW.WriteString("\n")
	parseSW.WriteString("self.addEventListener(\"activate\", function(event) {\n")
	parseSW.WriteString("  event.waitUntil(\n")
	parseSW.WriteString("    caches.keys().then(function(keys) {\n")
	parseSW.WriteString("      return Promise.all(\n")
	parseSW.WriteString("        keys\n")
	parseSW.WriteString("          .filter(function(key) {\n")
	parseSW.WriteString("            return key.indexOf(CACHE_PREFIX) === 0 && key !== CACHE_VERSION;\n")
	parseSW.WriteString("          })\n")
	parseSW.WriteString("          .map(function(key) { return caches.delete(key); })\n")
	parseSW.WriteString("      );\n")
	parseSW.WriteString("    }).then(function() {\n")
	parseSW.WriteString("      return self.clients.claim();\n")
	parseSW.WriteString("    })\n")
	parseSW.WriteString("  );\n")
	parseSW.WriteString("});\n")
	parseSW.WriteString("\n")
	parseSW.WriteString("self.addEventListener(\"fetch\", function(event) {\n")
	parseSW.WriteString("  if (event.request.method !== \"GET\") { return; }\n")
	parseSW.WriteString("  event.respondWith(\n")
	parseSW.WriteString("    caches.match(event.request).then(function(cached) {\n")
	parseSW.WriteString("      if (cached) { return cached; }\n")
	parseSW.WriteString("      return fetch(event.request).then(function(response) {\n")
	parseSW.WriteString("        if (!response || response.status !== 200 || response.type === \"opaque\") {\n")
	parseSW.WriteString("          return response;\n")
	parseSW.WriteString("        }\n")
	parseSW.WriteString("        var toCache = response.clone();\n")
	parseSW.WriteString("        caches.open(CACHE_VERSION).then(function(cache) {\n")
	parseSW.WriteString("          cache.put(event.request, toCache);\n")
	parseSW.WriteString("        });\n")
	parseSW.WriteString("        return response;\n")
	parseSW.WriteString("      }).catch(function() {\n")
	parseSW.WriteString("        return caches.match(\"index.html\");\n")
	parseSW.WriteString("      });\n")
	parseSW.WriteString("    })\n")
	parseSW.WriteString("  );\n")
	parseSW.WriteString("});\n")

	return parseSW.String()
}

// buildBootShell generates the single HTML document that boots the wasm app.
// The Go runtime loader (wasm_exec.js) is inlined from the local toolchain so
// the deployed artifact is exactly one shell plus one wasm binary.
// The shell also links the Web App Manifest and registers the service worker,
// and verifies the integrity of site.wasm (its embedded SHA-256) before
// executing it, so a tampered or truncated artifact is refused rather than run.
func buildBootShell(parseWasmSHA string) (string, error) {
	return buildBootShellWithNonce(parseWasmSHA, "")
}

func buildBootShellWithNonce(parseWasmSHA string, parseNonce string) (string, error) {
	parseGoroot, parseErr := exec.Command("go", "env", "GOROOT").Output()
	if parseErr != nil {
		return "", fmt.Errorf("resolve GOROOT: %w", parseErr)
	}
	parseExecPath := filepath.Join(strings.TrimSpace(string(parseGoroot)), "lib", "wasm", "wasm_exec.js")
	parseExecRaw, parseErr2 := os.ReadFile(parseExecPath)
	if parseErr2 != nil {
		return "", fmt.Errorf("read wasm_exec.js: %w", parseErr2)
	}
	parseNonceAttr := buildCSPNonceAttr(parseNonce)

	return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="description" content="Build React-class web UIs in pure Go: fine-grained reactivity, SSR with hydration, and crash containment built in.">
<meta name="theme-color" content="#0a0f1a">
<title>GoWebComponents — Go-native UIs for the browser</title>
<link rel="icon" type="image/svg+xml" href="favicon.svg">
<link rel="manifest" href="manifest.json">
<meta property="og:type" content="website">
<meta property="og:title" content="GoWebComponents — Go-native UIs for the browser">
<meta property="og:description" content="Build React-class web UIs in pure Go: fine-grained reactivity, SSR with hydration, and crash containment built in.">
<meta property="og:image" content="og-image.svg">
<meta name="twitter:card" content="summary_large_image">
<meta name="twitter:title" content="GoWebComponents — Go-native UIs for the browser">
<meta name="twitter:description" content="Build React-class web UIs in pure Go: fine-grained reactivity, SSR with hydration, and crash containment built in.">
<meta name="twitter:image" content="og-image.svg">
<style` + parseNonceAttr + `>
html,body{margin:0;background:#0a0f1a;color:#64748b;font:14px ui-monospace,Consolas,monospace}
#boot{min-height:100vh;display:flex;align-items:center;justify-content:center;gap:10px}
#boot .dot{width:14px;height:14px;border:2px solid rgba(148,163,184,.3);border-top-color:#22d3ee;border-radius:999px;animation:r .7s linear infinite}
@keyframes r{to{transform:rotate(360deg)}}
</style>
</head>
<body>
<div id="app"><div id="boot"><span class="dot"></span>booting Go…</div></div>
<script` + parseNonceAttr + `>
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
	var EXPECTED_WASM_SHA = "` + parseWasmSHA + `";
	var BUILD_ID = EXPECTED_WASM_SHA.slice(0, 12);
	var VERSION_RELOAD_KEY = "gwc.version.reload." + BUILD_ID;
	var VERSION_SNAPSHOT_KEY = "gwc.version.snapshot." + BUILD_ID;
	function toHex(buf) { var b = new Uint8Array(buf), s = ""; for (var i = 0; i < b.length; i++) { s += b[i].toString(16).padStart(2, "0"); } return s; }
	function storageGet(store, key) { try { return store.getItem(key); } catch (e) { return ""; } }
	function storageSet(store, key, value) { try { store.setItem(key, value); } catch (e) {} }
	function captureRefreshSnapshot() {
		var keys = ["gwc.state.snapshot", "gwc:ssr:state", "__GWC_BOOTSTRAP__"];
		for (var i = 0; i < keys.length; i++) {
			var value = storageGet(localStorage, keys[i]) || storageGet(sessionStorage, keys[i]);
			if (value) { storageSet(sessionStorage, VERSION_SNAPSHOT_KEY, value); return; }
		}
	}
	function checkVersionSkew() {
		return fetch("version.json", { cache: "reload" })
			.then(function (resp) { return resp && resp.ok ? resp.json() : null; })
			.then(function (meta) {
				var serverBuild = meta && (meta.buildId || (meta.sha256 || "").slice(0, 12));
				if (!serverBuild || serverBuild === BUILD_ID) return false;
				if (storageGet(sessionStorage, VERSION_RELOAD_KEY)) return false;
				storageSet(sessionStorage, VERSION_RELOAD_KEY, "1");
				captureRefreshSnapshot();
				return true;
			}, function () { return false; });
	}
	// Subresource-integrity for the wasm: fetch the bytes, verify their SHA-256
	// against the digest embedded at build time, and only instantiate on a match.
	// Falls back to instantiating without the check only when crypto.subtle is
	// unavailable (insecure context), so the app still boots in dev over plain http.
	var load = checkVersionSkew().then(function (shouldRefresh) {
			if (shouldRefresh) {
				return fetch("index.html", { cache: "reload" }).then(function () {
					location.reload();
					return new Promise(function () {});
				}, function () {
					location.reload();
					return new Promise(function () {});
				});
			}
			return fetch("site.wasm");
		})
		.then(function (resp) { return resp.arrayBuffer(); })
		.then(function (bytes) {
			if (!self.crypto || !crypto.subtle) {
				return WebAssembly.instantiate(bytes, go.importObject);
			}
			return crypto.subtle.digest("SHA-256", bytes).then(function (digest) {
				if (toHex(digest) !== EXPECTED_WASM_SHA) {
					throw new Error("integrity check failed: site.wasm digest mismatch");
				}
				return WebAssembly.instantiate(bytes, go.importObject);
			});
		});
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
	// Register the service worker after the wasm boot is underway.
	// Failure is swallowed so a missing sw.js never blocks the app.
	if ("serviceWorker" in navigator) {
		navigator.serviceWorker.register("sw.js").catch(function () {});
	}
})();
</script>
</body>
</html>
`, nil
}

func buildCSPNonceAttr(parseNonce string) string {
	parseNonce = strings.TrimSpace(parseNonce)
	if parseNonce == "" {
		return ""
	}
	return ` nonce="` + htmlpkg.EscapeString(parseNonce) + `"`
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
