package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"

	chatpb "github.com/monstercameron/GoWebComponents/v5/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const parseCatalogBundleVersion = "2026.03.28"

var parseCatalogBootstrapDefaultNamespaces = []string{"chat", "marketing"}

// GetCatalogBootstrap returns one typed locale bootstrap payload for one namespace set.
func (parseS *chatServer) GetCatalogBootstrap(parseCtx context.Context, parseReq *chatpb.GetCatalogBootstrapRequest) (*chatpb.GetCatalogBootstrapResponse, error) {
	_ = parseCtx
	parseLocale := "en"
	parseNamespaces := make([]string, 0)
	parseKnownVersion := ""
	parseKnownHash := ""
	if parseReq != nil {
		parseLocale = parseNormalizeCatalogLocale(parseReq.GetLocale())
		parseNamespaces = parseReq.GetNamespaces()
		parseKnownVersion = strings.TrimSpace(parseReq.GetKnownVersion())
		parseKnownHash = strings.TrimSpace(parseReq.GetKnownHash())
	}
	parseLoader := parseBuildCatalogLoader(nil, parseCatalogBundleVersion)
	parseResolvedNamespaces := parseBuildCatalogBootstrapNamespaces(parseNamespaces)
	parseCatalogRows := make([]*chatpb.CatalogNamespacePayload, 0, len(parseResolvedNamespaces))
	parseFallbackLocale := parseLocale
	for _, parseNamespace := range parseResolvedNamespaces {
		parsePayload, parseErr := parseLoader.parseLoadCatalogNamespace(parseNamespace, parseLocale)
		if parseErr != nil {
			return nil, status.Errorf(codes.Internal, "load catalog namespace %q: %v", parseNamespace, parseErr)
		}
		if parsePayload == nil {
			continue
		}
		if strings.TrimSpace(parsePayload.GetFallbackLocale()) != "" {
			parseFallbackLocale = parsePayload.GetFallbackLocale()
		}
		parseCatalogRows = append(parseCatalogRows, parsePayload)
	}
	parseSortCatalogNamespacePayloads(parseCatalogRows)
	parseBundleHash := parseBuildCatalogBootstrapHash(parseCatalogRows)
	parseResponse := &chatpb.GetCatalogBootstrapResponse{
		Locale:         parseLocale,
		FallbackLocale: parseFallbackLocale,
		BundleVersion:  parseCatalogBundleVersion,
		BundleHash:     parseBundleHash,
		Catalogs:       parseCatalogRows,
	}
	if parseKnownVersion == parseCatalogBundleVersion && parseKnownHash != "" && parseKnownHash == parseBundleHash {
		parseResponse.IsNotModified = true
		parseResponse.Catalogs = nil
	}
	return parseResponse, nil
}

// GetCatalogNamespace returns one typed locale namespace payload and optional not-modified cache signal.
func (parseS *chatServer) GetCatalogNamespace(parseCtx context.Context, parseReq *chatpb.GetCatalogNamespaceRequest) (*chatpb.GetCatalogNamespaceResponse, error) {
	_ = parseCtx
	parseNamespace := ""
	parseLocale := "en"
	parseKnownVersion := ""
	parseKnownHash := ""
	if parseReq != nil {
		parseNamespace = strings.TrimSpace(parseReq.GetNamespace())
		parseLocale = parseNormalizeCatalogLocale(parseReq.GetLocale())
		parseKnownVersion = strings.TrimSpace(parseReq.GetKnownVersion())
		parseKnownHash = strings.TrimSpace(parseReq.GetKnownHash())
	}
	if parseNamespace == "" {
		return nil, status.Error(codes.InvalidArgument, "namespace is required")
	}
	parseLoader := parseBuildCatalogLoader(nil, parseCatalogBundleVersion)
	parsePayload, parseErr := parseLoader.parseLoadCatalogNamespace(parseNamespace, parseLocale)
	if parseErr != nil {
		return nil, status.Errorf(codes.Internal, "load catalog namespace %q: %v", parseNamespace, parseErr)
	}
	parseResponse := &chatpb.GetCatalogNamespaceResponse{
		Catalog: parsePayload,
	}
	if parsePayload != nil && parseKnownVersion == parsePayload.GetVersion() && parseKnownHash != "" && parseKnownHash == parsePayload.GetContentHash() {
		parseResponse.IsNotModified = true
		parseResponse.Catalog = nil
	}
	return parseResponse, nil
}

// parseBuildCatalogBootstrapNamespaces builds one stable deduplicated namespace list for bootstrap fetches.
func parseBuildCatalogBootstrapNamespaces(parseNamespaces []string) []string {
	if len(parseNamespaces) == 0 {
		return append([]string{}, parseCatalogBootstrapDefaultNamespaces...)
	}
	parseNamespaceSeen := make(map[string]struct{}, len(parseNamespaces))
	parseResolvedNamespaces := make([]string, 0, len(parseNamespaces))
	for _, parseNamespace := range parseNamespaces {
		parseNamespace = strings.TrimSpace(parseNamespace)
		if parseNamespace == "" {
			continue
		}
		if _, hasParseNamespace := parseNamespaceSeen[parseNamespace]; hasParseNamespace {
			continue
		}
		parseNamespaceSeen[parseNamespace] = struct{}{}
		parseResolvedNamespaces = append(parseResolvedNamespaces, parseNamespace)
	}
	if len(parseResolvedNamespaces) == 0 {
		return append([]string{}, parseCatalogBootstrapDefaultNamespaces...)
	}
	return parseResolvedNamespaces
}

// parseSortCatalogNamespacePayloads sorts one catalog namespace payload slice by namespace then locale.
func parseSortCatalogNamespacePayloads(parseRows []*chatpb.CatalogNamespacePayload) {
	sort.Slice(parseRows, func(parseIndexA int, parseIndexB int) bool {
		parseRowA := parseRows[parseIndexA]
		parseRowB := parseRows[parseIndexB]
		parseNamespaceA := ""
		parseNamespaceB := ""
		parseLocaleA := ""
		parseLocaleB := ""
		if parseRowA != nil {
			parseNamespaceA = parseRowA.GetNamespace()
			parseLocaleA = parseRowA.GetLocale()
		}
		if parseRowB != nil {
			parseNamespaceB = parseRowB.GetNamespace()
			parseLocaleB = parseRowB.GetLocale()
		}
		if parseNamespaceA == parseNamespaceB {
			return parseLocaleA < parseLocaleB
		}
		return parseNamespaceA < parseNamespaceB
	})
}

// parseBuildCatalogBootstrapHash builds one deterministic hash from namespace payload hash rows.
func parseBuildCatalogBootstrapHash(parseRows []*chatpb.CatalogNamespacePayload) string {
	var parseHashInput strings.Builder
	parseHashInput.WriteString(parseCatalogBundleVersion)
	for _, parseRow := range parseRows {
		if parseRow == nil {
			continue
		}
		parseHashInput.WriteString("|" + parseRow.GetNamespace() + ":" + parseRow.GetLocale() + ":" + parseRow.GetContentHash())
	}
	parseDigest := sha256.Sum256([]byte(parseHashInput.String()))
	return hex.EncodeToString(parseDigest[:])
}
