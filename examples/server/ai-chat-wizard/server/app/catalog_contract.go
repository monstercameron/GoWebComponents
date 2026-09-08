package app

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"

	chatpb "github.com/monstercameron/GoWebComponents/v6/examples/server/ai-chat-wizard/proto"
)

type parseCatalogSourceWrite struct {
	SourceLayer     string
	SourceID        string
	SourceVersion   string
	SourceUpdatedAt string
}

// parseBuildCatalogNamespacePayload builds one typed server-owned catalog contract payload.
func parseBuildCatalogNamespacePayload(parseNamespace string, parseLocale string, parseFallbackLocale string, parseVersion string, parseSourceWrite parseCatalogSourceWrite, parseMessages map[string]string) *chatpb.CatalogNamespacePayload {
	parseNamespace = strings.TrimSpace(parseNamespace)
	parseLocale = parseNormalizeCatalogLocale(parseLocale)
	parseFallbackLocale = parseNormalizeCatalogFallbackLocale(parseFallbackLocale, parseLocale)
	parseVersion = parseNormalizeCatalogVersion(parseVersion)
	parseSource := parseBuildCatalogSourceMetadata(parseSourceWrite)
	parseMessageRows := parseBuildCatalogMessagePayloads(parseMessages)
	return &chatpb.CatalogNamespacePayload{
		Namespace:      parseNamespace,
		Locale:         parseLocale,
		FallbackLocale: parseFallbackLocale,
		Version:        parseVersion,
		ContentHash:    parseBuildCatalogContentHash(parseNamespace, parseLocale, parseFallbackLocale, parseVersion, parseMessageRows),
		Source:         parseSource,
		Messages:       parseMessageRows,
	}
}

// parseBuildCatalogSourceMetadata builds one typed catalog source-layer metadata payload.
func parseBuildCatalogSourceMetadata(parseWrite parseCatalogSourceWrite) *chatpb.CatalogSourceMetadata {
	parseSourceLayer := parseNormalizeCatalogSourceLayer(parseWrite.SourceLayer)
	return &chatpb.CatalogSourceMetadata{
		SourceLayer:     parseSourceLayer,
		SourceId:        strings.TrimSpace(parseWrite.SourceID),
		SourceVersion:   strings.TrimSpace(parseWrite.SourceVersion),
		SourceUpdatedAt: strings.TrimSpace(parseWrite.SourceUpdatedAt),
	}
}

// parseBuildCatalogMessagePayloads builds one stable message payload slice from one key-value map.
func parseBuildCatalogMessagePayloads(parseMessages map[string]string) []*chatpb.CatalogMessagePayload {
	if len(parseMessages) == 0 {
		return make([]*chatpb.CatalogMessagePayload, 0)
	}
	parseMessageByKey := make(map[string]string, len(parseMessages))
	parseMessageKeys := make([]string, 0, len(parseMessages))
	for parseMessageKey, parseMessageValue := range parseMessages {
		parseMessageKey = strings.TrimSpace(parseMessageKey)
		if parseMessageKey == "" {
			continue
		}
		parseMessageByKey[parseMessageKey] = strings.TrimSpace(parseMessageValue)
		parseMessageKeys = append(parseMessageKeys, parseMessageKey)
	}
	sort.Strings(parseMessageKeys)
	parseMessageRows := make([]*chatpb.CatalogMessagePayload, 0, len(parseMessageKeys))
	for _, parseMessageKey := range parseMessageKeys {
		parseMessageRows = append(parseMessageRows, &chatpb.CatalogMessagePayload{
			MessageKey:   parseMessageKey,
			MessageValue: parseMessageByKey[parseMessageKey],
		})
	}
	return parseMessageRows
}

// parseBuildCatalogContentHash builds one deterministic content hash for one catalog namespace payload.
func parseBuildCatalogContentHash(parseNamespace string, parseLocale string, parseFallbackLocale string, parseVersion string, parseMessages []*chatpb.CatalogMessagePayload) string {
	var parseHashInput strings.Builder
	parseHashInput.WriteString(parseNamespace + "|" + parseLocale + "|" + parseFallbackLocale + "|" + parseVersion)
	for _, parseMessage := range parseMessages {
		if parseMessage == nil {
			continue
		}
		parseHashInput.WriteString("|" + parseMessage.GetMessageKey() + "=" + parseMessage.GetMessageValue())
	}
	parseDigest := sha256.Sum256([]byte(parseHashInput.String()))
	return hex.EncodeToString(parseDigest[:])
}

// parseNormalizeCatalogLocale normalizes one locale id into one lower-case BCP-47 style token.
func parseNormalizeCatalogLocale(parseLocale string) string {
	parseLocale = strings.TrimSpace(parseLocale)
	parseLocale = strings.ReplaceAll(parseLocale, "_", "-")
	parseLocale = strings.ToLower(parseLocale)
	if parseLocale == "" {
		return "en"
	}
	return parseLocale
}

// parseNormalizeCatalogFallbackLocale normalizes one fallback locale with one locale default.
func parseNormalizeCatalogFallbackLocale(parseFallbackLocale string, parseLocale string) string {
	if strings.TrimSpace(parseFallbackLocale) == "" {
		return parseNormalizeCatalogLocale(parseLocale)
	}
	return parseNormalizeCatalogLocale(parseFallbackLocale)
}

// parseNormalizeCatalogVersion normalizes one catalog version token.
func parseNormalizeCatalogVersion(parseVersion string) string {
	parseVersion = strings.TrimSpace(parseVersion)
	if parseVersion == "" {
		return "v1"
	}
	return parseVersion
}
