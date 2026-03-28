package app

import "strings"

type parseCatalogLaunchTruthCapability string

const (
	parseCatalogLaunchTruthCapabilityDocsSearch    parseCatalogLaunchTruthCapability = "docs_search"
	parseCatalogLaunchTruthCapabilityTeamKnowledge parseCatalogLaunchTruthCapability = "team_knowledge"
	parseCatalogLaunchTruthCapabilityGoogleLogin   parseCatalogLaunchTruthCapability = "google_login"
	parseCatalogLaunchTruthCapabilityEnterpriseSSO parseCatalogLaunchTruthCapability = "enterprise_sso"
)

type parseCatalogLaunchTruthConfig map[parseCatalogLaunchTruthCapability]bool

type parseCatalogLaunchTruthRule struct {
	parseLaunchCapabilities []parseCatalogLaunchTruthCapability
}

// parseBuildCatalogLaunchTruthConfig returns one launch capability matrix for content filtering.
func parseBuildCatalogLaunchTruthConfig() parseCatalogLaunchTruthConfig {
	return parseCatalogLaunchTruthConfig{
		parseCatalogLaunchTruthCapabilityDocsSearch:    false,
		parseCatalogLaunchTruthCapabilityTeamKnowledge: false,
		parseCatalogLaunchTruthCapabilityGoogleLogin:   false,
		parseCatalogLaunchTruthCapabilityEnterpriseSSO: false,
	}
}

var parseCatalogLaunchTruthMessageRules = map[string]parseCatalogLaunchTruthRule{
	"hero.home.body":                    {parseLaunchCapabilities: []parseCatalogLaunchTruthCapability{parseCatalogLaunchTruthCapabilityDocsSearch, parseCatalogLaunchTruthCapabilityTeamKnowledge}},
	"hero.pricing.body":                 {parseLaunchCapabilities: []parseCatalogLaunchTruthCapability{parseCatalogLaunchTruthCapabilityEnterpriseSSO}},
	"product.home.body":                 {parseLaunchCapabilities: []parseCatalogLaunchTruthCapability{parseCatalogLaunchTruthCapabilityDocsSearch, parseCatalogLaunchTruthCapabilityTeamKnowledge}},
	"product.home.card.docqa.title":     {parseLaunchCapabilities: []parseCatalogLaunchTruthCapability{parseCatalogLaunchTruthCapabilityDocsSearch}},
	"product.home.card.docqa.body":      {parseLaunchCapabilities: []parseCatalogLaunchTruthCapability{parseCatalogLaunchTruthCapabilityDocsSearch}},
	"product.home.card.enterprise.body": {parseLaunchCapabilities: []parseCatalogLaunchTruthCapability{parseCatalogLaunchTruthCapabilityEnterpriseSSO}},
	"product.capabilities.body":         {parseLaunchCapabilities: []parseCatalogLaunchTruthCapability{parseCatalogLaunchTruthCapabilityDocsSearch, parseCatalogLaunchTruthCapabilityTeamKnowledge}},
	"pricing.enterprise.body":           {parseLaunchCapabilities: []parseCatalogLaunchTruthCapability{parseCatalogLaunchTruthCapabilityEnterpriseSSO}},
	"pricing.enterprise.description":    {parseLaunchCapabilities: []parseCatalogLaunchTruthCapability{parseCatalogLaunchTruthCapabilityEnterpriseSSO}},
	"pricing.enterprise.feature.1":      {parseLaunchCapabilities: []parseCatalogLaunchTruthCapability{parseCatalogLaunchTruthCapabilityEnterpriseSSO}},
	"compare.admin.enterprise":          {parseLaunchCapabilities: []parseCatalogLaunchTruthCapability{parseCatalogLaunchTruthCapabilityEnterpriseSSO}},
	"compare.compliance.enterprise":     {parseLaunchCapabilities: []parseCatalogLaunchTruthCapability{parseCatalogLaunchTruthCapabilityEnterpriseSSO}},
	"info.security.access.body":         {parseLaunchCapabilities: []parseCatalogLaunchTruthCapability{parseCatalogLaunchTruthCapabilityEnterpriseSSO}},
}

var parseCatalogLaunchTruthFallbackByLanguage = map[string]map[parseCatalogLaunchTruthCapability]string{
	"en": {
		parseCatalogLaunchTruthCapabilityDocsSearch:    "This launch uses the shipped feature set only; document search and knowledge discovery are planned for a later release.",
		parseCatalogLaunchTruthCapabilityTeamKnowledge: "This launch uses the shipped feature set only; team knowledge management is planned for a later release.",
		parseCatalogLaunchTruthCapabilityGoogleLogin:   "This launch keeps Google sign-in behind configuration; local credentials remain the supported path.",
		parseCatalogLaunchTruthCapabilityEnterpriseSSO: "This launch uses the shipped feature set only; enterprise SSO is planned for a later release.",
	},
	"es": {
		parseCatalogLaunchTruthCapabilityDocsSearch:    "This launch uses the shipped feature set only; document search and knowledge discovery are planned for a later release.",
		parseCatalogLaunchTruthCapabilityTeamKnowledge: "This launch uses the shipped feature set only; team knowledge management is planned for a later release.",
		parseCatalogLaunchTruthCapabilityGoogleLogin:   "This launch keeps Google sign-in behind configuration; local credentials remain the supported path.",
		parseCatalogLaunchTruthCapabilityEnterpriseSSO: "This launch uses the shipped feature set only; enterprise SSO is planned for a later release.",
	},
	"fr": {
		parseCatalogLaunchTruthCapabilityDocsSearch:    "This launch uses the shipped feature set only; document search and knowledge discovery are planned for a later release.",
		parseCatalogLaunchTruthCapabilityTeamKnowledge: "This launch uses the shipped feature set only; team knowledge management is planned for a later release.",
		parseCatalogLaunchTruthCapabilityGoogleLogin:   "This launch keeps Google sign-in behind configuration; local credentials remain the supported path.",
		parseCatalogLaunchTruthCapabilityEnterpriseSSO: "This launch uses the shipped feature set only; enterprise SSO is planned for a later release.",
	},
	"de": {
		parseCatalogLaunchTruthCapabilityDocsSearch:    "This launch uses the shipped feature set only; document search and knowledge discovery are planned for a later release.",
		parseCatalogLaunchTruthCapabilityTeamKnowledge: "This launch uses the shipped feature set only; team knowledge management is planned for a later release.",
		parseCatalogLaunchTruthCapabilityGoogleLogin:   "This launch keeps Google sign-in behind configuration; local credentials remain the supported path.",
		parseCatalogLaunchTruthCapabilityEnterpriseSSO: "This launch uses the shipped feature set only; enterprise SSO is planned for a later release.",
	},
}

// parseApplyCatalogLaunchTruthFilter returns launch-safe catalog messages based on shipped capability settings.
func parseApplyCatalogLaunchTruthFilter(parseMessages map[string]string, parseLocale string) map[string]string {
	return parseApplyCatalogLaunchTruthFilterWithConfig(parseMessages, parseLocale, parseBuildCatalogLaunchTruthConfig())
}

// parseApplyCatalogLaunchTruthFilterWithConfig returns launch-safe messages for one explicit config and locale.
func parseApplyCatalogLaunchTruthFilterWithConfig(parseMessages map[string]string, parseLocale string, parseConfig parseCatalogLaunchTruthConfig) map[string]string {
	parseResult := make(map[string]string, len(parseMessages))
	parseLanguage := parseNormalizeCatalogLocaleForTruth(parseLocale)
	for parseMessageKey, parseMessageValue := range parseMessages {
		parseMessageKey = strings.TrimSpace(parseMessageKey)
		parseMessageValue = strings.TrimSpace(parseMessageValue)
		if parseMessageKey == "" || parseMessageValue == "" {
			continue
		}
		parseRule, hasParseRule := parseCatalogLaunchTruthMessageRules[parseMessageKey]
		if hasParseRule {
			parseReplacement := parseResolveCatalogLaunchTruthReplacement(parseRule.parseLaunchCapabilities, parseConfig, parseLanguage)
			if parseReplacement != "" {
				parseResult[parseMessageKey] = parseReplacement
				continue
			}
		}
		parseResult[parseMessageKey] = parseMessageValue
	}
	return parseResult
}

// parseResolveCatalogLaunchTruthReplacement returns a launch-safe message if one capability is not available.
func parseResolveCatalogLaunchTruthReplacement(parseCapabilities []parseCatalogLaunchTruthCapability, parseConfig parseCatalogLaunchTruthConfig, parseLanguage string) string {
	for _, parseCapability := range parseCapabilities {
		if parseConfig[parseCapability] {
			continue
		}
		if parseCatalogFallbackByLanguage, hasParseLanguage := parseCatalogLaunchTruthFallbackByLanguage[parseLanguage]; hasParseLanguage {
			if parseFallback, hasParseFallback := parseCatalogFallbackByLanguage[parseCapability]; hasParseFallback {
				return parseFallback
			}
		}
		if parseCatalogFallbackByLanguage, hasParseLanguage := parseCatalogLaunchTruthFallbackByLanguage["en"]; hasParseLanguage {
			if parseFallback, hasParseFallback := parseCatalogFallbackByLanguage[parseCapability]; hasParseFallback {
				return parseFallback
			}
		}
	}
	return ""
}

// parseNormalizeCatalogLocaleForTruth normalizes a locale before launch-truth lookup.
func parseNormalizeCatalogLocaleForTruth(parseLocale string) string {
	parseLocale = strings.TrimSpace(strings.ToLower(parseLocale))
	parseLocale = strings.ReplaceAll(parseLocale, "_", "-")
	if parseLocale == "" {
		return "en"
	}
	parseParts := strings.SplitN(parseLocale, "-", 2)
	if len(parseParts) == 0 {
		return "en"
	}
	return parseParts[0]
}
