package pwa

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

type WasmReleaseFlags struct {
	Trimpath    bool   `json:"trimpath,omitempty"`
	LDFlags     string `json:"ldflags,omitempty"`
	GCFlags     string `json:"gcflags,omitempty"`
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
	BuildID   string                         `json:"buildId,omitempty"`
	Flags     WasmReleaseFlags               `json:"flags"`
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

type WasmRolloutConfig struct {
	Stable          WasmReleaseManifest
	Canary          WasmReleaseManifest
	CanaryPercent   int
	Salt            string
	CacheTTLSeconds int
	Rollback        bool
}

type WasmRolloutDecision struct {
	Cohort           string
	BuildID          string
	WasmURL          string
	SHA256           string
	ManifestRevision string
	CacheTTLSeconds  int
	RolledBack       bool
}

type VersionSkewRefreshInput struct {
	ClientBuildID      string
	ServerBuildID      string
	ReloadAlreadyTried bool
	StateSnapshotJSON  []byte
	VersionsCanMigrate bool
}

type VersionSkewRefreshDecision struct {
	Mismatch             bool
	ShouldReload         bool
	BypassCache          bool
	LoopGuarded          bool
	SnapshotBeforeReload []byte
	RestoreAfterReload   bool
	Reason               string
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
	parseNormalized.BuildID = strings.TrimSpace(parseNormalized.BuildID)
	parseNormalized.Flags.LDFlags = strings.TrimSpace(parseNormalized.Flags.LDFlags)
	parseNormalized.Flags.GCFlags = strings.TrimSpace(parseNormalized.Flags.GCFlags)
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
	parseParts = append(parseParts, parseNormalized.Package, parseNormalized.Profile, parseNormalized.GOOS+"/"+parseNormalized.GOARCH, parseNormalized.BuildID)
	for _, parseName2 := range parseKeys {
		parseArtifact := parseNormalized.Artifacts[parseName2]
		parseParts = append(parseParts, parseName2+":"+parseArtifact.Path+":"+parseArtifact.SHA256)
	}
	parseSum := sha256.Sum256([]byte(strings.Join(parseParts, "|")))
	return hex.EncodeToString(parseSum[:])
}

// ChooseWasmRollout deterministically assigns one client to the stable or
// canary wasm artifact. Rollback forces every client to stable immediately and
// caps the response TTL at one second so caches converge quickly.
func ChooseWasmRollout(parseConfig WasmRolloutConfig, parseClientID string) (WasmRolloutDecision, error) {
	parseConfig.Stable = parseConfig.Stable.Normalized()
	parseConfig.Canary = parseConfig.Canary.Normalized()
	if parseErr := parseConfig.Stable.Validate(); parseErr != nil {
		return WasmRolloutDecision{}, fmt.Errorf("stable manifest: %w", parseErr)
	}
	if parseConfig.CanaryPercent < 0 || parseConfig.CanaryPercent > 100 {
		return WasmRolloutDecision{}, fmt.Errorf("pwa rollout canary percent must be between 0 and 100")
	}
	parseTTL := parseConfig.CacheTTLSeconds
	if parseTTL < 0 {
		parseTTL = 0
	}
	if parseConfig.Rollback {
		if parseTTL == 0 || parseTTL > 1 {
			parseTTL = 1
		}
		return buildWasmRolloutDecision("stable", parseConfig.Stable, parseTTL, true), nil
	}
	isCanary := false
	if parseConfig.CanaryPercent > 0 {
		if parseErr := parseConfig.Canary.Validate(); parseErr != nil {
			return WasmRolloutDecision{}, fmt.Errorf("canary manifest: %w", parseErr)
		}
		isCanary = wasmRolloutBucket(parseClientID, parseConfig.Salt) < parseConfig.CanaryPercent
	}
	if isCanary {
		return buildWasmRolloutDecision("canary", parseConfig.Canary, parseTTL, false), nil
	}
	return buildWasmRolloutDecision("stable", parseConfig.Stable, parseTTL, false), nil
}

// EvaluateVersionSkewRefresh decides whether a stale wasm/client build should
// perform one cache-bypassing reload and carry a compatible state snapshot
// across that refresh.
func EvaluateVersionSkewRefresh(parseInput VersionSkewRefreshInput) VersionSkewRefreshDecision {
	parseClient := strings.TrimSpace(parseInput.ClientBuildID)
	parseServer := strings.TrimSpace(parseInput.ServerBuildID)
	if parseClient == "" || parseServer == "" || parseClient == parseServer {
		return VersionSkewRefreshDecision{Reason: "versions match"}
	}
	parseDecision := VersionSkewRefreshDecision{
		Mismatch: true,
		Reason:   "client build differs from server build",
	}
	if parseInput.ReloadAlreadyTried {
		parseDecision.LoopGuarded = true
		parseDecision.Reason = "version mismatch persists after one forced reload"
		return parseDecision
	}
	parseDecision.ShouldReload = true
	parseDecision.BypassCache = true
	parseDecision.SnapshotBeforeReload = append([]byte(nil), parseInput.StateSnapshotJSON...)
	parseDecision.RestoreAfterReload = parseInput.VersionsCanMigrate && len(parseDecision.SnapshotBeforeReload) > 0
	return parseDecision
}

func buildWasmRolloutDecision(parseCohort string, parseManifest WasmReleaseManifest, parseTTL int, isRollback bool) WasmRolloutDecision {
	parseWasm := parseManifest.Artifacts["wasm"]
	parseRevision := parseManifest.Revision()
	return WasmRolloutDecision{
		Cohort:           parseCohort,
		BuildID:          parseManifestBuildID(parseManifest),
		WasmURL:          parseWasm.Path,
		SHA256:           parseWasm.SHA256,
		ManifestRevision: parseRevision,
		CacheTTLSeconds:  parseTTL,
		RolledBack:       isRollback,
	}
}

func parseManifestBuildID(parseManifest WasmReleaseManifest) string {
	if parseManifest.BuildID != "" {
		return parseManifest.BuildID
	}
	return parseManifest.Revision()[:12]
}

func wasmRolloutBucket(parseClientID string, parseSalt string) int {
	parseClientID = strings.TrimSpace(parseClientID)
	if parseClientID == "" {
		parseClientID = "anonymous"
	}
	parseSum := sha256.Sum256([]byte(strings.TrimSpace(parseSalt) + "\x00" + parseClientID))
	return int(parseSum[0]) % 100
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
