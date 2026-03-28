package app

import (
	"context"
	"sort"
	"strings"
	"unicode"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	parseCatalogSourceLayerGoEmbed            = "go_embed"
	parseCatalogSourceLayerSiteOverride       = "site_override"
	parseCatalogSourceLayerEnvironmentOverride = "environment_override"
	parseCatalogSourceLayerTenantOverride     = "tenant_override"
	parseCatalogSourceLayerExperimentOverride = "experiment_override"
	parseCatalogSourceLayerCampaignOverride   = "campaign_override"
)

// parseNormalizeCatalogSourceLayer normalizes one catalog source layer token.
func parseNormalizeCatalogSourceLayer(parseSourceLayer string) string {
	parseSourceLayer = strings.TrimSpace(strings.ToLower(parseSourceLayer))
	parseSourceLayer = strings.ReplaceAll(parseSourceLayer, "-", "_")
	if parseSourceLayer == "" {
		return parseCatalogSourceLayerGoEmbed
	}
	return parseSourceLayer
}

// parseValidateCatalogMessageTemplateSet validates placeholder syntax for one catalog message map.
func parseValidateCatalogMessageTemplateSet(parseMessages map[string]string) error {
	parseMessageKeys := make([]string, 0, len(parseMessages))
	for parseMessageKey := range parseMessages {
		parseMessageKeys = append(parseMessageKeys, parseMessageKey)
	}
	sort.Strings(parseMessageKeys)
	for _, parseMessageKey := range parseMessageKeys {
		parseTrimmedMessageKey := strings.TrimSpace(parseMessageKey)
		if parseTrimmedMessageKey == "" {
			return status.Error(codes.InvalidArgument, "catalog message key is required")
		}
		if _, parseErr := parseExtractCatalogMessageTemplateSet(parseMessages[parseMessageKey]); parseErr != nil {
			return status.Errorf(codes.InvalidArgument, "catalog message %q has malformed placeholders: %v", parseTrimmedMessageKey, parseErr)
		}
	}
	return nil
}

// parseValidateCatalogMessageTemplateParity validates candidate placeholders against one base-layer message map.
func parseValidateCatalogMessageTemplateParity(parseBaseMessages map[string]string, parseCandidateMessages map[string]string) error {
	if len(parseCandidateMessages) == 0 {
		return nil
	}
	parseNormalizedBaseMessages := make(map[string]string, len(parseBaseMessages))
	for parseMessageKey, parseMessageValue := range parseBaseMessages {
		parseTrimmedMessageKey := strings.TrimSpace(parseMessageKey)
		if parseTrimmedMessageKey == "" {
			continue
		}
		parseNormalizedBaseMessages[parseTrimmedMessageKey] = parseMessageValue
	}
	parseCandidateKeys := make([]string, 0, len(parseCandidateMessages))
	for parseMessageKey := range parseCandidateMessages {
		parseCandidateKeys = append(parseCandidateKeys, parseMessageKey)
	}
	sort.Strings(parseCandidateKeys)
	for _, parseMessageKey := range parseCandidateKeys {
		parseTrimmedMessageKey := strings.TrimSpace(parseMessageKey)
		if parseTrimmedMessageKey == "" {
			return status.Error(codes.InvalidArgument, "catalog message key is required")
		}
		parseBaseMessageValue, hasParseBaseMessage := parseNormalizedBaseMessages[parseTrimmedMessageKey]
		if !hasParseBaseMessage {
			return status.Errorf(codes.InvalidArgument, "catalog message %q is not present in base layer", parseTrimmedMessageKey)
		}
		parseBaseTemplateSet, parseErr := parseExtractCatalogMessageTemplateSet(parseBaseMessageValue)
		if parseErr != nil {
			return status.Errorf(codes.InvalidArgument, "catalog base message %q has malformed placeholders: %v", parseTrimmedMessageKey, parseErr)
		}
		parseCandidateTemplateSet, parseErr := parseExtractCatalogMessageTemplateSet(parseCandidateMessages[parseMessageKey])
		if parseErr != nil {
			return status.Errorf(codes.InvalidArgument, "catalog message %q has malformed placeholders: %v", parseTrimmedMessageKey, parseErr)
		}
		for parseTemplateToken := range parseBaseTemplateSet {
			if _, hasParseToken := parseCandidateTemplateSet[parseTemplateToken]; !hasParseToken {
				return status.Errorf(codes.InvalidArgument, "catalog message %q is missing template variable %q", parseTrimmedMessageKey, parseTemplateToken)
			}
		}
		for parseTemplateToken := range parseCandidateTemplateSet {
			if _, hasParseToken := parseBaseTemplateSet[parseTemplateToken]; !hasParseToken {
				return status.Errorf(codes.InvalidArgument, "catalog message %q contains unsupported template variable %q", parseTrimmedMessageKey, parseTemplateToken)
			}
		}
	}
	return nil
}

// parseExtractCatalogMessageTemplateSet parses one message template and returns referenced placeholder tokens.
func parseExtractCatalogMessageTemplateSet(parseMessageValue string) (map[string]struct{}, error) {
	parseTemplateSet := make(map[string]struct{})
	parseMessageValue = strings.TrimSpace(parseMessageValue)
	for {
		parseTemplateOpenIndex := strings.Index(parseMessageValue, "{{")
		parseTemplateCloseIndex := strings.Index(parseMessageValue, "}}")
		if parseTemplateCloseIndex >= 0 && (parseTemplateOpenIndex < 0 || parseTemplateCloseIndex < parseTemplateOpenIndex) {
			return nil, status.Error(codes.InvalidArgument, "unexpected template close token")
		}
		if parseTemplateOpenIndex < 0 {
			return parseTemplateSet, nil
		}
		parseTokenSearch := parseMessageValue[parseTemplateOpenIndex+2:]
		parseTokenEndIndex := strings.Index(parseTokenSearch, "}}")
		if parseTokenEndIndex < 0 {
			return nil, status.Error(codes.InvalidArgument, "template token is missing close token")
		}
		parseTemplateToken := strings.TrimSpace(parseTokenSearch[:parseTokenEndIndex])
		if !parseIsCatalogTemplateTokenValid(parseTemplateToken) {
			return nil, status.Errorf(codes.InvalidArgument, "template token %q is malformed", parseTemplateToken)
		}
		parseTemplateSet[parseTemplateToken] = struct{}{}
		parseMessageValue = parseTokenSearch[parseTokenEndIndex+2:]
	}
}

// parseIsCatalogTemplateTokenValid reports whether one template token uses one allowlisted identifier format.
func parseIsCatalogTemplateTokenValid(parseTemplateToken string) bool {
	parseTemplateToken = strings.TrimSpace(parseTemplateToken)
	if parseTemplateToken == "" {
		return false
	}
	for parseTokenIndex, parseTokenRune := range parseTemplateToken {
		switch {
		case parseTokenIndex == 0 && (unicode.IsLetter(parseTokenRune) || parseTokenRune == '_'):
			continue
		case parseTokenIndex > 0 && (unicode.IsLetter(parseTokenRune) || unicode.IsDigit(parseTokenRune) || parseTokenRune == '_' || parseTokenRune == '.'):
			continue
		default:
			return false
		}
	}
	return true
}

// parseRequireCatalogOverrideWriteScope authorizes one override-layer write against namespace and caller scope boundaries.
func (parseS *chatServer) parseRequireCatalogOverrideWriteScope(parseCtx context.Context, parseNamespace string, parseSourceLayer string, parseWorkspaceID int64) (parseAdminAccessScope, error) {
	parseNamespace = strings.TrimSpace(parseNamespace)
	if parseNamespace == "" {
		return parseAdminAccessScope{}, status.Error(codes.InvalidArgument, "catalog namespace is required")
	}
	parseNamespaceClass := parseResolveCatalogNamespaceClass(parseNamespace)
	if parseNamespaceClass == parseCatalogNamespaceClassUnknown {
		return parseAdminAccessScope{}, status.Errorf(codes.PermissionDenied, "catalog namespace %q is not allowlisted", parseNamespace)
	}
	parseSourceLayer = parseNormalizeCatalogSourceLayer(parseSourceLayer)
	switch parseSourceLayer {
	case parseCatalogSourceLayerGoEmbed:
		return parseAdminAccessScope{}, status.Errorf(codes.PermissionDenied, "catalog source layer %q is immutable", parseSourceLayer)
	case parseCatalogSourceLayerSiteOverride, parseCatalogSourceLayerEnvironmentOverride, parseCatalogSourceLayerExperimentOverride, parseCatalogSourceLayerCampaignOverride:
		parseScope, parseErr := parseS.parseRequireAdminAccessScope(parseCtx)
		if parseErr != nil {
			return parseAdminAccessScope{}, parseErr
		}
		if !parseScope.isPlatformScope {
			return parseAdminAccessScope{}, status.Errorf(codes.PermissionDenied, "catalog source layer %q requires superuser scope", parseSourceLayer)
		}
		return parseScope, nil
	case parseCatalogSourceLayerTenantOverride:
		parseScope, parseErr := parseS.parseRequireAdminAccessScope(parseCtx)
		if parseErr != nil {
			return parseAdminAccessScope{}, parseErr
		}
		if parseScope.isPlatformScope {
			return parseScope, nil
		}
		if parseNamespaceClass == parseCatalogNamespaceClassPublic {
			return parseAdminAccessScope{}, status.Error(codes.PermissionDenied, "workspace admin cannot write public catalog namespaces")
		}
		if parseWorkspaceID <= 0 {
			return parseAdminAccessScope{}, status.Error(codes.InvalidArgument, "workspace id is required for tenant override")
		}
		if _, hasParseWorkspace := parseScope.workspaceIDs[parseWorkspaceID]; !hasParseWorkspace {
			return parseAdminAccessScope{}, status.Error(codes.PermissionDenied, "workspace outside admin scope")
		}
		if parseNamespaceClass == parseCatalogNamespaceClassAdmin {
			parseSliceID, parseErr := parseResolveCatalogAdminSliceID(parseNamespace)
			if parseErr != nil {
				return parseAdminAccessScope{}, parseErr
			}
			if _, parseErr = parseS.parseRequireAdminSliceScope(parseCtx, parseSliceID); parseErr != nil {
				return parseAdminAccessScope{}, parseErr
			}
		}
		return parseScope, nil
	default:
		return parseAdminAccessScope{}, status.Errorf(codes.InvalidArgument, "unsupported catalog source layer %q", parseSourceLayer)
	}
}
