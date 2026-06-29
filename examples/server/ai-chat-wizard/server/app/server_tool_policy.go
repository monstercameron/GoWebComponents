package app

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	chatpb "github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/proto"
)

const (
	serverToolPolicyConfigEnabledKey           = "server-tools.enabled"
	serverToolPolicyConfigWhitelistKey         = "server-tools.command-policy.v1"
	serverToolPolicyConfigMaxSessionSecondsKey = "server-tools.max-session-seconds"
	serverToolPolicyConfigMaxOutputBytesKey    = "server-tools.max-output-bytes"

	serverToolPolicyDefaultMaxSessionSeconds int32 = 300
	serverToolPolicyDefaultMaxOutputBytes    int32 = 256 * 1024

	serverToolPolicyMinimumMaxSessionSeconds int32 = 5
	serverToolPolicyMaximumMaxSessionSeconds int32 = 12 * 60 * 60
	serverToolPolicyMinimumMaxOutputBytes    int32 = 1024
	serverToolPolicyMaximumMaxOutputBytes    int32 = 16 * 1024 * 1024
)

type parseServerToolPolicyEnvelope struct {
	ParseApprovedTools []parseServerToolPolicyJSONRule `json:"approved_tools"`
}

type parseServerToolPolicyJSONRule struct {
	ParseToolID           string   `json:"tool_id"`
	ParseDescription      string   `json:"description"`
	IsParseEnabled        *bool    `json:"is_enabled"`
	ParseShell            string   `json:"shell"`
	ParseArgvPrefix       []string `json:"argv_prefix"`
	ParseAllowArgsRegex   []string `json:"allow_args_regex"`
	ParseDenyArgsRegex    []string `json:"deny_args_regex"`
	ParseAllowCwdPrefixes []string `json:"allow_cwd_prefixes"`
}

// parseResolveServerToolPolicy resolves one policy response from site_config rows with safe defaults.
func parseResolveServerToolPolicy(parseStore *Store) (*chatpb.GetServerToolPolicyResponse, error) {
	parseResponse := &chatpb.GetServerToolPolicyResponse{
		IsEnabled:         false,
		MaxSessionSeconds: serverToolPolicyDefaultMaxSessionSeconds,
		MaxOutputBytes:    serverToolPolicyDefaultMaxOutputBytes,
		ApprovedTools:     nil,
		Source:            "defaults",
		UpdatedAt:         "",
	}
	if parseStore == nil {
		return parseResponse, nil
	}

	parseRows, parseErr := parseStore.parseListSiteConfigs()
	if parseErr != nil {
		return nil, parseErr
	}
	parseRowsByKey := parseBuildServerToolPolicyRowsByKey(parseRows)

	var parseUpdatedAt string
	var isParseSiteConfigUsed bool
	if parseRow, parseOk := parseRowsByKey[serverToolPolicyConfigEnabledKey]; parseOk {
		parseResponse.IsEnabled = parseBuildServerToolPolicyBool(parseRow.ConfigValue, false)
		parseUpdatedAt = parseBuildServerToolPolicyNewestTimestamp(parseUpdatedAt, parseRow.UpdatedAt)
		isParseSiteConfigUsed = true
	}
	if parseRow, parseOk := parseRowsByKey[serverToolPolicyConfigMaxSessionSecondsKey]; parseOk {
		parseResponse.MaxSessionSeconds = parseBuildServerToolPolicyInt32(parseRow.ConfigValue, serverToolPolicyDefaultMaxSessionSeconds, serverToolPolicyMinimumMaxSessionSeconds, serverToolPolicyMaximumMaxSessionSeconds)
		parseUpdatedAt = parseBuildServerToolPolicyNewestTimestamp(parseUpdatedAt, parseRow.UpdatedAt)
		isParseSiteConfigUsed = true
	}
	if parseRow, parseOk := parseRowsByKey[serverToolPolicyConfigMaxOutputBytesKey]; parseOk {
		parseResponse.MaxOutputBytes = parseBuildServerToolPolicyInt32(parseRow.ConfigValue, serverToolPolicyDefaultMaxOutputBytes, serverToolPolicyMinimumMaxOutputBytes, serverToolPolicyMaximumMaxOutputBytes)
		parseUpdatedAt = parseBuildServerToolPolicyNewestTimestamp(parseUpdatedAt, parseRow.UpdatedAt)
		isParseSiteConfigUsed = true
	}
	if parseRow, parseOk := parseRowsByKey[serverToolPolicyConfigWhitelistKey]; parseOk {
		parseRules, parseErr2 := parseBuildServerToolPolicyRules(parseRow.ConfigValue)
		if parseErr2 != nil {
			return nil, fmt.Errorf("parse whitelist rules from site_config[%s]: %w", serverToolPolicyConfigWhitelistKey, parseErr2)
		}
		parseResponse.ApprovedTools = parseRules
		parseUpdatedAt = parseBuildServerToolPolicyNewestTimestamp(parseUpdatedAt, parseRow.UpdatedAt)
		isParseSiteConfigUsed = true
	}

	if isParseSiteConfigUsed {
		parseResponse.Source = "site_config"
	}
	parseResponse.UpdatedAt = parseUpdatedAt
	return parseResponse, nil
}

// parseBuildServerToolPolicyRowsByKey builds one fast key lookup from site_config rows.
func parseBuildServerToolPolicyRowsByKey(parseRows []parseSiteConfigRow) map[string]parseSiteConfigRow {
	parseRowsByKey := make(map[string]parseSiteConfigRow, len(parseRows))
	for _, parseRow := range parseRows {
		parseKey := strings.TrimSpace(parseRow.ConfigKey)
		if parseKey == "" {
			continue
		}
		parseRowsByKey[parseKey] = parseRow
	}
	return parseRowsByKey
}

// parseBuildServerToolPolicyBool parses one configuration boolean with fallback behavior.
func parseBuildServerToolPolicyBool(parseRawValue string, isParseFallback bool) bool {
	parseValue := strings.ToLower(strings.TrimSpace(parseRawValue))
	switch parseValue {
	case "1", "true", "t", "yes", "y", "on":
		return true
	case "0", "false", "f", "no", "n", "off":
		return false
	default:
		return isParseFallback
	}
}

// parseBuildServerToolPolicyInt32 parses one bounded int32 with fallback behavior.
func parseBuildServerToolPolicyInt32(parseRawValue string, parseFallback, parseMinimum, parseMaximum int32) int32 {
	parseValue := strings.TrimSpace(parseRawValue)
	if parseValue == "" {
		return parseFallback
	}
	parseParsed, parseErr := strconv.ParseInt(parseValue, 10, 32)
	if parseErr != nil {
		return parseFallback
	}
	return parseClampServerToolPolicyInt32(int32(parseParsed), parseMinimum, parseMaximum)
}

// parseClampServerToolPolicyInt32 clamps one int32 value into inclusive bounds.
func parseClampServerToolPolicyInt32(parseValue, parseMinimum, parseMaximum int32) int32 {
	if parseValue < parseMinimum {
		return parseMinimum
	}
	if parseValue > parseMaximum {
		return parseMaximum
	}
	return parseValue
}

// parseBuildServerToolPolicyRules parses one whitelist JSON payload into protobuf rule rows.
func parseBuildServerToolPolicyRules(parseRawJSON string) ([]*chatpb.ServerToolPolicyRule, error) {
	parseRawJSON = strings.TrimSpace(parseRawJSON)
	if parseRawJSON == "" {
		return nil, nil
	}

	var parseEnvelope parseServerToolPolicyEnvelope
	parseEnvelopeErr := json.Unmarshal([]byte(parseRawJSON), &parseEnvelope)
	if parseEnvelopeErr == nil {
		return parseBuildServerToolPolicyRulesFromJSON(parseEnvelope.ParseApprovedTools), nil
	}

	var parseArray []parseServerToolPolicyJSONRule
	parseArrayErr := json.Unmarshal([]byte(parseRawJSON), &parseArray)
	if parseArrayErr != nil {
		if parseEnvelopeErr != nil {
			return nil, fmt.Errorf("parse as object: %v; parse as array: %w", parseEnvelopeErr, parseArrayErr)
		}
		return nil, parseArrayErr
	}
	return parseBuildServerToolPolicyRulesFromJSON(parseArray), nil
}

// parseBuildServerToolPolicyRulesFromJSON sanitizes one JSON rule list into protobuf policy rules.
func parseBuildServerToolPolicyRulesFromJSON(parseJSONRules []parseServerToolPolicyJSONRule) []*chatpb.ServerToolPolicyRule {
	parseRules := make([]*chatpb.ServerToolPolicyRule, 0, len(parseJSONRules))
	for _, parseJSONRule := range parseJSONRules {
		parseRule, parseOk := parseBuildServerToolPolicyRule(parseJSONRule)
		if !parseOk {
			continue
		}
		parseRules = append(parseRules, parseRule)
	}
	return parseRules
}

// parseBuildServerToolPolicyRule sanitizes one JSON rule into one protobuf rule when required fields exist.
func parseBuildServerToolPolicyRule(parseJSONRule parseServerToolPolicyJSONRule) (*chatpb.ServerToolPolicyRule, bool) {
	parseToolID := strings.TrimSpace(parseJSONRule.ParseToolID)
	if parseToolID == "" {
		return nil, false
	}
	isParseEnabled := true
	if parseJSONRule.IsParseEnabled != nil {
		isParseEnabled = *parseJSONRule.IsParseEnabled
	}
	return &chatpb.ServerToolPolicyRule{
		ToolId:           parseToolID,
		Description:      strings.TrimSpace(parseJSONRule.ParseDescription),
		IsEnabled:        isParseEnabled,
		Shell:            parseBuildServerToolPolicyShell(parseJSONRule.ParseShell),
		ArgvPrefix:       parseBuildServerToolPolicyStringList(parseJSONRule.ParseArgvPrefix),
		AllowArgsRegex:   parseBuildServerToolPolicyStringList(parseJSONRule.ParseAllowArgsRegex),
		DenyArgsRegex:    parseBuildServerToolPolicyStringList(parseJSONRule.ParseDenyArgsRegex),
		AllowCwdPrefixes: parseBuildServerToolPolicyStringList(parseJSONRule.ParseAllowCwdPrefixes),
	}, true
}

// parseBuildServerToolPolicyShell normalizes one shell selector into supported values.
func parseBuildServerToolPolicyShell(parseRawShell string) string {
	switch strings.ToLower(strings.TrimSpace(parseRawShell)) {
	case "powershell", "bash", "auto":
		return strings.ToLower(strings.TrimSpace(parseRawShell))
	default:
		return "auto"
	}
}

// parseBuildServerToolPolicyStringList trims one dynamic string list and drops blanks.
func parseBuildServerToolPolicyStringList(parseRawValues []string) []string {
	parseValues := make([]string, 0, len(parseRawValues))
	for _, parseRawValue := range parseRawValues {
		parseValue := strings.TrimSpace(parseRawValue)
		if parseValue == "" {
			continue
		}
		parseValues = append(parseValues, parseValue)
	}
	return parseValues
}

// parseBuildServerToolPolicyNewestTimestamp keeps the latest RFC3339 timestamp by lexical compare.
func parseBuildServerToolPolicyNewestTimestamp(parseCurrent string, parseCandidate string) string {
	parseCurrent = strings.TrimSpace(parseCurrent)
	parseCandidate = strings.TrimSpace(parseCandidate)
	if parseCandidate == "" {
		return parseCurrent
	}
	if parseCurrent == "" || parseCandidate > parseCurrent {
		return parseCandidate
	}
	return parseCurrent
}
