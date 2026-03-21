package pwa

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"
)

type WasmReleaseFlags struct {
	Trimpath    bool   `json:"trimpath,omitempty"`
	LDFlags     string `json:"ldflags,omitempty"`
	BuildVCS    string `json:"buildvcs,omitempty"`
	Compression bool   `json:"compression,omitempty"`
}

type WasmReleaseArtifact struct {
	Path   string `json:"path,omitempty"`
	Bytes  int64  `json:"bytes,omitempty"`
	SHA256 string `json:"sha256,omitempty"`
}

type WasmReleaseManifest struct {
	Package   string                         `json:"package,omitempty"`
	Profile   string                         `json:"profile,omitempty"`
	GOOS      string                         `json:"goos,omitempty"`
	GOARCH    string                         `json:"goarch,omitempty"`
	Flags     WasmReleaseFlags               `json:"flags,omitempty"`
	Artifacts map[string]WasmReleaseArtifact `json:"artifacts,omitempty"`
}

type ServiceWorkerAssetPlanOptions struct {
	BaseURL       string
	CachePrefix   string
	ImmutableURLs []string
	ShellURLs     []string
}

type ServiceWorkerAssetPlan struct {
	CacheName        string
	ManifestRevision string
	WasmURL          string
	ImmutableURLs    []string
	ShellURLs        []string
	PrecacheURLs     []string
}

func ParseWasmReleaseManifestJSON(data []byte) (WasmReleaseManifest, error) {
	var manifest WasmReleaseManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return WasmReleaseManifest{}, err
	}
	manifest = manifest.Normalized()
	if err := manifest.Validate(); err != nil {
		return WasmReleaseManifest{}, err
	}
	return manifest, nil
}

func (m WasmReleaseManifest) Normalized() WasmReleaseManifest {
	normalized := m
	normalized.Package = strings.TrimSpace(normalized.Package)
	normalized.Profile = strings.TrimSpace(normalized.Profile)
	normalized.GOOS = strings.TrimSpace(normalized.GOOS)
	normalized.GOARCH = strings.TrimSpace(normalized.GOARCH)
	normalized.Flags.LDFlags = strings.TrimSpace(normalized.Flags.LDFlags)
	normalized.Flags.BuildVCS = strings.TrimSpace(normalized.Flags.BuildVCS)
	if len(normalized.Artifacts) == 0 {
		normalized.Artifacts = nil
		return normalized
	}
	copyArtifacts := make(map[string]WasmReleaseArtifact, len(normalized.Artifacts))
	for name, artifact := range normalized.Artifacts {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		artifact.Path = strings.TrimSpace(artifact.Path)
		artifact.SHA256 = strings.ToLower(strings.TrimSpace(artifact.SHA256))
		copyArtifacts[name] = artifact
	}
	if len(copyArtifacts) == 0 {
		normalized.Artifacts = nil
		return normalized
	}
	normalized.Artifacts = copyArtifacts
	return normalized
}

func (m WasmReleaseManifest) Validate() error {
	normalized := m.Normalized()
	if normalized.Package == "" {
		return errors.New("pwa wasm release manifest requires a non-empty package")
	}
	if normalized.GOOS != "js" {
		return errors.New("pwa wasm release manifest requires goos=js")
	}
	if normalized.GOARCH != "wasm" {
		return errors.New("pwa wasm release manifest requires goarch=wasm")
	}
	wasm, ok := normalized.Artifacts["wasm"]
	if !ok {
		return errors.New("pwa wasm release manifest requires a wasm artifact")
	}
	if wasm.Path == "" {
		return errors.New("pwa wasm release manifest requires wasm.path")
	}
	if wasm.SHA256 == "" {
		return errors.New("pwa wasm release manifest requires wasm.sha256")
	}
	return nil
}

func (m WasmReleaseManifest) Revision() string {
	normalized := m.Normalized()
	keys := make([]string, 0, len(normalized.Artifacts))
	for name := range normalized.Artifacts {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys)+3)
	parts = append(parts, normalized.Package, normalized.Profile, normalized.GOOS+"/"+normalized.GOARCH)
	for _, name := range keys {
		artifact := normalized.Artifacts[name]
		parts = append(parts, name+":"+artifact.Path+":"+artifact.SHA256)
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])
}

func BuildServiceWorkerAssetPlan(manifest WasmReleaseManifest, options ...ServiceWorkerAssetPlanOptions) (ServiceWorkerAssetPlan, error) {
	normalized := manifest.Normalized()
	if err := normalized.Validate(); err != nil {
		return ServiceWorkerAssetPlan{}, err
	}
	resolved := ServiceWorkerAssetPlanOptions{}
	if len(options) > 0 {
		resolved = options[0]
	}
	resolved.BaseURL = strings.TrimSpace(resolved.BaseURL)
	resolved.CachePrefix = strings.TrimSpace(resolved.CachePrefix)
	if resolved.CachePrefix == "" {
		resolved.CachePrefix = "gwc-release"
	}
	revision := normalized.Revision()
	wasmURL := resolveServiceWorkerAssetURL(resolved.BaseURL, normalized.Artifacts["wasm"].Path)
	immutableURLs := normalizeServiceWorkerURLs(resolved.BaseURL, resolved.ImmutableURLs)
	shellURLs := normalizeServiceWorkerURLs(resolved.BaseURL, resolved.ShellURLs)
	precacheURLs := dedupeServiceWorkerURLs(append([]string{wasmURL}, append(shellURLs, immutableURLs...)...))
	return ServiceWorkerAssetPlan{
		CacheName:        resolved.CachePrefix + "-" + revision[:12],
		ManifestRevision: revision,
		WasmURL:          wasmURL,
		ImmutableURLs:    immutableURLs,
		ShellURLs:        shellURLs,
		PrecacheURLs:     precacheURLs,
	}, nil
}

func normalizeServiceWorkerURLs(baseURL string, urls []string) []string {
	if len(urls) == 0 {
		return nil
	}
	resolved := make([]string, 0, len(urls))
	for _, url := range urls {
		url = resolveServiceWorkerAssetURL(baseURL, url)
		if url != "" {
			resolved = append(resolved, url)
		}
	}
	if len(resolved) == 0 {
		return nil
	}
	return dedupeServiceWorkerURLs(resolved)
}

func resolveServiceWorkerAssetURL(baseURL string, assetPath string) string {
	assetPath = strings.TrimSpace(assetPath)
	if assetPath == "" {
		return ""
	}
	if baseURL == "" {
		return assetPath
	}
	return strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(assetPath, "/")
}

func dedupeServiceWorkerURLs(urls []string) []string {
	if len(urls) == 0 {
		return nil
	}
	seen := map[string]bool{}
	resolved := make([]string, 0, len(urls))
	for _, url := range urls {
		url = strings.TrimSpace(url)
		if url == "" || seen[url] {
			continue
		}
		seen[url] = true
		resolved = append(resolved, url)
	}
	if len(resolved) == 0 {
		return nil
	}
	return resolved
}
