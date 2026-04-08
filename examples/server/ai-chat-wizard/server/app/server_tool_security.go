package app

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	chatpb "github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const parseServerToolDangerousConfirmMetadataKey = "x-chat-server-tool-dangerous-confirm"
const parseServerToolDangerousConfirmMetadataAltKey = "x-confirm-dangerous-change"

// parseValidateSetServerToolPolicyRequest validates one server-tool policy mutation request.
func parseValidateSetServerToolPolicyRequest(parseReq *chatpb.SetServerToolPolicyRequest) error {
	if parseReq == nil {
		return status.Error(codes.InvalidArgument, "server tool policy request is required")
	}
	if parseReq.GetMaxSessionSeconds() < 0 {
		return status.Error(codes.InvalidArgument, "max session seconds must be non-negative")
	}
	if parseReq.GetMaxOutputBytes() < 0 {
		return status.Error(codes.InvalidArgument, "max output bytes must be non-negative")
	}
	parseRules, parseErr := parseValidateServerToolPolicyRuleSet(parseReq.GetApprovedTools())
	if parseErr != nil {
		return parseErr
	}
	if parseReq.GetIsEnabled() && len(parseRules) == 0 {
		return status.Error(codes.InvalidArgument, "server tools enabled policy requires at least one approved tool rule")
	}
	if parseReq.GetIsEnabled() && !parseHasServerToolPolicyEnabledRule(parseRules) {
		return status.Error(codes.InvalidArgument, "server tools enabled policy requires at least one enabled tool rule")
	}
	return nil
}

// parseValidateServerToolPolicyRuleSet validates one server-tool policy rule list and returns sanitized rules.
func parseValidateServerToolPolicyRuleSet(parseRawRules []*chatpb.ServerToolPolicyRule) ([]*chatpb.ServerToolPolicyRule, error) {
	parseRules := parseBuildServerToolPolicyRulesFromRequest(parseRawRules)
	parseRuleIDs := make(map[string]struct{}, len(parseRules))
	for _, parseRule := range parseRules {
		parseRuleID := strings.TrimSpace(parseRule.GetToolId())
		if parseRuleID == "" {
			return nil, status.Error(codes.InvalidArgument, "server tool rule tool_id is required")
		}
		if _, hasParseRuleID := parseRuleIDs[parseRuleID]; hasParseRuleID {
			return nil, status.Errorf(codes.InvalidArgument, "server tool rule tool_id %q must be unique", parseRuleID)
		}
		parseRuleIDs[parseRuleID] = struct{}{}
		parseRuleShell := parseBuildServerToolPolicyShell(parseRule.GetShell())
		parseRawShell := strings.TrimSpace(strings.ToLower(parseRule.GetShell()))
		if parseRawShell != "" && parseRuleShell != parseRawShell {
			return nil, status.Errorf(codes.InvalidArgument, "server tool rule %q shell is invalid", parseRuleID)
		}
		parseArgvPrefix := parseBuildServerToolPolicyStringList(parseRule.GetArgvPrefix())
		if parseRule.GetIsEnabled() && len(parseArgvPrefix) == 0 {
			return nil, status.Errorf(codes.InvalidArgument, "server tool rule %q argv_prefix is required for enabled rules", parseRuleID)
		}
		if parseErr := parseValidateServerToolArgvPrefixTokens(parseArgvPrefix); parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "server tool rule %q argv_prefix invalid: %v", parseRuleID, parseErr)
		}
		parseAllowArgsRegex := parseBuildServerToolPolicyStringList(parseRule.GetAllowArgsRegex())
		parseDenyArgsRegex := parseBuildServerToolPolicyStringList(parseRule.GetDenyArgsRegex())
		if parseRule.GetIsEnabled() && len(parseAllowArgsRegex)+len(parseDenyArgsRegex) == 0 {
			return nil, status.Errorf(codes.InvalidArgument, "server tool rule %q requires one argument policy regex", parseRuleID)
		}
		if parseErr := parseValidateServerToolRegexList(parseAllowArgsRegex); parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "server tool rule %q allow_args_regex invalid: %v", parseRuleID, parseErr)
		}
		if parseErr := parseValidateServerToolRegexList(parseDenyArgsRegex); parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "server tool rule %q deny_args_regex invalid: %v", parseRuleID, parseErr)
		}
	}
	return parseRules, nil
}

// parseValidateServerToolArgvPrefixTokens validates one argv prefix token list.
func parseValidateServerToolArgvPrefixTokens(parseArgvPrefix []string) error {
	for _, parseToken := range parseArgvPrefix {
		parseTrimmedToken := strings.TrimSpace(parseToken)
		if parseTrimmedToken == "" {
			return errors.New("argv prefix token is required")
		}
		if parseTrimmedToken != parseToken {
			return errors.New("argv prefix token cannot include leading or trailing whitespace")
		}
		if strings.ContainsAny(parseToken, "\r\n\t") {
			return errors.New("argv prefix token cannot include control characters")
		}
		if strings.Contains(parseToken, " ") {
			return errors.New("argv prefix token cannot include spaces")
		}
	}
	return nil
}

// parseValidateServerToolRegexList validates one list of regex values.
func parseValidateServerToolRegexList(parseRegexValues []string) error {
	for _, parseRegexValue := range parseRegexValues {
		if _, parseErr := regexp.Compile(parseRegexValue); parseErr != nil {
			return parseErr
		}
	}
	return nil
}

// parseHasServerToolPolicyEnabledRule reports whether one rule list contains one enabled rule.
func parseHasServerToolPolicyEnabledRule(parseRules []*chatpb.ServerToolPolicyRule) bool {
	for _, parseRule := range parseRules {
		if parseRule != nil && parseRule.GetIsEnabled() {
			return true
		}
	}
	return false
}

// parseResolveServerToolDangerousChangeConfirmation resolves one explicit dangerous-change confirmation flag from metadata.
func parseResolveServerToolDangerousChangeConfirmation(parseCtx context.Context) bool {
	parseMetadata, hasParseMetadata := metadata.FromIncomingContext(parseCtx)
	if !hasParseMetadata {
		return false
	}
	parseConfirmValue := parseFirstNonEmptyMetadataValue(parseMetadata, parseServerToolDangerousConfirmMetadataKey, parseServerToolDangerousConfirmMetadataAltKey)
	switch strings.TrimSpace(strings.ToLower(parseConfirmValue)) {
	case "1", "true", "yes", "y", "confirm", "confirmed":
		return true
	default:
		return false
	}
}

// parseHasServerToolPolicyDangerousChange reports whether one policy update widens runtime risk and requires explicit confirmation.
func parseHasServerToolPolicyDangerousChange(parseCurrent *chatpb.GetServerToolPolicyResponse, parseReq *chatpb.SetServerToolPolicyRequest) bool {
	if parseReq == nil {
		return false
	}
	if parseCurrent == nil {
		parseCurrent = &chatpb.GetServerToolPolicyResponse{}
	}
	if !parseCurrent.GetIsEnabled() && parseReq.GetIsEnabled() {
		return true
	}
	parseCurrentMaxSessionSeconds := parseCurrent.GetMaxSessionSeconds()
	if parseCurrentMaxSessionSeconds <= 0 {
		parseCurrentMaxSessionSeconds = serverToolPolicyDefaultMaxSessionSeconds
	}
	parseRequestedMaxSessionSeconds := parseReq.GetMaxSessionSeconds()
	if parseRequestedMaxSessionSeconds <= 0 {
		parseRequestedMaxSessionSeconds = serverToolPolicyDefaultMaxSessionSeconds
	}
	if parseRequestedMaxSessionSeconds > parseCurrentMaxSessionSeconds {
		return true
	}
	parseCurrentMaxOutputBytes := parseCurrent.GetMaxOutputBytes()
	if parseCurrentMaxOutputBytes <= 0 {
		parseCurrentMaxOutputBytes = serverToolPolicyDefaultMaxOutputBytes
	}
	parseRequestedMaxOutputBytes := parseReq.GetMaxOutputBytes()
	if parseRequestedMaxOutputBytes <= 0 {
		parseRequestedMaxOutputBytes = serverToolPolicyDefaultMaxOutputBytes
	}
	if parseRequestedMaxOutputBytes > parseCurrentMaxOutputBytes {
		return true
	}
	return parseHasServerToolPolicyEnabledRuleExpansion(parseCurrent.GetApprovedTools(), parseReq.GetApprovedTools())
}

// parseHasServerToolPolicyEnabledRuleExpansion reports whether enabled tool rules are added or widened by one policy update.
func parseHasServerToolPolicyEnabledRuleExpansion(parseCurrentRules []*chatpb.ServerToolPolicyRule, parseRequestedRules []*chatpb.ServerToolPolicyRule) bool {
	parseCurrentRuleFingerprints := parseBuildServerToolRuleFingerprintMap(parseCurrentRules, true)
	parseRequestedRuleFingerprints := parseBuildServerToolRuleFingerprintMap(parseRequestedRules, true)
	if len(parseRequestedRuleFingerprints) == 0 {
		return false
	}
	if len(parseRequestedRuleFingerprints) > len(parseCurrentRuleFingerprints) {
		return true
	}
	for parseRuleID, parseRequestedFingerprint := range parseRequestedRuleFingerprints {
		parseCurrentFingerprint, hasParseCurrentRule := parseCurrentRuleFingerprints[parseRuleID]
		if !hasParseCurrentRule {
			return true
		}
		if parseRequestedFingerprint != parseCurrentFingerprint {
			return true
		}
	}
	return false
}

// parseBuildServerToolRuleFingerprintMap returns one fingerprint map for enabled or disabled rules keyed by tool id.
func parseBuildServerToolRuleFingerprintMap(parseRules []*chatpb.ServerToolPolicyRule, isParseEnabledOnly bool) map[string]string {
	parseRuleFingerprints := make(map[string]string)
	for _, parseRule := range parseBuildServerToolPolicyRulesFromRequest(parseRules) {
		if parseRule == nil {
			continue
		}
		if isParseEnabledOnly && !parseRule.GetIsEnabled() {
			continue
		}
		parseRuleID := strings.TrimSpace(parseRule.GetToolId())
		if parseRuleID == "" {
			continue
		}
		parseRuleFingerprints[parseRuleID] = strings.Join(
			[]string{
				parseBuildServerToolPolicyShell(parseRule.GetShell()),
				strings.Join(parseBuildServerToolPolicyStringList(parseRule.GetArgvPrefix()), "\x1f"),
				strings.Join(parseBuildServerToolPolicyStringList(parseRule.GetAllowArgsRegex()), "\x1f"),
				strings.Join(parseBuildServerToolPolicyStringList(parseRule.GetDenyArgsRegex()), "\x1f"),
				strings.Join(parseBuildServerToolPolicyStringList(parseRule.GetAllowCwdPrefixes()), "\x1f"),
			},
			"\x1e",
		)
	}
	return parseRuleFingerprints
}

// parseStoreServerToolPolicy persists one policy request and returns the applied policy snapshot.
func parseStoreServerToolPolicy(parseStore *Store, parseUserID int64, parseReq *chatpb.SetServerToolPolicyRequest) (*chatpb.GetServerToolPolicyResponse, error) {
	if parseStore == nil {
		return nil, errors.New("store unavailable")
	}
	if parseUserID <= 0 {
		return nil, errors.New("server tool policy user id is required")
	}
	if parseReq == nil {
		return nil, errors.New("server tool policy request is required")
	}
	if parseErr := parseValidateSetServerToolPolicyRequest(parseReq); parseErr != nil {
		return nil, parseErr
	}
	parseRules, parseErr := parseValidateServerToolPolicyRuleSet(parseReq.GetApprovedTools())
	if parseErr != nil {
		return nil, parseErr
	}
	parseMaxSessionSeconds := parseReq.GetMaxSessionSeconds()
	if parseMaxSessionSeconds <= 0 {
		parseMaxSessionSeconds = serverToolPolicyDefaultMaxSessionSeconds
	}
	parseMaxSessionSeconds = parseClampServerToolPolicyInt32(parseMaxSessionSeconds, serverToolPolicyMinimumMaxSessionSeconds, serverToolPolicyMaximumMaxSessionSeconds)
	parseMaxOutputBytes := parseReq.GetMaxOutputBytes()
	if parseMaxOutputBytes <= 0 {
		parseMaxOutputBytes = serverToolPolicyDefaultMaxOutputBytes
	}
	parseMaxOutputBytes = parseClampServerToolPolicyInt32(parseMaxOutputBytes, serverToolPolicyMinimumMaxOutputBytes, serverToolPolicyMaximumMaxOutputBytes)
	parseRulesJSON, parseErr := parseBuildServerToolPolicyRulesJSON(parseRules)
	if parseErr != nil {
		return nil, parseErr
	}
	if _, parseErr = parseStore.parseSetServerToolPolicy(parseServerToolPolicyStoreWrite{
		IsEnabled:         parseReq.GetIsEnabled(),
		MaxSessionSeconds: parseMaxSessionSeconds,
		MaxOutputBytes:    parseMaxOutputBytes,
		ApprovedToolsJSON: parseRulesJSON,
		UpdatedByUserID:   parseUserID,
		Source:            "set_server_tool_policy.rpc",
	}); parseErr != nil {
		return nil, parseErr
	}
	return parseResolveServerToolPolicy(parseStore)
}

// parseBuildServerToolPolicyRulesFromRequest sanitizes one policy-rule request payload.
func parseBuildServerToolPolicyRulesFromRequest(parseRawRules []*chatpb.ServerToolPolicyRule) []*chatpb.ServerToolPolicyRule {
	parseRules := make([]*chatpb.ServerToolPolicyRule, 0, len(parseRawRules))
	for _, parseRawRule := range parseRawRules {
		if parseRawRule == nil {
			continue
		}
		parseRuleID := strings.TrimSpace(parseRawRule.GetToolId())
		if parseRuleID == "" {
			continue
		}
		parseRules = append(parseRules, &chatpb.ServerToolPolicyRule{
			ToolId:           parseRuleID,
			Description:      strings.TrimSpace(parseRawRule.GetDescription()),
			IsEnabled:        parseRawRule.GetIsEnabled(),
			Shell:            parseBuildServerToolPolicyShell(parseRawRule.GetShell()),
			ArgvPrefix:       parseBuildServerToolPolicyStringList(parseRawRule.GetArgvPrefix()),
			AllowArgsRegex:   parseBuildServerToolPolicyStringList(parseRawRule.GetAllowArgsRegex()),
			DenyArgsRegex:    parseBuildServerToolPolicyStringList(parseRawRule.GetDenyArgsRegex()),
			AllowCwdPrefixes: parseBuildServerToolPolicyStringList(parseRawRule.GetAllowCwdPrefixes()),
		})
	}
	return parseRules
}

// parseBuildServerToolPolicyRulesJSON encodes one policy rule list into the site-config JSON envelope.
func parseBuildServerToolPolicyRulesJSON(parseRules []*chatpb.ServerToolPolicyRule) (string, error) {
	parseJSONRules := make([]parseServerToolPolicyJSONRule, 0, len(parseRules))
	for _, parseRule := range parseBuildServerToolPolicyRulesFromRequest(parseRules) {
		parseIsEnabled := parseRule.GetIsEnabled()
		parseJSONRules = append(parseJSONRules, parseServerToolPolicyJSONRule{
			ParseToolID:           strings.TrimSpace(parseRule.GetToolId()),
			ParseDescription:      strings.TrimSpace(parseRule.GetDescription()),
			IsParseEnabled:        &parseIsEnabled,
			ParseShell:            parseBuildServerToolPolicyShell(parseRule.GetShell()),
			ParseArgvPrefix:       parseBuildServerToolPolicyStringList(parseRule.GetArgvPrefix()),
			ParseAllowArgsRegex:   parseBuildServerToolPolicyStringList(parseRule.GetAllowArgsRegex()),
			ParseDenyArgsRegex:    parseBuildServerToolPolicyStringList(parseRule.GetDenyArgsRegex()),
			ParseAllowCwdPrefixes: parseBuildServerToolPolicyStringList(parseRule.GetAllowCwdPrefixes()),
		})
	}
	parseRawJSON, parseErr := json.Marshal(parseServerToolPolicyEnvelope{ParseApprovedTools: parseJSONRules})
	if parseErr != nil {
		return "", parseErr
	}
	return string(parseRawJSON), nil
}

// parseAuthorizeRunServerToolStart validates one start frame against the active server-tool policy.
func parseAuthorizeRunServerToolStart(parsePolicy *chatpb.GetServerToolPolicyResponse, parseStart *chatpb.RunServerToolStart) error {
	if parseStart == nil {
		return status.Error(codes.InvalidArgument, "run server tool start payload is required")
	}
	if parsePolicy == nil || !parsePolicy.GetIsEnabled() {
		return status.Error(codes.PermissionDenied, "server tool policy is disabled")
	}
	parseRules := parseBuildServerToolPolicyRulesFromRequest(parsePolicy.GetApprovedTools())
	if len(parseRules) == 0 {
		return status.Error(codes.PermissionDenied, "server tool policy has no approved command rules")
	}
	parseTimeoutSeconds := parseStart.GetTimeoutSeconds()
	if parseTimeoutSeconds <= 0 {
		parseTimeoutSeconds = parsePolicy.GetMaxSessionSeconds()
	}
	if parseTimeoutSeconds <= 0 {
		parseTimeoutSeconds = serverToolPolicyDefaultMaxSessionSeconds
	}
	parseMaxSessionSeconds := parsePolicy.GetMaxSessionSeconds()
	if parseMaxSessionSeconds <= 0 {
		parseMaxSessionSeconds = serverToolPolicyDefaultMaxSessionSeconds
	}
	if parseTimeoutSeconds > parseMaxSessionSeconds {
		return status.Errorf(codes.PermissionDenied, "requested timeout %d exceeds policy limit %d", parseTimeoutSeconds, parseMaxSessionSeconds)
	}
	parseRawShell := strings.TrimSpace(strings.ToLower(parseStart.GetShell()))
	parseShell := parseBuildServerToolPolicyShell(parseRawShell)
	if parseRawShell != "" && parseShell != parseRawShell {
		return status.Error(codes.InvalidArgument, "run server tool shell is invalid")
	}
	parseCommandTokens, parseErr := parseSplitServerToolCommandTokens(parseStart.GetCommand())
	if parseErr != nil {
		return status.Error(codes.InvalidArgument, parseErr.Error())
	}
	for _, parseRule := range parseRules {
		if parseRule == nil || !parseRule.GetIsEnabled() {
			continue
		}
		if parseErr = parseMatchServerToolCommandRule(parseRule, parseShell, parseCommandTokens, parseStart.GetCwd()); parseErr == nil {
			return nil
		}
	}
	return status.Error(codes.PermissionDenied, "run server tool command denied by policy")
}

// parseSplitServerToolCommandTokens parses one command string into argv tokens.
func parseSplitServerToolCommandTokens(parseCommand string) ([]string, error) {
	parseCommand = strings.TrimSpace(parseCommand)
	if parseCommand == "" {
		return nil, errors.New("run server tool command is required")
	}
	if strings.ContainsAny(parseCommand, "\r\n") {
		return nil, errors.New("run server tool command cannot contain newlines")
	}
	if strings.ContainsAny(parseCommand, "\"'`") {
		return nil, errors.New("run server tool command cannot contain quoted arguments")
	}
	parseCommandTokens := strings.Fields(parseCommand)
	if len(parseCommandTokens) == 0 {
		return nil, errors.New("run server tool command is required")
	}
	return parseCommandTokens, nil
}

// parseMatchServerToolCommandRule validates one command invocation against one rule definition.
func parseMatchServerToolCommandRule(parseRule *chatpb.ServerToolPolicyRule, parseShell string, parseCommandTokens []string, parseCWD string) error {
	if parseRule == nil || !parseRule.GetIsEnabled() {
		return errors.New("rule disabled")
	}
	parseRuleShell := parseBuildServerToolPolicyShell(parseRule.GetShell())
	if parseRuleShell != "auto" && parseShell != "auto" && parseRuleShell != parseShell {
		return errors.New("shell denied")
	}
	parseArgvPrefix := parseBuildServerToolPolicyStringList(parseRule.GetArgvPrefix())
	if len(parseArgvPrefix) == 0 {
		return errors.New("missing argv prefix")
	}
	if len(parseCommandTokens) < len(parseArgvPrefix) {
		return errors.New("command does not match argv prefix")
	}
	for parsePrefixIndex, parsePrefixToken := range parseArgvPrefix {
		if parseCommandTokens[parsePrefixIndex] != parsePrefixToken {
			return errors.New("command does not match argv prefix")
		}
	}
	parseCommandArgs := parseCommandTokens[len(parseArgvPrefix):]
	if parseErr := parseValidateServerToolCommandArgs(parseCommandArgs, parseRule.GetAllowArgsRegex(), parseRule.GetDenyArgsRegex()); parseErr != nil {
		return parseErr
	}
	if parseErr := parseValidateServerToolCWD(parseCWD, parseRule.GetAllowCwdPrefixes()); parseErr != nil {
		return parseErr
	}
	return nil
}

// parseValidateServerToolCommandArgs validates command args against allow and deny regex policies.
func parseValidateServerToolCommandArgs(parseArgs []string, parseAllowRegexValues []string, parseDenyRegexValues []string) error {
	parseAllowRegexValues = parseBuildServerToolPolicyStringList(parseAllowRegexValues)
	parseDenyRegexValues = parseBuildServerToolPolicyStringList(parseDenyRegexValues)
	if len(parseAllowRegexValues)+len(parseDenyRegexValues) == 0 {
		return errors.New("missing argument policy regex")
	}
	parseAllowRegex, parseErr := parseCompileServerToolRegexList(parseAllowRegexValues)
	if parseErr != nil {
		return parseErr
	}
	parseDenyRegex, parseErr := parseCompileServerToolRegexList(parseDenyRegexValues)
	if parseErr != nil {
		return parseErr
	}
	for _, parseArg := range parseArgs {
		if len(parseAllowRegex) > 0 && !parseMatchServerToolRegexList(parseAllowRegex, parseArg) {
			return errors.New("argument denied by allow_args_regex")
		}
		if parseMatchServerToolRegexList(parseDenyRegex, parseArg) {
			return errors.New("argument denied by deny_args_regex")
		}
	}
	return nil
}

// parseCompileServerToolRegexList compiles one regex-pattern list.
func parseCompileServerToolRegexList(parseRegexValues []string) ([]*regexp.Regexp, error) {
	parseRegexList := make([]*regexp.Regexp, 0, len(parseRegexValues))
	for _, parseRegexValue := range parseRegexValues {
		parseRegex, parseErr := regexp.Compile(parseRegexValue)
		if parseErr != nil {
			return nil, parseErr
		}
		parseRegexList = append(parseRegexList, parseRegex)
	}
	return parseRegexList, nil
}

// parseMatchServerToolRegexList reports whether any compiled regex matches one input value.
func parseMatchServerToolRegexList(parseRegexList []*regexp.Regexp, parseValue string) bool {
	for _, parseRegex := range parseRegexList {
		if parseRegex != nil && parseRegex.MatchString(parseValue) {
			return true
		}
	}
	return false
}

// parseValidateServerToolCWD validates one requested working directory against cwd-prefix policy.
func parseValidateServerToolCWD(parseCWD string, parseAllowCWDPrefixes []string) error {
	parseAllowCWDPrefixes = parseBuildServerToolPolicyStringList(parseAllowCWDPrefixes)
	if len(parseAllowCWDPrefixes) == 0 {
		return nil
	}
	parseCWD = strings.TrimSpace(parseCWD)
	if parseCWD == "" {
		return errors.New("working directory is required by policy")
	}
	parseNormalizedCWD := parseNormalizeServerToolPath(parseCWD)
	for _, parseAllowPrefix := range parseAllowCWDPrefixes {
		parseNormalizedAllowPrefix := parseNormalizeServerToolPath(parseAllowPrefix)
		if parseNormalizedAllowPrefix == "." {
			return nil
		}
		if parseNormalizedAllowPrefix != "" && strings.HasPrefix(parseNormalizedCWD, parseNormalizedAllowPrefix) {
			return nil
		}
	}
	return errors.New("working directory denied by allow_cwd_prefixes")
}

// parseNormalizeServerToolPath normalizes one path for case-insensitive prefix comparisons.
func parseNormalizeServerToolPath(parsePath string) string {
	parsePath = strings.TrimSpace(parsePath)
	parsePath = strings.ReplaceAll(parsePath, "\\", "/")
	parsePath = strings.TrimRight(parsePath, "/")
	return strings.ToLower(parsePath)
}
