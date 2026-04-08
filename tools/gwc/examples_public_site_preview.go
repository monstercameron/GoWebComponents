package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var examplesPublicSitePopupHostFiles = map[string][]string{
	"multi-client-binary":   {"multi-client-binary-popup.html"},
	"multi-client-presence": {"multi-client-presence-popup.html"},
	"multi-window-console":  {"multi-window-console-popup.html"},
}

// stageExamplesPublicSitePreviewRoots stages one runnable preview root per public example.
func stageExamplesPublicSitePreviewRoots(parseL launcher, buildConfigs []buildConfig) error {
	parsePreviewRoot := filepath.Join(parseL.examplesDir, "public-examples-site", "assets", "examples")
	if parseErr := os.RemoveAll(parsePreviewRoot); parseErr != nil {
		return fmt.Errorf("clear public example preview roots: %w", parseErr)
	}
	if parseErr := os.MkdirAll(parsePreviewRoot, 0755); parseErr != nil {
		return fmt.Errorf("create public example preview root: %w", parseErr)
	}

	for _, parseBuildConfig := range buildConfigs {
		parseBinaryName := filepath.Base(parseBuildConfig.outputPath)
		if parseBinaryName == "public-examples-site.wasm" {
			continue
		}

		parseExampleSlug := strings.TrimSuffix(parseBinaryName, filepath.Ext(parseBinaryName))
		parseExampleSourceDir := filepath.Join(parseL.examplesDir, "public", parseExampleSlug)
		parseExamplePreviewDir := filepath.Join(parsePreviewRoot, parseExampleSlug)

		if parseErr := copyExamplesPreviewRuntimeAssets(parseExampleSourceDir, parseExamplePreviewDir); parseErr != nil {
			return parseErr
		}
		if parseErr := copyExamplesBuildBinary(parseBuildConfig.outputPath, filepath.Join(parseExamplePreviewDir, "app.wasm")); parseErr != nil {
			return parseErr
		}
		if parseErr := writeExamplesPublicSitePreviewHost(filepath.Join(parseExamplePreviewDir, "index.html")); parseErr != nil {
			return parseErr
		}
		if parseErr := writeExamplesPublicSitePreviewAliases(parseExamplePreviewDir, examplesPublicSitePopupHostFiles[parseExampleSlug]); parseErr != nil {
			return parseErr
		}
	}

	return nil
}

// copyExamplesPreviewRuntimeAssets copies non-Go runtime assets into one staged preview root.
func copyExamplesPreviewRuntimeAssets(parseSourceRoot string, parseTargetRoot string) error {
	parseSourceInfo, parseErr := os.Stat(parseSourceRoot)
	if parseErr != nil {
		return fmt.Errorf("inspect public example source directory: %w", parseErr)
	}
	if !parseSourceInfo.IsDir() {
		return fmt.Errorf("public example source path is not a directory: %s", parseSourceRoot)
	}

	return filepath.WalkDir(parseSourceRoot, func(parseSourcePath string, parseDirEntry os.DirEntry, parseWalkErr error) error {
		if parseWalkErr != nil {
			return parseWalkErr
		}

		parseRelativePath, parseErr := filepath.Rel(parseSourceRoot, parseSourcePath)
		if parseErr != nil {
			return fmt.Errorf("resolve preview runtime relative path: %w", parseErr)
		}
		parseTargetPath := filepath.Join(parseTargetRoot, parseRelativePath)

		if parseDirEntry.IsDir() {
			if parseErr := os.MkdirAll(parseTargetPath, 0755); parseErr != nil {
				return fmt.Errorf("create preview runtime directory: %w", parseErr)
			}
			return nil
		}
		if !shouldCopyExamplesPreviewAsset(parseSourcePath) {
			return nil
		}
		return copyExamplesFile(parseSourcePath, parseTargetPath)
	})
}

// shouldCopyExamplesPreviewAsset reports whether one public example file should be copied into the runnable preview root.
func shouldCopyExamplesPreviewAsset(parseSourcePath string) bool {
	parseExtension := strings.ToLower(filepath.Ext(parseSourcePath))
	switch parseExtension {
	case ".go", ".md":
		return false
	default:
		return true
	}
}

// writeExamplesPublicSitePreviewHost writes the generic HTML shell that boots one staged public example wasm binary.
func writeExamplesPublicSitePreviewHost(parseTargetPath string) error {
	if parseErr := os.MkdirAll(filepath.Dir(parseTargetPath), 0755); parseErr != nil {
		return fmt.Errorf("create preview host directory: %w", parseErr)
	}
	if parseErr := os.WriteFile(parseTargetPath, []byte(buildExamplesPublicSitePreviewHostHTML()), 0644); parseErr != nil {
		return fmt.Errorf("write preview host html: %w", parseErr)
	}
	return nil
}

// writeExamplesPublicSitePreviewAliases writes any additional HTML entrypoints that a previewed example expects at runtime.
func writeExamplesPublicSitePreviewAliases(parsePreviewDir string, parseFileNames []string) error {
	for _, parseFileName := range parseFileNames {
		parseFileName = strings.TrimSpace(parseFileName)
		if parseFileName == "" {
			continue
		}
		if parseErr := writeExamplesPublicSitePreviewHost(filepath.Join(parsePreviewDir, parseFileName)); parseErr != nil {
			return parseErr
		}
	}
	return nil
}

// buildExamplesPublicSitePreviewHostHTML returns the static HTML shell used to boot one staged preview example.
func buildExamplesPublicSitePreviewHostHTML() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>GoWebComponents Example Preview</title>
    <link rel="stylesheet" href="../../../../static/css/tailwind.css">
    <link rel="stylesheet" href="../../../../static/css/example-shell.css">
    <script src="../../../../static/script/wasm_exec.js"></script>
    <script src="../../../../static/script/example-logger.js"></script>
    <style>
        html {
            scrollbar-gutter: stable;
        }

        body {
            min-height: 100vh;
            background:
                radial-gradient(circle at top left, rgba(34, 211, 238, 0.16), transparent 28%),
                radial-gradient(circle at top right, rgba(16, 185, 129, 0.10), transparent 22%),
                linear-gradient(180deg, #08111d 0%, #0b1523 100%);
        }

        #preview-shell {
            position: fixed;
            inset: 0;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 1rem;
            transition: opacity 220ms ease;
            z-index: 20;
        }

        #preview-shell[data-state="ready"] {
            opacity: 0;
            pointer-events: none;
        }

        #preview-shell-card {
            width: min(100%, 34rem);
            border: 1px solid rgba(255, 255, 255, 0.10);
            border-radius: 1.5rem;
            background: rgba(255, 255, 255, 0.05);
            box-shadow: 0 24px 80px rgba(0, 0, 0, 0.35);
            backdrop-filter: blur(20px);
            padding: 1.5rem;
        }

        #preview-progress-track {
            width: 100%;
            height: 0.75rem;
            border-radius: 9999px;
            overflow: hidden;
            background: rgba(148, 163, 184, 0.14);
            border: 1px solid rgba(255, 255, 255, 0.08);
        }

        #preview-progress-bar {
            height: 100%;
            width: 0%;
            border-radius: 9999px;
            background: linear-gradient(90deg, rgba(34, 211, 238, 0.92), rgba(16, 185, 129, 0.95));
            transition: width 120ms linear;
        }

        #preview-shell[data-indeterminate="true"] #preview-progress-bar {
            width: 35%;
            animation: preview-progress-indeterminate 1.1s ease-in-out infinite;
        }

        @keyframes preview-progress-indeterminate {
            0% { transform: translateX(-120%); }
            100% { transform: translateX(320%); }
        }

        #preview-error-box {
            display: none;
        }

        #preview-shell[data-state="error"] #preview-error-box {
            display: block;
        }

        #preview-shell[data-state="error"] #preview-progress-track {
            display: none;
        }

        #app[data-boot="pending"] {
            visibility: hidden;
        }

        #app {
            container-type: inline-size;
        }

        #app p[class*="text-slate-300"],
        #app p[class*="text-slate-400"],
        #app p[class*="text-slate-200"],
        #app p[class*="text-cyan-50/80"],
        #app p[class*="text-cyan-100/80"],
        #app p[class*="text-emerald-50/90"],
        #app p[class*="text-rose-100"],
        #app p[class*="leading-7"],
        #app p[class*="leading-8"] {
            font-size: 0.78rem !important;
            line-height: 1.4 !important;
            max-width: 40rem;
            text-wrap: balance;
        }

        #app .gwc-example-panel p[class*="text-slate-300"],
        #app .gwc-example-panel p[class*="text-slate-400"],
        #app .gwc-example-panel p[class*="text-slate-200"],
        #app .gwc-example-panel p[class*="text-cyan-50/80"],
        #app .gwc-example-panel p[class*="text-cyan-100/80"],
        #app .gwc-example-panel p[class*="text-emerald-50/90"],
        #app .gwc-example-panel p[class*="text-rose-100"],
        #app .gwc-example-panel p[class*="leading-7"],
        #app .gwc-example-panel p[class*="leading-8"] {
            display: -webkit-box;
            -webkit-line-clamp: 2;
            -webkit-box-orient: vertical;
            overflow: hidden;
        }

        #app .gwc-example-panel ul[class*="text-slate-300"],
        #app .gwc-example-panel ul[class*="text-slate-200"],
        #app .gwc-example-panel ul[class*="text-slate-400"] {
            gap: 0.45rem !important;
            margin-top: 0.5rem !important;
        }

        #app .gwc-example-panel ul[class*="text-slate-300"] li:nth-child(n+3),
        #app .gwc-example-panel ul[class*="text-slate-200"] li:nth-child(n+3),
        #app .gwc-example-panel ul[class*="text-slate-400"] li:nth-child(n+3) {
            display: none;
        }
    </style>
</head>
<body class="bg-[#08111d] text-white min-h-screen example-shell">
    <div id="preview-shell" data-state="loading" data-indeterminate="true" aria-live="polite">
        <div id="preview-shell-card">
            <div class="text-xs uppercase tracking-[0.18em] text-cyan-200">GoWebComponents preview</div>
            <h1 class="mt-3 text-3xl font-semibold tracking-tight text-white">Loading example</h1>
            <p id="preview-status" class="mt-3 text-sm leading-7 text-slate-300">Preparing WebAssembly runtime...</p>
            <div class="mt-5 flex items-center justify-between gap-3 text-xs uppercase tracking-[0.18em] text-slate-400">
                <span id="preview-phase">Starting</span>
                <span id="preview-percent">0%</span>
            </div>
            <div id="preview-progress-track" class="mt-3">
                <div id="preview-progress-bar"></div>
            </div>
            <div id="preview-error-box" class="mt-5 rounded-2xl border border-rose-400/20 bg-rose-400/10 px-4 py-3 text-sm leading-6 text-rose-100"></div>
        </div>
    </div>
    <div id="app" data-boot="pending"></div>
    <script>
        window.__gwcExampleMountSelector = "#app";

        (async function bootstrapExamplePreview() {
            const shell = document.getElementById("preview-shell");
            const appRoot = document.getElementById("app");
            const statusNode = document.getElementById("preview-status");
            const percentNode = document.getElementById("preview-percent");
            const phaseNode = document.getElementById("preview-phase");
            const progressBar = document.getElementById("preview-progress-bar");
            const errorBox = document.getElementById("preview-error-box");
            const pageVersion = new URLSearchParams(window.location.search).get("v");
            const wasmURL = pageVersion ? "./app.wasm?v=" + encodeURIComponent(pageVersion) : "./app.wasm";
            const previewCacheName = "gwc-public-examples-preview-v2";
            const previewCacheLegacyName = "gwc-public-examples-preview-v1";
            const previewCacheRecordsKey = "gwc-public-examples-preview-records-v2";
            const previewCacheMaxAgeMs = 10 * 60 * 1000;
            const previewCacheRetainCount = 2;

            function getPreviewCacheURL(url) {
                try {
                    return new URL(url, window.location.href).href;
                } catch (error) {
                    return url;
                }
            }

            function getPreviewCacheGroup(url) {
                try {
                    const cacheURL = new URL(url, window.location.href);
                    return cacheURL.origin + cacheURL.pathname;
                } catch (error) {
                    return url;
                }
            }

            function getPreviewCacheRecords() {
                if (typeof localStorage === "undefined") {
                    return [];
                }
                try {
                    const rawValue = localStorage.getItem(previewCacheRecordsKey);
                    const parsedValue = JSON.parse(rawValue || "[]");
                    if (!Array.isArray(parsedValue)) {
                        return [];
                    }
                    return parsedValue.filter((record) => record && typeof record.url === "string" && typeof record.usedAt === "number");
                } catch (error) {
                    return [];
                }
            }

            function storePreviewCacheRecords(records) {
                if (typeof localStorage === "undefined") {
                    return;
                }
                try {
                    localStorage.setItem(previewCacheRecordsKey, JSON.stringify(records));
                } catch (error) {
                    return;
                }
            }

            async function clearPreviewCacheOverflow(activeURL) {
                if (typeof caches === "undefined" || !caches || typeof caches.open !== "function") {
                    return;
                }

                if (typeof caches.delete === "function") {
                    try {
                        await caches.delete(previewCacheLegacyName);
                    } catch (error) {
                        // Ignore best-effort cache cleanup failures.
                    }
                }

                const cache = await caches.open(previewCacheName);
                const cacheKeys = await cache.keys();
                const cacheURLs = new Set(cacheKeys.map((request) => request.url));
                const activeRecord = { url: activeURL, usedAt: Date.now() };

                let records = getPreviewCacheRecords().filter((record) => cacheURLs.has(record.url) && activeRecord.usedAt-record.usedAt <= previewCacheMaxAgeMs);
                const activeGroup = getPreviewCacheGroup(activeRecord.url);
                records = records.filter((record) => getPreviewCacheGroup(record.url) !== activeGroup);
                records.unshift(activeRecord);

                const retainRecords = records.slice(0, previewCacheRetainCount);
                const retainURLs = new Set(retainRecords.map((record) => record.url));
                for (const request of cacheKeys) {
                    if (!retainURLs.has(request.url)) {
                        await cache.delete(request);
                    }
                }
                storePreviewCacheRecords(retainRecords);
            }

            function setProgress(percent, options) {
                const resolved = options || {};
                const clamped = Math.max(0, Math.min(100, Math.round(percent)));
                shell.dataset.indeterminate = resolved.indeterminate ? "true" : "false";
                progressBar.style.width = clamped + "%";
                percentNode.textContent = resolved.indeterminate ? "..." : clamped + "%";
                if (resolved.phase) {
                    phaseNode.textContent = resolved.phase;
                }
                if (resolved.message) {
                    statusNode.textContent = resolved.message;
                }
            }

            function setError(detail) {
                shell.dataset.state = "error";
                shell.dataset.indeterminate = "false";
                phaseNode.textContent = "Compatibility error";
                percentNode.textContent = "Failed";
                statusNode.textContent = detail;
                errorBox.textContent = detail;
                console.error("Failed to load preview example:", detail);
            }

            function shouldStreamInstantiate(response) {
                if (typeof WebAssembly.instantiateStreaming !== "function") {
                    return false;
                }
                if (!response.body || typeof response.body.getReader !== "function") {
                    return false;
                }
                const contentTypeHeader = response.headers.get("content-type") || "";
                return contentTypeHeader.toLowerCase().includes("application/wasm");
            }

            async function fetchWasmWithProgress(url) {
                setProgress(5, {
                    indeterminate: true,
                    phase: "Connecting",
                    message: "Requesting the example WebAssembly binary..."
                });

                const response = await fetch(url, { cache: "no-store" });
                if (!response.ok) {
                    throw new Error("The example WebAssembly binary could not be fetched (" + response.status + " " + response.statusText + ").");
                }
                if (typeof caches !== "undefined" && caches && typeof caches.open === "function") {
                    const cacheURL = getPreviewCacheURL(url);
                    const cache = await caches.open(previewCacheName);
                    await cache.put(cacheURL, response.clone());
                    await clearPreviewCacheOverflow(cacheURL);
                }

                if (shouldStreamInstantiate(response)) {
                    setProgress(70, {
                        indeterminate: true,
                        phase: "Downloading",
                        message: "Downloading and compiling example WebAssembly..."
                    });
                    return { shouldStream: true, response };
                }

                const contentLengthHeader = response.headers.get("content-length");
                const totalBytes = contentLengthHeader ? Number(contentLengthHeader) : 0;
                if (!response.body || typeof response.body.getReader !== "function") {
                    setProgress(50, {
                        indeterminate: true,
                        phase: "Downloading",
                        message: "Downloading example WebAssembly binary..."
                    });
                    return { shouldStream: false, bytes: new Uint8Array(await response.arrayBuffer()) };
                }

                const reader = response.body.getReader();
                const chunks = [];
                let receivedBytes = 0;
                while (true) {
                    const result = await reader.read();
                    if (result.done) {
                        break;
                    }
                    chunks.push(result.value);
                    receivedBytes += result.value.length;
                    if (totalBytes > 0) {
                        const progress = 10 + ((receivedBytes / totalBytes) * 70);
                        setProgress(progress, {
                            indeterminate: false,
                            phase: "Downloading",
                            message: "Downloading example WebAssembly binary..."
                        });
                    } else {
                        setProgress(50, {
                            indeterminate: true,
                            phase: "Downloading",
                            message: "Downloading example WebAssembly binary..."
                        });
                    }
                }

                const bytes = new Uint8Array(receivedBytes);
                let offset = 0;
                for (const chunk of chunks) {
                    bytes.set(chunk, offset);
                    offset += chunk.length;
                }
                return { shouldStream: false, bytes };
            }

            async function loadCachedWasmWithProgress(url) {
                if (typeof caches !== "undefined" && caches && typeof caches.open === "function") {
                    const cacheURL = getPreviewCacheURL(url);
                    const cache = await caches.open(previewCacheName);
                    const cachedResponse = await cache.match(cacheURL);
                    if (cachedResponse) {
                        await clearPreviewCacheOverflow(cacheURL);
                        setProgress(60, {
                            indeterminate: true,
                            phase: "Cache",
                            message: "Loading cached example WebAssembly..."
                        });
                        if (shouldStreamInstantiate(cachedResponse)) {
                            return { shouldStream: true, response: cachedResponse };
                        }
                        return { shouldStream: false, bytes: new Uint8Array(await cachedResponse.arrayBuffer()) };
                    }
                }
                return fetchWasmWithProgress(url);
            }

            try {
                if (typeof WebAssembly === "undefined") {
                    throw new Error("WebAssembly is not available in this browser.");
                }
                if (typeof Go === "undefined") {
                    throw new Error("The Go WebAssembly runtime script did not load.");
                }
                if (typeof fetch !== "function") {
                    throw new Error("Fetch is not available in this browser.");
                }

                const wasmPayload = await loadCachedWasmWithProgress(wasmURL);

                const go = new Go();
                let result;
                if (wasmPayload.shouldStream) {
                    result = await WebAssembly.instantiateStreaming(Promise.resolve(wasmPayload.response), go.importObject);
                } else {
                    setProgress(88, {
                        indeterminate: false,
                        phase: "Compiling",
                        message: "Compiling and instantiating example WebAssembly..."
                    });
                    result = await WebAssembly.instantiate(wasmPayload.bytes, go.importObject);
                }

                setProgress(100, {
                    indeterminate: false,
                    phase: "Starting",
                    message: "Starting the example runtime..."
                });

                appRoot.dataset.boot = "ready";
                shell.dataset.state = "ready";
                go.run(result.instance);
            } catch (error) {
                setError(error instanceof Error ? error.message : String(error));
            }
        })();
    </script>
</body>
</html>
`
}
