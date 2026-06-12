package main

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/andybalholm/brotli"
)

type releaseManifestSnapshot struct {
	Package     string                           `json:"package"`
	Profile     string                           `json:"profile"`
	GOOS        string                           `json:"goos"`
	GOARCH      string                           `json:"goarch"`
	Artifacts   map[string]releaseArtifactRecord `json:"artifacts"`
	Attribution *releaseAttributionRecord        `json:"attribution,omitempty"`
}

type releasePackageAttributionSnapshot struct {
	Mode     string                     `json:"mode"`
	Packages []releasePackageSizeRecord `json:"packages"`
}

func releaseWriteDiffReport(parseCompareManifest string, parseManifestPath string, parseAttribution *releaseAttributionRecord, parseArtifacts map[string]releaseArtifactRecord, parseOutDir string) (*releaseDiffArtifactRecord, error) {
	if strings.TrimSpace(parseCompareManifest) == "" {
		return nil, nil
	}
	parseBaseline, parseErr := releaseReadManifestSnapshot(parseCompareManifest)
	if parseErr != nil {
		return nil, parseErr
	}
	parseArtifactChanges := releaseCompareArtifactRecords(parseBaseline.Artifacts, parseArtifacts)
	parseLikelyCulprits, parseErr := releaseComparePackageAttributionRecords(parseCompareManifest, parseBaseline.Attribution, parseOutDir, parseAttribution)
	if parseErr != nil {
		return nil, parseErr
	}
	parseFileName := "wasm-release-size-diff.json"
	parsePayload := map[string]any{
		"baselineManifestPath": baselinePathForJSON(parseCompareManifest),
		"currentManifestPath":  parseManifestPath,
		"artifactChanges":      parseArtifactChanges,
		"likelyCulprits":       parseLikelyCulprits,
	}
	parseEncoded, parseErr := releaseMarshalIndent(parsePayload, "", "  ")
	if parseErr != nil {
		return nil, fmt.Errorf("encode release diff report: %w", parseErr)
	}
	parseEncoded = append(parseEncoded, '\n')
	if parseErr2 := os.WriteFile(filepath.Join(parseOutDir, parseFileName), parseEncoded, 0644); parseErr2 != nil {
		return nil, fmt.Errorf("write release diff report: %w", parseErr2)
	}
	return &releaseDiffArtifactRecord{
		Path:                 parseFileName,
		BaselineManifestPath: parseCompareManifest,
		ArtifactChanges:      parseArtifactChanges,
		LikelyCulprits:       parseLikelyCulprits,
	}, nil
}

func baselinePathForJSON(parsePath string) string {
	return parsePath
}

func releaseReadManifestSnapshot(parsePath string) (releaseManifestSnapshot, error) {
	parseManifestBytes, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		return releaseManifestSnapshot{}, fmt.Errorf("read compare manifest: %w", parseErr)
	}
	var parseManifest releaseManifestSnapshot
	if parseErr2 := json.Unmarshal(parseManifestBytes, &parseManifest); parseErr2 != nil {
		return releaseManifestSnapshot{}, fmt.Errorf("parse compare manifest: %w", parseErr2)
	}
	if strings.TrimSpace(parseManifest.Package) == "" {
		return releaseManifestSnapshot{}, errors.New("compare manifest is missing package")
	}
	if parseManifest.GOOS != "js" || parseManifest.GOARCH != "wasm" {
		return releaseManifestSnapshot{}, errors.New("compare manifest must describe a js/wasm release")
	}
	if len(parseManifest.Artifacts) == 0 {
		return releaseManifestSnapshot{}, errors.New("compare manifest is missing artifacts")
	}
	return parseManifest, nil
}

func releaseCompareArtifactRecords(parseBaseline map[string]releaseArtifactRecord, parseCurrent map[string]releaseArtifactRecord) []releaseArtifactDiffRecord {
	parseKeys := map[string]struct{}{}
	for parseKey := range parseBaseline {
		parseKeys[parseKey] = struct{}{}
	}
	for parseKey2 := range parseCurrent {
		parseKeys[parseKey2] = struct{}{}
	}
	parseNames := make([]string, 0, len(parseKeys))
	for parseKey3 := range parseKeys {
		parseNames = append(parseNames, parseKey3)
	}
	sort.Strings(parseNames)
	parseChanges := make([]releaseArtifactDiffRecord, 0, len(parseNames))
	for _, parseName := range parseNames {
		parseBaselineArtifact, hasBaseline := parseBaseline[parseName]
		parseCurrentArtifact, hasCurrent := parseCurrent[parseName]
		parseRecord := releaseArtifactDiffRecord{Name: parseName}
		if hasBaseline {
			parseRecord.BaselinePath = parseBaselineArtifact.Path
			parseRecord.BaselineBytes = new(parseBaselineArtifact.Bytes)
		}
		if hasCurrent {
			parseRecord.CurrentPath = parseCurrentArtifact.Path
			parseRecord.CurrentBytes = new(parseCurrentArtifact.Bytes)
		}
		switch {
		case hasBaseline && hasCurrent:
			parseDelta := parseCurrentArtifact.Bytes - parseBaselineArtifact.Bytes
			parseRecord.DeltaBytes = new(parseDelta)
			parseRecord.DeltaPercent = releasePercentDeltaPointer(parseBaselineArtifact.Bytes, parseCurrentArtifact.Bytes)
			if parseDelta > 0 {
				parseRecord.Status = "grew"
			} else if parseDelta < 0 {
				parseRecord.Status = "shrank"
			} else {
				parseRecord.Status = "unchanged"
			}
		case hasCurrent:
			parseRecord.Status = "added"
		default:
			parseRecord.Status = "removed"
		}
		parseChanges = append(parseChanges, parseRecord)
	}
	return parseChanges
}

func releaseComparePackageAttributionRecords(parseBaselineManifestPath string, parseBaselineAttribution *releaseAttributionRecord, parseCurrentOutDir string, parseCurrentAttribution *releaseAttributionRecord) ([]releasePackageDiffRecord, error) {
	if parseBaselineAttribution == nil || parseCurrentAttribution == nil {
		return nil, nil
	}
	parseBaselinePackages, parseErr := releaseReadPackageAttributionSnapshot(filepath.Join(filepath.Dir(parseBaselineManifestPath), filepath.FromSlash(parseBaselineAttribution.Path)))
	if parseErr != nil {
		return nil, parseErr
	}
	parseCurrentPackages, parseErr := releaseReadPackageAttributionSnapshot(filepath.Join(parseCurrentOutDir, filepath.FromSlash(parseCurrentAttribution.Path)))
	if parseErr != nil {
		return nil, parseErr
	}
	parseBaselineMap := map[string]releasePackageSizeRecord{}
	for _, parseRecord := range parseBaselinePackages.Packages {
		parseBaselineMap[parseRecord.ImportPath] = parseRecord
	}
	parseCurrentMap := map[string]releasePackageSizeRecord{}
	for _, parseRecord2 := range parseCurrentPackages.Packages {
		parseCurrentMap[parseRecord2.ImportPath] = parseRecord2
	}
	parseKeys := map[string]struct{}{}
	for parseKey := range parseBaselineMap {
		parseKeys[parseKey] = struct{}{}
	}
	for parseKey2 := range parseCurrentMap {
		parseKeys[parseKey2] = struct{}{}
	}
	parseDeltas := make([]releasePackageDiffRecord, 0, len(parseKeys))
	for parseImportPath := range parseKeys {
		parseBaselineRecord, hasBaseline := parseBaselineMap[parseImportPath]
		parseCurrentRecord, hasCurrent := parseCurrentMap[parseImportPath]
		parseRecord3 := releasePackageDiffRecord{ImportPath: parseImportPath}
		if hasBaseline {
			parseRecord3.BaselineArchiveBytes = new(parseBaselineRecord.ArchiveBytes)
			parseRecord3.BaselineSourceBytes = new(parseBaselineRecord.SourceBytes)
		}
		if hasCurrent {
			parseRecord3.CurrentArchiveBytes = new(parseCurrentRecord.ArchiveBytes)
			parseRecord3.CurrentSourceBytes = new(parseCurrentRecord.SourceBytes)
		}
		switch {
		case hasBaseline && hasCurrent:
			parseArchiveDelta := parseCurrentRecord.ArchiveBytes - parseBaselineRecord.ArchiveBytes
			parseSourceDelta := parseCurrentRecord.SourceBytes - parseBaselineRecord.SourceBytes
			parseRecord3.ArchiveDeltaBytes = new(parseArchiveDelta)
			parseRecord3.ArchiveDeltaPercent = releasePercentDeltaPointer(parseBaselineRecord.ArchiveBytes, parseCurrentRecord.ArchiveBytes)
			parseRecord3.SourceDeltaBytes = new(parseSourceDelta)
			if parseArchiveDelta > 0 {
				parseRecord3.Status = "grew"
			} else if parseArchiveDelta < 0 {
				parseRecord3.Status = "shrank"
			} else if parseSourceDelta != 0 {
				parseRecord3.Status = "source-only-change"
			} else {
				parseRecord3.Status = "unchanged"
			}
		case hasCurrent:
			parseRecord3.ArchiveDeltaBytes = new(parseCurrentRecord.ArchiveBytes)
			parseRecord3.SourceDeltaBytes = new(parseCurrentRecord.SourceBytes)
			parseRecord3.Status = "added"
		default:
			parseRecord3.ArchiveDeltaBytes = new(-parseBaselineRecord.ArchiveBytes)
			parseRecord3.SourceDeltaBytes = new(-parseBaselineRecord.SourceBytes)
			parseRecord3.Status = "removed"
		}
		if parseRecord3.ArchiveDeltaBytes != nil && *parseRecord3.ArchiveDeltaBytes > 0 {
			parseDeltas = append(parseDeltas, parseRecord3)
		}
	}
	sort.Slice(parseDeltas, func(parseI int, parseJ int) bool {
		parseLeft := derefInt64(parseDeltas[parseI].ArchiveDeltaBytes)
		parseRight := derefInt64(parseDeltas[parseJ].ArchiveDeltaBytes)
		if parseLeft != parseRight {
			return parseLeft > parseRight
		}
		parseLeftSource := derefInt64(parseDeltas[parseI].SourceDeltaBytes)
		parseRightSource := derefInt64(parseDeltas[parseJ].SourceDeltaBytes)
		if parseLeftSource != parseRightSource {
			return parseLeftSource > parseRightSource
		}
		return parseDeltas[parseI].ImportPath < parseDeltas[parseJ].ImportPath
	})
	if len(parseDeltas) > 10 {
		parseDeltas = parseDeltas[:10]
	}
	if len(parseDeltas) == 0 {
		return nil, nil
	}
	return parseDeltas, nil
}

func releaseReadPackageAttributionSnapshot(parsePath string) (releasePackageAttributionSnapshot, error) {
	parseData, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		return releasePackageAttributionSnapshot{}, fmt.Errorf("read package attribution artifact: %w", parseErr)
	}
	var parseSnapshot releasePackageAttributionSnapshot
	if parseErr2 := json.Unmarshal(parseData, &parseSnapshot); parseErr2 != nil {
		return releasePackageAttributionSnapshot{}, fmt.Errorf("parse package attribution artifact: %w", parseErr2)
	}
	return parseSnapshot, nil
}

func releasePercentDeltaPointer(parseBaseline int64, parseCurrent int64) *float64 {
	if parseBaseline == 0 {
		return nil
	}
	parseDelta := float64(parseCurrent-parseBaseline) / float64(parseBaseline) * 100
	return &parseDelta
}

func derefInt64(parseValue *int64) int64 {
	if parseValue == nil {
		return 0
	}
	return *parseValue
}

func releaseArtifactRecordForPath(parseBaseDir string, parseArtifactPath string) (releaseArtifactRecord, error) {
	parseArtifactBytes, parseErr := os.ReadFile(parseArtifactPath)
	if parseErr != nil {
		return releaseArtifactRecord{}, fmt.Errorf("read release artifact: %w", parseErr)
	}
	parseArtifactInfo, parseErr := os.Stat(parseArtifactPath)
	if parseErr != nil {
		return releaseArtifactRecord{}, fmt.Errorf("inspect release artifact: %w", parseErr)
	}
	parseRelPath, parseErr := filepath.Rel(parseBaseDir, parseArtifactPath)
	if parseErr != nil {
		return releaseArtifactRecord{}, fmt.Errorf("resolve release artifact path: %w", parseErr)
	}
	parseHash := sha256.Sum256(parseArtifactBytes)
	return releaseArtifactRecord{
		Path:   filepath.ToSlash(parseRelPath),
		Bytes:  parseArtifactInfo.Size(),
		SHA256: fmt.Sprintf("%x", parseHash[:]),
	}, nil
}

func writeGzipSidecar(parseSourcePath string, parseTargetPath string) error {
	parseInputBytes, parseErr := os.ReadFile(parseSourcePath)
	if parseErr != nil {
		return fmt.Errorf("read source artifact for gzip: %w", parseErr)
	}
	parseOutputFile, parseErr := os.Create(parseTargetPath)
	if parseErr != nil {
		return fmt.Errorf("create gzip sidecar: %w", parseErr)
	}
	defer parseOutputFile.Close()
	parseGzipWriter, parseErr := gzip.NewWriterLevel(parseOutputFile, gzip.BestCompression)
	if parseErr != nil {
		return fmt.Errorf("create gzip writer: %w", parseErr)
	}
	if _, parseErr2 := parseGzipWriter.Write(parseInputBytes); parseErr2 != nil {
		parseGzipWriter.Close()
		return fmt.Errorf("write gzip sidecar: %w", parseErr2)
	}
	if parseErr3 := parseGzipWriter.Close(); parseErr3 != nil {
		return fmt.Errorf("finalize gzip sidecar: %w", parseErr3)
	}
	return nil
}

func writeBrotliSidecar(parseSourcePath string, parseTargetPath string) error {
	parseInputBytes, parseErr := os.ReadFile(parseSourcePath)
	if parseErr != nil {
		return fmt.Errorf("read source artifact for brotli: %w", parseErr)
	}
	parseOutputFile, parseErr := os.Create(parseTargetPath)
	if parseErr != nil {
		return fmt.Errorf("create brotli sidecar: %w", parseErr)
	}
	defer parseOutputFile.Close()
	parseBrotliWriter := brotli.NewWriterLevel(parseOutputFile, brotli.BestCompression)
	if _, parseErr2 := parseBrotliWriter.Write(parseInputBytes); parseErr2 != nil {
		parseBrotliWriter.Close()
		return fmt.Errorf("write brotli sidecar: %w", parseErr2)
	}
	if parseErr3 := parseBrotliWriter.Close(); parseErr3 != nil {
		return fmt.Errorf("finalize brotli sidecar: %w", parseErr3)
	}
	return nil
}

func loadReleaseBudgets(parsePath string) (map[string]int64, error) {
	parseContent, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		return nil, fmt.Errorf("read budgets file: %w", parseErr)
	}
	parseRaw := map[string]any{}
	if parseErr2 := json.Unmarshal(parseContent, &parseRaw); parseErr2 != nil {
		return nil, fmt.Errorf("parse budgets file: %w", parseErr2)
	}
	parseBudgets := map[string]int64{}
	for parseKey, parseValue := range parseRaw {
		parseNumber, parseOk := parseValue.(float64)
		if !parseOk {
			return nil, fmt.Errorf("budget %q must be numeric", parseKey)
		}
		parseBudgets[parseKey] = int64(parseNumber)
	}
	return parseBudgets, nil
}

func assertReleaseBudgets(parseBudgets map[string]int64, parseArtifacts map[string]releaseArtifactRecord) error {
	parseChecks := []struct {
		budgetKey string
		artifact  string
		label     string
	}{
		{budgetKey: "raw_bytes", artifact: "wasm", label: "raw wasm"},
		{budgetKey: "gzip_bytes", artifact: "gzip", label: "gzip sidecar"},
		{budgetKey: "brotli_bytes", artifact: "brotli", label: "brotli sidecar"},
	}
	for _, parseCheck := range parseChecks {
		parseLimit, parseOk := parseBudgets[parseCheck.budgetKey]
		if !parseOk {
			continue
		}
		parseArtifact, parseOk := parseArtifacts[parseCheck.artifact]
		if !parseOk {
			continue
		}
		if parseArtifact.Bytes > parseLimit {
			return fmt.Errorf("artifact budget exceeded for %s: %d bytes > %d bytes", parseCheck.label, parseArtifact.Bytes, parseLimit)
		}
	}
	return nil
}
