package app

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"
)

type parseServerToolPolicyStoreWrite struct {
	IsEnabled         bool
	MaxSessionSeconds int32
	MaxOutputBytes    int32
	ApprovedToolsJSON string
	UpdatedByUserID   int64
	Source            string
}

type parseServerToolPolicyHistoryRow struct {
	ID                int64
	IsEnabled         bool
	MaxSessionSeconds int32
	MaxOutputBytes    int32
	ApprovedToolsJSON string
	UpdatedByUserID   int64
	Source            string
	CreatedAt         string
}

// parseSetServerToolPolicy persists one complete server-tool policy snapshot and appends one immutable history row.
func (parseS *Store) parseSetServerToolPolicy(parseWrite parseServerToolPolicyStoreWrite) (parseServerToolPolicyHistoryRow, error) {
	if parseS == nil || parseS.db == nil {
		return parseServerToolPolicyHistoryRow{}, errors.New("set server tool policy: store is required")
	}
	if parseS.queries.upsertSiteConfig == "" {
		return parseServerToolPolicyHistoryRow{}, errors.New("set server tool policy: upsert site config query missing")
	}
	if parseS.queries.createServerToolPolicyHistory == "" {
		return parseServerToolPolicyHistoryRow{}, errors.New("set server tool policy: create history query missing")
	}
	if parseWrite.UpdatedByUserID <= 0 {
		return parseServerToolPolicyHistoryRow{}, errors.New("set server tool policy: updated by user id is required")
	}

	parseNormalized := parseServerToolPolicyStoreWrite{
		IsEnabled:         parseWrite.IsEnabled,
		MaxSessionSeconds: parseClampServerToolPolicyInt32(parseWrite.MaxSessionSeconds, serverToolPolicyMinimumMaxSessionSeconds, serverToolPolicyMaximumMaxSessionSeconds),
		MaxOutputBytes:    parseClampServerToolPolicyInt32(parseWrite.MaxOutputBytes, serverToolPolicyMinimumMaxOutputBytes, serverToolPolicyMaximumMaxOutputBytes),
		ApprovedToolsJSON: strings.TrimSpace(parseWrite.ApprovedToolsJSON),
		UpdatedByUserID:   parseWrite.UpdatedByUserID,
		Source:            strings.TrimSpace(parseWrite.Source),
	}
	if parseNormalized.MaxSessionSeconds == 0 {
		parseNormalized.MaxSessionSeconds = serverToolPolicyDefaultMaxSessionSeconds
	}
	if parseNormalized.MaxOutputBytes == 0 {
		parseNormalized.MaxOutputBytes = serverToolPolicyDefaultMaxOutputBytes
	}
	if parseNormalized.ApprovedToolsJSON == "" {
		parseNormalized.ApprovedToolsJSON = "[]"
	}
	if parseNormalized.Source == "" {
		parseNormalized.Source = "set_server_tool_policy"
	}

	parseNow := time.Now().UTC().Format(time.RFC3339)
	parseTx, parseErr := parseS.db.Begin()
	if parseErr != nil {
		return parseServerToolPolicyHistoryRow{}, parseErr
	}
	defer parseRollbackIfNeeded(parseTx)

	if parseErr = parseSetServerToolPolicyConfigRow(parseTx, parseS.queries.upsertSiteConfig, parseSiteConfigWrite{
		ConfigKey:       serverToolPolicyConfigEnabledKey,
		ConfigValue:     parseBuildServerToolPolicyBoolValue(parseNormalized.IsEnabled),
		ValueType:       "bool",
		Description:     "Enable or disable server tool execution.",
		UpdatedByUserID: parseNormalized.UpdatedByUserID,
	}, parseNow); parseErr != nil {
		return parseServerToolPolicyHistoryRow{}, parseErr
	}
	if parseErr = parseSetServerToolPolicyConfigRow(parseTx, parseS.queries.upsertSiteConfig, parseSiteConfigWrite{
		ConfigKey:       serverToolPolicyConfigMaxSessionSecondsKey,
		ConfigValue:     parseBuildServerToolPolicyIntValue(parseNormalized.MaxSessionSeconds),
		ValueType:       "int",
		Description:     "Maximum server tool runtime per session in seconds.",
		UpdatedByUserID: parseNormalized.UpdatedByUserID,
	}, parseNow); parseErr != nil {
		return parseServerToolPolicyHistoryRow{}, parseErr
	}
	if parseErr = parseSetServerToolPolicyConfigRow(parseTx, parseS.queries.upsertSiteConfig, parseSiteConfigWrite{
		ConfigKey:       serverToolPolicyConfigMaxOutputBytesKey,
		ConfigValue:     parseBuildServerToolPolicyIntValue(parseNormalized.MaxOutputBytes),
		ValueType:       "int",
		Description:     "Maximum combined stdout/stderr bytes per server tool session.",
		UpdatedByUserID: parseNormalized.UpdatedByUserID,
	}, parseNow); parseErr != nil {
		return parseServerToolPolicyHistoryRow{}, parseErr
	}
	if parseErr = parseSetServerToolPolicyConfigRow(parseTx, parseS.queries.upsertSiteConfig, parseSiteConfigWrite{
		ConfigKey:       serverToolPolicyConfigWhitelistKey,
		ConfigValue:     parseNormalized.ApprovedToolsJSON,
		ValueType:       "json",
		Description:     "Approved server tool command policy and argument constraints.",
		UpdatedByUserID: parseNormalized.UpdatedByUserID,
	}, parseNow); parseErr != nil {
		return parseServerToolPolicyHistoryRow{}, parseErr
	}

	parseHistoryResult, parseErr := parseTx.Exec(
		parseS.queries.createServerToolPolicyHistory,
		parseBuildServerToolPolicyBoolBit(parseNormalized.IsEnabled),
		parseNormalized.MaxSessionSeconds,
		parseNormalized.MaxOutputBytes,
		parseNormalized.ApprovedToolsJSON,
		parseNormalized.UpdatedByUserID,
		parseNormalized.Source,
		parseNow,
	)
	if parseErr != nil {
		return parseServerToolPolicyHistoryRow{}, parseErr
	}
	parseHistoryID, parseErr := parseHistoryResult.LastInsertId()
	if parseErr != nil {
		return parseServerToolPolicyHistoryRow{}, parseErr
	}

	if parseErr = parseTx.Commit(); parseErr != nil {
		return parseServerToolPolicyHistoryRow{}, parseErr
	}

	return parseServerToolPolicyHistoryRow{
		ID:                parseHistoryID,
		IsEnabled:         parseNormalized.IsEnabled,
		MaxSessionSeconds: parseNormalized.MaxSessionSeconds,
		MaxOutputBytes:    parseNormalized.MaxOutputBytes,
		ApprovedToolsJSON: parseNormalized.ApprovedToolsJSON,
		UpdatedByUserID:   parseNormalized.UpdatedByUserID,
		Source:            parseNormalized.Source,
		CreatedAt:         parseNow,
	}, nil
}

// parseListServerToolPolicyHistory lists immutable server-tool policy history rows newest-first.
func (parseS *Store) parseListServerToolPolicyHistory(parseLimit int64) ([]parseServerToolPolicyHistoryRow, error) {
	if parseS == nil || parseS.db == nil {
		return nil, errors.New("list server tool policy history: store is required")
	}
	if parseS.queries.listServerToolPolicyHistory == "" {
		return nil, errors.New("list server tool policy history: query missing")
	}
	if parseLimit <= 0 {
		parseLimit = 50
	}
	if parseLimit > 500 {
		parseLimit = 500
	}

	parseRows, parseErr := parseS.db.Query(parseS.queries.listServerToolPolicyHistory, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseHistoryRows := make([]parseServerToolPolicyHistoryRow, 0)
	for parseRows.Next() {
		var parseRow parseServerToolPolicyHistoryRow
		var parseEnabledBit int64
		if parseErr = parseRows.Scan(
			&parseRow.ID,
			&parseEnabledBit,
			&parseRow.MaxSessionSeconds,
			&parseRow.MaxOutputBytes,
			&parseRow.ApprovedToolsJSON,
			&parseRow.UpdatedByUserID,
			&parseRow.Source,
			&parseRow.CreatedAt,
		); parseErr != nil {
			return nil, parseErr
		}
		parseRow.IsEnabled = parseEnabledBit != 0
		parseHistoryRows = append(parseHistoryRows, parseRow)
	}
	return parseHistoryRows, parseRows.Err()
}

// parseSetServerToolPolicyConfigRow writes one site-config row inside one existing transaction.
func parseSetServerToolPolicyConfigRow(parseTx *sql.Tx, parseQuery string, parseWrite parseSiteConfigWrite, parseUpdatedAt string) error {
	if parseTx == nil {
		return errors.New("set server tool policy config row: tx is required")
	}
	if strings.TrimSpace(parseQuery) == "" {
		return errors.New("set server tool policy config row: query is required")
	}
	_, parseErr := parseTx.Exec(
		parseQuery,
		parseNormalizeSUKey(parseWrite.ConfigKey),
		strings.TrimSpace(parseWrite.ConfigValue),
		parseNormalizeSUValue(parseWrite.ValueType, "string"),
		strings.TrimSpace(parseWrite.Description),
		parseWrite.UpdatedByUserID,
		strings.TrimSpace(parseUpdatedAt),
	)
	return parseErr
}

// parseBuildServerToolPolicyBoolValue formats one policy boolean as canonical site-config text.
func parseBuildServerToolPolicyBoolValue(isParseEnabled bool) string {
	if isParseEnabled {
		return "true"
	}
	return "false"
}

// parseBuildServerToolPolicyBoolBit formats one policy boolean as integer bit storage for policy history rows.
func parseBuildServerToolPolicyBoolBit(isParseEnabled bool) int64 {
	if isParseEnabled {
		return 1
	}
	return 0
}

// parseBuildServerToolPolicyIntValue formats one policy int32 value as canonical site-config text.
func parseBuildServerToolPolicyIntValue(parseValue int32) string {
	return strings.TrimSpace(strconv.FormatInt(int64(parseValue), 10))
}

// parseRollbackIfNeeded executes one best-effort rollback after transaction exits without commit.
func parseRollbackIfNeeded(parseTx *sql.Tx) {
	if parseTx == nil {
		return
	}
	_ = parseTx.Rollback()
}
