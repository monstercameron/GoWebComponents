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

// ParseWasmReleaseManifestJSON parses and validates a WasmReleaseManifest from JSON bytes.
func ParseWasmReleaseManifestJSON(parseData []byte) (WasmReleaseManifest, error) {
	var parseManifest WasmReleaseManifest
	if parseErr := json.Unmarshal(parseData, &parseManifest); parseErr != nil {
		return WasmReleaseManifest{}, parseErr
	}
	parseManifest = parseManifest.Normalized()
	if parseErr2 := parseManifest.Validate(); parseErr2 != nil {
		return WasmReleaseManifest{}, parseErr2
	}
	return parseManifest, nil
}

func (parseM WasmReleaseManifest) Normalized() WasmReleaseManifest {
	parseNormalized := parseM
	parseNormalized.Package = strings.TrimSpace(parseNormalized.Package)
	parseNormalized.Profile = strings.TrimSpace(parseNormalized.Profile)
	parseNormalized.GOOS = strings.TrimSpace(parseNormalized.GOOS)
	parseNormalized.GOARCH = strings.TrimSpace(parseNormalized.GOARCH)
	parseNormalized.Flags.LDFlags = strings.TrimSpace(parseNormalized.Flags.LDFlags)
	parseNormalized.Flags.BuildVCS = strings.TrimSpace(parseNormalized.Flags.BuildVCS)
	if len(parseNormalized.Artifacts) == 0 {
		parseNormalized.Artifacts = nil
		return parseNormalized
	}
	parseCopyArtifacts := make(map[string]WasmReleaseArtifact, len(parseNormalized.Artifacts))
	for parseName, parseArtifact := range parseNormalized.Artifacts {
		parseName = strings.TrimSpace(parseName)
		if parseName == "" {
			continue
		}
		parseArtifact.Path = strings.TrimSpace(parseArtifact.Path)
		parseArtifact.SHA256 = strings.ToLower(strings.TrimSpace(parseArtifact.SHA256))
		parseCopyArtifacts[parseName] = parseArtifact
	}
	if len(parseCopyArtifacts) == 0 {
		parseNormalized.Artifacts = nil
		return parseNormalized
	}
	parseNormalized.Artifacts = parseCopyArtifacts
	return parseNormalized
}

func (parseM WasmReleaseManifest) Validate() error {
	parseNormalized := parseM.Normalized()
	if parseNormalized.Package == "" {
		return errors.New("pwa wasm release manifest requires a non-empty package")
	}
	if parseNormalized.GOOS != "js" {
		return errors.New("pwa wasm release manifest requires goos=js")
	}
	if parseNormalized.GOARCH != "wasm" {
		return errors.New("pwa wasm release manifest requires goarch=wasm")
	}
	parseWasm, parseOk := parseNormalized.Artifacts["wasm"]
	if !parseOk {
		return errors.New("pwa wasm release manifest requires a wasm artifact")
	}
	if parseWasm.Path == "" {
		return errors.New("pwa wasm release manifest requires wasm.path")
	}
	if parseWasm.SHA256 == "" {
		return errors.New("pwa wasm release manifest requires wasm.sha256")
	}
	return nil
}

func (parseM WasmReleaseManifest) Revision() string {
	parseNormalized := parseM.Normalized()
	parseKeys := make([]string, 0, len(parseNormalized.Artifacts))
	for parseName := range parseNormalized.Artifacts {
		parseKeys = append(parseKeys, parseName)
	}
	sort.Strings(parseKeys)
	parseParts := make([]string, 0, len(parseKeys)+3)
	parseParts = append(parseParts, parseNormalized.Package, parseNormalized.Profile, parseNormalized.GOOS+"/"+parseNormalized.GOARCH)
	for _, parseName2 := range parseKeys {
		parseArtifact := parseNormalized.Artifacts[parseName2]
		parseParts = append(parseParts, parseName2+":"+parseArtifact.Path+":"+parseArtifact.SHA256)
	}
	parseSum := sha256.Sum256([]byte(strings.Join(parseParts, "|")))
	return hex.EncodeToString(parseSum[:])
}

// BuildServiceWorkerAssetPlan builds a ServiceWorkerAssetPlan from a validated WasmReleaseManifest.
func BuildServiceWorkerAssetPlan(parseManifest WasmReleaseManifest, parseOptions ...ServiceWorkerAssetPlanOptions) (ServiceWorkerAssetPlan, error) {
	parseNormalized := parseManifest.Normalized()
	if parseErr := parseNormalized.Validate(); parseErr != nil {
		return ServiceWorkerAssetPlan{}, parseErr
	}
	parseResolved := ServiceWorkerAssetPlanOptions{}
	if len(parseOptions) > 0 {
		parseResolved = parseOptions[0]
	}
	parseResolved.BaseURL = strings.TrimSpace(parseResolved.BaseURL)
	parseResolved.CachePrefix = strings.TrimSpace(parseResolved.CachePrefix)
	if parseResolved.CachePrefix == "" {
		parseResolved.CachePrefix = "gwc-release"
	}
	parseRevision := parseNormalized.Revision()
	parseWasmURL := resolveServiceWorkerAssetURL(parseResolved.BaseURL, parseNormalized.Artifacts["wasm"].Path)
	parseImmutableURLs := normalizeServiceWorkerURLs(parseResolved.BaseURL, parseResolved.ImmutableURLs)
	parseShellURLs := normalizeServiceWorkerURLs(parseResolved.BaseURL, parseResolved.ShellURLs)
	parsePrecacheURLs := dedupeServiceWorkerURLs(append([]string{parseWasmURL}, append(parseShellURLs, parseImmutableURLs...)...))
	return ServiceWorkerAssetPlan{
		CacheName:        parseResolved.CachePrefix + "-" + parseRevision[:12],
		ManifestRevision: parseRevision,
		WasmURL:          parseWasmURL,
		ImmutableURLs:    parseImmutableURLs,
		ShellURLs:        parseShellURLs,
		PrecacheURLs:     parsePrecacheURLs,
	}, nil
}

func normalizeServiceWorkerURLs(parseBaseURL string, parseUrls []string) []string {
	if len(parseUrls) == 0 {
		return nil
	}
	parseResolved := make([]string, 0, len(parseUrls))
	for _, parseUrl := range parseUrls {
		parseUrl = resolveServiceWorkerAssetURL(parseBaseURL, parseUrl)
		if parseUrl != "" {
			parseResolved = append(parseResolved, parseUrl)
		}
	}
	if len(parseResolved) == 0 {
		return nil
	}
	return dedupeServiceWorkerURLs(parseResolved)
}

func resolveServiceWorkerAssetURL(parseBaseURL string, parseAssetPath string) string {
	parseAssetPath = strings.TrimSpace(parseAssetPath)
	if parseAssetPath == "" {
		return ""
	}
	if parseBaseURL == "" {
		return parseAssetPath
	}
	return strings.TrimRight(parseBaseURL, "/") + "/" + strings.TrimLeft(parseAssetPath, "/")
}

func dedupeServiceWorkerURLs(parseUrls []string) []string {
	if len(parseUrls) == 0 {
		return nil
	}
	parseSeen := map[string]bool{}
	parseResolved := make([]string, 0, len(parseUrls))
	for _, parseUrl := range parseUrls {
		parseUrl = strings.TrimSpace(parseUrl)
		if parseUrl == "" || parseSeen[parseUrl] {
			continue
		}
		parseSeen[parseUrl] = true
		parseResolved = append(parseResolved, parseUrl)
	}
	if len(parseResolved) == 0 {
		return nil
	}
	return parseResolved
}
