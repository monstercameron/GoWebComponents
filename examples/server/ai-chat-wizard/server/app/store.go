package app

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	// Pure-Go SQLite via embedded WebAssembly (wazero). No CGo, no modernc/libc.
	"github.com/google/uuid"
	"github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/server/provider"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

var errUserAlreadyExists = errors.New("user already exists")
var errStoreUserMissing = errors.New("store user missing")
var errStoreConversationMissing = errors.New("store conversation missing")
var errStoreUserDisabled = errors.New("store user disabled")
var errStoreUserAuthBlocked = errors.New("store user auth blocked")
var errStoreWorkspaceSuspended = errors.New("store workspace suspended")
var newConversationPublicID = func() string { return uuid.NewString() }

// Store wraps a SQLite database for chat history persistence.
type Store struct {
	db      *sql.DB
	queries storeQueries
}

type parseUsageEventWrite struct {
	EventID                 string
	UserID                  int64
	ConversationID          int64
	ProviderID              string
	ModelID                 string
	PromptTokens            int64
	CompletionTokens        int64
	UsageSource             string
	ProviderRequestID       string
	InputCostPerMillionUSD  float64
	OutputCostPerMillionUSD float64
	PricingCurrency         string
	InputCostUSD            float64
	OutputCostUSD           float64
	TotalCostUSD            float64
	ClientID                string
	TraceID                 string
	SpanID                  string
	TraceState              string
	Status                  string
	ErrorMessage            string
}

type parseUsageEventRow struct {
	EventID                 string
	ConversationID          int64
	ProviderID              string
	ModelID                 string
	PromptTokens            int64
	CompletionTokens        int64
	UsageSource             string
	ProviderRequestID       string
	InputCostPerMillionUSD  float64
	OutputCostPerMillionUSD float64
	PricingCurrency         string
	InputCostUSD            float64
	OutputCostUSD           float64
	TotalCostUSD            float64
	ClientID                string
	TraceID                 string
	SpanID                  string
	TraceState              string
	Status                  string
	ErrorMessage            string
	CreatedAt               string
}

type authUserRecord struct {
	ID           int64
	Email        string
	PasswordHash string
}

const maxDBOpenConns = 8

// parseOpenChatStore opens or creates the SQLite database at path and initialises the schema.
func parseOpenChatStore(parsePath string) (*Store, error) {
	return parseOpenChatStoreWithRecovery(parsePath, true)
}

func parseOpenChatStoreWithRecovery(parsePath string, isAllowRecovery bool) (*Store, error) {
	parseQueries, parseErr := parseLoadStoreQueries()
	if parseErr != nil {
		return nil, parseErr
	}
	parseDbDir := strings.TrimSpace(filepath.Dir(parsePath))
	if parseDbDir != "" && parseDbDir != "." {
		if parseErr2 := os.MkdirAll(parseDbDir, 0o755); parseErr2 != nil {
			return nil, fmt.Errorf("create db directory %q: %w", parseDbDir, parseErr2)
		}
	}
	parseDsn := "file:" + parsePath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	parseDb, parseErr := sql.Open("sqlite3", parseDsn)
	if parseErr != nil {
		return nil, parseErr
	}
	parseMaxOpenConns := maxDBOpenConns
	if parseCpuCount := runtime.GOMAXPROCS(0); parseCpuCount > 0 && parseCpuCount < parseMaxOpenConns {
		parseMaxOpenConns = parseCpuCount
	}
	if parseMaxOpenConns < 4 {
		parseMaxOpenConns = 4
	}
	parseDb.SetMaxOpenConns(parseMaxOpenConns)
	parseDb.SetMaxIdleConns(parseMaxOpenConns)
	parseDb.SetConnMaxIdleTime(5 * time.Minute)
	if _, parseErr3 := parseDb.Exec(parseQueries.schema); parseErr3 != nil {
		_ = parseDb.Close()
		if isAllowRecovery && shouldResetIncompatibleStore(parsePath, parseErr3) {
			if parseBackupPath, parseRecoveryErr := parseBackupIncompatibleStoreFiles(parsePath); parseRecoveryErr == nil {
				return parseOpenChatStoreWithRecovery(parsePath, false)
			} else {
				return nil, fmt.Errorf("backup incompatible store %q: %w", parseBackupPath, parseRecoveryErr)
			}
		}
		return nil, parseErr3
	}
	parseRunBestEffortStatements(parseDb, parseQueries.migrations)
	if parseErr4 := parseEnsureConversationPublicIDs(parseDb); parseErr4 != nil {
		_ = parseDb.Close()
		return nil, parseErr4
	}
	return &Store{db: parseDb, queries: parseQueries}, nil
}

func parseEnsureConversationPublicIDs(parseDb *sql.DB) error {
	parseRows, parseErr := parseDb.Query(`SELECT id FROM conversations WHERE TRIM(COALESCE(public_id, '')) = '' ORDER BY id`)
	if parseErr != nil {
		return parseErr
	}
	defer parseRows.Close()

	var parseConversationIDs []int64
	for parseRows.Next() {
		var parseConversationID int64
		if parseErr2 := parseRows.Scan(&parseConversationID); parseErr2 != nil {
			return parseErr2
		}
		parseConversationIDs = append(parseConversationIDs, parseConversationID)
	}
	if parseErr3 := parseRows.Err(); parseErr3 != nil {
		return parseErr3
	}

	for _, parseConversationID2 := range parseConversationIDs {
		if parseErr4 := parseAssignConversationPublicID(parseDb, parseConversationID2); parseErr4 != nil {
			return parseErr4
		}
	}

	_, parseErr = parseDb.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_conversations_public_id ON conversations(public_id)`)
	return parseErr
}

func parseAssignConversationPublicID(parseDb *sql.DB, parseConversationID int64) error {
	for range 8 {
		parsePublicID := strings.TrimSpace(newConversationPublicID())
		if parsePublicID == "" {
			continue
		}
		if _, parseErr := parseDb.Exec(`UPDATE conversations SET public_id = ? WHERE id = ? AND TRIM(COALESCE(public_id, '')) = ''`, parsePublicID, parseConversationID); parseErr != nil {
			if isConversationPublicIDConflict(parseErr) {
				continue
			}
			return parseErr
		}
		return nil
	}
	return fmt.Errorf("assign public_id for conversation %d: exhausted retries", parseConversationID)
}

func isConversationPublicIDConflict(parseErr error) bool {
	if parseErr == nil {
		return false
	}
	parseLowered := strings.ToLower(parseErr.Error())
	return strings.Contains(parseLowered, "unique") && strings.Contains(parseLowered, "public_id")
}

func shouldResetIncompatibleStore(parsePath string, parseErr error) bool {
	if strings.TrimSpace(parsePath) == "" || parseErr == nil {
		return false
	}
	if _, parseStatErr := os.Stat(parsePath); parseStatErr != nil {
		return false
	}
	parseLowered := strings.ToLower(parseErr.Error())
	return strings.Contains(parseLowered, "no such column") ||
		strings.Contains(parseLowered, "has no column named") ||
		strings.Contains(parseLowered, "table ") ||
		strings.Contains(parseLowered, "malformed")
}

func parseBackupIncompatibleStoreFiles(parsePath string) (string, error) {
	parseSuffix := ".incompatible-" + time.Now().UTC().Format("20060102T150405") + ".bak"
	parseBackupPath := parsePath + parseSuffix
	if parseErr := os.MkdirAll(filepath.Dir(parsePath), 0o755); parseErr != nil {
		return parseBackupPath, parseErr
	}
	if parseErr2 := os.Rename(parsePath, parseBackupPath); parseErr2 != nil {
		return parseBackupPath, parseErr2
	}
	for _, parseExt := range []string{"-wal", "-shm"} {
		parseSrc := parsePath + parseExt
		if _, parseErr3 := os.Stat(parseSrc); parseErr3 == nil {
			_ = os.Rename(parseSrc, parseBackupPath+parseExt)
		}
	}
	return parseBackupPath, nil
}

func parseRunBestEffortStatements(parseDb *sql.DB, parseStatements string) {
	for parseRawStatement := range strings.SplitSeq(parseStatements, ";") {
		parseStatement := strings.TrimSpace(parseRawStatement)
		if parseStatement == "" {
			continue
		}
		_, _ = parseDb.Exec(parseStatement)
	}
}

func (parseS *Store) parseClose() { _ = parseS.db.Close() }

func (parseS *Store) parseCreateUser(parseEmail, parsePasswordHash, parseDisplayName string) (int64, error) {
	parseTx, parseErr := parseS.db.Begin()
	if parseErr != nil {
		return 0, parseErr
	}
	defer func() {
		_ = parseTx.Rollback()
	}()

	parseNow := time.Now().UTC()
	parseInsertResult, parseErr := parseTx.Exec(
		parseS.queries.createUser,
		parseNormalizeAuthEmail(parseEmail), parsePasswordHash, parseNow.Format(time.RFC3339),
	)
	if parseErr != nil {
		if strings.Contains(strings.ToLower(parseErr.Error()), "unique") {
			return 0, errUserAlreadyExists
		}
		return 0, parseErr
	}
	parseUserID, parseErr := parseInsertResult.LastInsertId()
	if parseErr != nil {
		return 0, parseErr
	}
	parseResolvedName := strings.TrimSpace(parseDisplayName)
	if parseResolvedName == "" {
		parseResolvedName = parseDefaultDisplayNameFromEmail(parseEmail)
	}
	if _, parseErr2 := parseTx.Exec(
		parseS.queries.upsertUserProfileName,
		parseUserID, parseResolvedName, parseNow.Unix(),
	); parseErr2 != nil {
		return 0, parseErr2
	}
	if parseErr3 := parseTx.Commit(); parseErr3 != nil {
		return 0, parseErr3
	}
	return parseUserID, nil
}

func (parseS *Store) getUserAuthByEmail(parseEmail string) (authUserRecord, error) {
	parseRow := parseS.db.QueryRow(parseS.queries.getUserAuthByEmail, parseNormalizeAuthEmail(parseEmail))
	parseRecord := authUserRecord{}
	if parseErr := parseRow.Scan(&parseRecord.ID, &parseRecord.Email, &parseRecord.PasswordHash); parseErr != nil {
		return authUserRecord{}, parseErr
	}
	parseRecord.Email = parseNormalizeAuthEmail(parseRecord.Email)
	return parseRecord, nil
}

func (parseS *Store) parseUserExists(parseUserID int64) (bool, error) {
	parseRow := parseS.db.QueryRow(parseS.queries.userExists, parseUserID)
	var parseExists int
	if parseErr := parseRow.Scan(&parseExists); parseErr != nil {
		return false, parseErr
	}
	return parseExists != 0, nil
}

func (parseS *Store) parseConversationOwnedByUser(parseUserID, parseConversationID int64) (bool, error) {
	parseRow := parseS.db.QueryRow(parseS.queries.conversationOwnedByUser, parseConversationID, parseUserID)
	var parseExists int
	if parseErr := parseRow.Scan(&parseExists); parseErr != nil {
		return false, parseErr
	}
	return parseExists != 0, nil
}

// resolveConversationRoute returns the owner-scoped internal conversation id for
// one public UUID. This is the server authority for browser routes; future
// shared-thread visibility should extend this resolver instead of bypassing it.
func (parseS *Store) parseResolveConversationRoute(parseUserID int64, parsePublicID string) (conversationSummaryRow, bool, error) {
	parsePublicID = strings.TrimSpace(parsePublicID)
	if parsePublicID == "" {
		return conversationSummaryRow{}, false, nil
	}
	parseRow := parseS.db.QueryRow(parseS.queries.resolveConversationRoute, parseUserID, parsePublicID)
	var parseSummary conversationSummaryRow
	if parseErr := parseRow.Scan(&parseSummary.ID, &parseSummary.PublicID); parseErr != nil {
		if errors.Is(parseErr, sql.ErrNoRows) {
			return conversationSummaryRow{}, false, nil
		}
		return conversationSummaryRow{}, false, parseErr
	}
	return parseSummary, true, nil
}

// createConversation inserts a new conversation row and returns its id.
func (parseS *Store) parseCreateConversation(parseUserID int64) (int64, error) {
	for range 8 {
		parsePublicID := strings.TrimSpace(newConversationPublicID())
		if parsePublicID == "" {
			continue
		}
		parseInsertResult, parseErr := parseS.db.Exec(
			parseS.queries.createConversation,
			parseUserID, parsePublicID, time.Now().UTC().Format(time.RFC3339), parseUserID,
		)
		if parseErr != nil {
			if isConversationPublicIDConflict(parseErr) {
				continue
			}
			return 0, parseErr
		}
		parseRowsAffected, parseErr := parseInsertResult.RowsAffected()
		if parseErr == nil && parseRowsAffected == 0 {
			return 0, errStoreUserMissing
		}
		return parseInsertResult.LastInsertId()
	}
	return 0, errors.New("create conversation: exhausted public id retries")
}

// saveConversationMessage appends a message to the given conversation.
func (parseS *Store) parseSaveConversationMessage(parseUserID, parseConversationID int64, parseRole, parseContent, parseModelID string, parsePromptTokens, parseCompletionTokens int64) error {
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.saveConversationMessage,
		parseConversationID, parseRole, parseContent, parseModelID, parsePromptTokens, parseCompletionTokens, time.Now().UTC().Format(time.RFC3339), parseConversationID, parseUserID,
	)
	if parseErr != nil {
		return parseErr
	}
	parseRowsAffected, parseErr := parseResult.RowsAffected()
	if parseErr == nil && parseRowsAffected == 0 {
		return errStoreConversationMissing
	}
	return nil
}

// parseSaveUsageEvent appends one immutable usage event row scoped to a user-owned conversation.
func (parseS *Store) parseSaveUsageEvent(parseUsage parseUsageEventWrite) error {
	if strings.TrimSpace(parseUsage.EventID) == "" {
		return errors.New("save usage event: event id is required")
	}
	parseResult, parseErr := parseS.db.Exec(
		parseS.queries.saveUsageEvent,
		parseUsage.EventID,
		parseUsage.UserID,
		parseUsage.ConversationID,
		parseUsage.ProviderID,
		parseUsage.ModelID,
		parseUsage.PromptTokens,
		parseUsage.CompletionTokens,
		parseUsage.UsageSource,
		parseUsage.ProviderRequestID,
		parseUsage.InputCostPerMillionUSD,
		parseUsage.OutputCostPerMillionUSD,
		parseUsage.PricingCurrency,
		parseUsage.InputCostUSD,
		parseUsage.OutputCostUSD,
		parseUsage.TotalCostUSD,
		parseUsage.ClientID,
		parseUsage.TraceID,
		parseUsage.SpanID,
		parseUsage.TraceState,
		parseUsage.Status,
		parseUsage.ErrorMessage,
		time.Now().UTC().Format(time.RFC3339),
		parseUsage.ConversationID,
		parseUsage.UserID,
	)
	if parseErr != nil {
		return parseErr
	}
	parseRowsAffected, parseErr := parseResult.RowsAffected()
	if parseErr == nil && parseRowsAffected == 0 {
		return errStoreConversationMissing
	}
	return nil
}

// parseListUsageEvents returns recent usage events for one authenticated user.
func (parseS *Store) parseListUsageEvents(parseUserID int64, parseLimit int64) ([]parseUsageEventRow, error) {
	if parseLimit <= 0 {
		parseLimit = 100
	}
	parseRows, parseErr := parseS.db.Query(parseS.queries.listUsageEvents, parseUserID, parseLimit)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseEvents := make([]parseUsageEventRow, 0)
	for parseRows.Next() {
		var parseRow parseUsageEventRow
		if parseErr2 := parseRows.Scan(
			&parseRow.EventID,
			&parseRow.ConversationID,
			&parseRow.ProviderID,
			&parseRow.ModelID,
			&parseRow.PromptTokens,
			&parseRow.CompletionTokens,
			&parseRow.UsageSource,
			&parseRow.ProviderRequestID,
			&parseRow.InputCostPerMillionUSD,
			&parseRow.OutputCostPerMillionUSD,
			&parseRow.PricingCurrency,
			&parseRow.InputCostUSD,
			&parseRow.OutputCostUSD,
			&parseRow.TotalCostUSD,
			&parseRow.ClientID,
			&parseRow.TraceID,
			&parseRow.SpanID,
			&parseRow.TraceState,
			&parseRow.Status,
			&parseRow.ErrorMessage,
			&parseRow.CreatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseEvents = append(parseEvents, parseRow)
	}
	return parseEvents, parseRows.Err()
}

// parseSumUsageTokensSince returns one user's total tokens since one UTC timestamp.
func (parseS *Store) parseSumUsageTokensSince(parseUserID int64, parseSince time.Time) (int64, error) {
	parseRow := parseS.db.QueryRow(
		parseS.queries.sumUsageTokensSince,
		parseUserID,
		parseSince.UTC().Format(time.RFC3339),
	)
	var parseTotalTokens int64
	if parseErr := parseRow.Scan(&parseTotalTokens); parseErr != nil {
		return 0, parseErr
	}
	return parseTotalTokens, nil
}

// parseGetModelPricingForModel resolves provider pricing metadata for one catalog model.
func (parseS *Store) parseGetModelPricingForModel(parseModelID string) (string, provider.ModelPricing, bool, error) {
	parseRow := parseS.db.QueryRow(parseS.queries.getModelPricing, strings.TrimSpace(parseModelID))
	var parseProviderID string
	var parsePricing provider.ModelPricing
	if parseErr := parseRow.Scan(&parseProviderID, &parsePricing.InputPerMillionUSD, &parsePricing.OutputPerMillionUSD, &parsePricing.Currency); parseErr != nil {
		if errors.Is(parseErr, sql.ErrNoRows) {
			return "", provider.ModelPricing{}, false, nil
		}
		return "", provider.ModelPricing{}, false, parseErr
	}
	return strings.TrimSpace(parseProviderID), parsePricing, true, nil
}

// parseSaveConversationTitle persists one conversation title for one user-owned conversation.
func (parseS *Store) parseSaveConversationTitle(parseUserID, parseConversationID int64, parseTitle string) error {
	parseResult, parseErr := parseS.db.Exec(parseS.queries.saveConversationTitle, parseTitle, parseConversationID, parseUserID)
	if parseErr != nil {
		return parseErr
	}
	parseRowsAffected, parseErr := parseResult.RowsAffected()
	if parseErr == nil && parseRowsAffected == 0 {
		return errStoreConversationMissing
	}
	return nil
}

// conversationSummaryRow is a lightweight view of a conversation for the sidebar list.
type conversationSummaryRow struct {
	ID        int64
	PublicID  string
	StartedAt string
	Preview   string
}

// listConversations returns all conversations for one user ordered newest-first.
func (parseS *Store) parseListConversations(parseUserID int64) ([]conversationSummaryRow, error) {
	parseRows, parseErr := parseS.db.Query(parseS.queries.listConversations, parseUserID)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()
	var parseConversationSummaries []conversationSummaryRow
	for parseRows.Next() {
		var parseConversationSummary conversationSummaryRow
		if parseErr2 := parseRows.Scan(&parseConversationSummary.ID, &parseConversationSummary.PublicID, &parseConversationSummary.StartedAt, &parseConversationSummary.Preview); parseErr2 != nil {
			return nil, parseErr2
		}
		parseConversationSummaries = append(parseConversationSummaries, parseConversationSummary)
	}
	return parseConversationSummaries, parseRows.Err()
}

// loadConversation returns all messages for one user-owned conversation in order.
func (parseS *Store) parseLoadConversation(parseUserID, parseConversationID int64) ([]struct {
	Role             string
	Content          string
	ModelID          string
	PromptTokens     int64
	CompletionTokens int64
}, error) {
	parseRows, parseErr := parseS.db.Query(
		parseS.queries.loadConversation, parseConversationID, parseUserID)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()
	var parseConversationMessages []struct {
		Role             string
		Content          string
		ModelID          string
		PromptTokens     int64
		CompletionTokens int64
	}
	for parseRows.Next() {
		var parseMessageRow struct {
			Role             string
			Content          string
			ModelID          string
			PromptTokens     int64
			CompletionTokens int64
		}
		if parseErr2 := parseRows.Scan(&parseMessageRow.Role, &parseMessageRow.Content, &parseMessageRow.ModelID, &parseMessageRow.PromptTokens, &parseMessageRow.CompletionTokens); parseErr2 != nil {
			return nil, parseErr2
		}
		parseConversationMessages = append(parseConversationMessages, parseMessageRow)
	}
	return parseConversationMessages, parseRows.Err()
}

// deleteConversation removes one user-owned conversation and all its messages.
func (parseS *Store) parseDeleteConversation(parseUserID, parseConversationID int64) error {
	parseTx, parseErr := parseS.db.Begin()
	if parseErr != nil {
		return parseErr
	}
	defer func() {
		_ = parseTx.Rollback()
	}()

	if _, parseErr2 := parseTx.Exec(
		parseS.queries.deleteConversationMessages,
		parseConversationID, parseUserID,
	); parseErr2 != nil {
		return parseErr2
	}
	if _, parseErr3 := parseTx.Exec(
		parseS.queries.deleteConversationUsageEvents,
		parseConversationID, parseUserID,
	); parseErr3 != nil {
		return parseErr3
	}
	if _, parseErr4 := parseTx.Exec(parseS.queries.deleteConversation, parseConversationID, parseUserID); parseErr4 != nil {
		return parseErr4
	}
	return parseTx.Commit()
}

// setUserName upserts one user's profile row.
func (parseS *Store) setUserName(parseUserID int64, parseName string, parseUpdatedAt int64) error {
	_, parseErr := parseS.db.Exec(
		parseS.queries.upsertUserProfileName,
		parseUserID, parseName, parseUpdatedAt,
	)
	return parseErr
}

func (parseS *Store) getUserName(parseUserID int64) (parseName string, parseUpdatedAt int64, parseErr error) {
	parseRow := parseS.db.QueryRow(parseS.queries.getUserName, parseUserID)
	if parseScanErr := parseRow.Scan(&parseName, &parseUpdatedAt); parseScanErr != nil {
		if errors.Is(parseScanErr, sql.ErrNoRows) {
			return "User", 0, nil
		}
		return "", 0, parseScanErr
	}
	return parseName, parseUpdatedAt, nil
}

func (parseS *Store) setSelectedModel(parseUserID int64, parseModel string) error {
	_, parseErr := parseS.db.Exec(
		parseS.queries.setSelectedModel,
		parseUserID, parseModel,
	)
	return parseErr
}

func (parseS *Store) getSelectedModel(parseUserID int64, parseFallback string) (string, error) {
	parseRow := parseS.db.QueryRow(parseS.queries.getSelectedModel, parseUserID)
	var parseSelectedModel string
	if parseScanErr := parseRow.Scan(&parseSelectedModel); parseScanErr != nil {
		if errors.Is(parseScanErr, sql.ErrNoRows) {
			return parseFallback, nil
		}
		return "", parseScanErr
	}
	if parseSelectedModel == "" {
		return parseFallback, nil
	}
	return parseSelectedModel, nil
}

func (parseS *Store) setSelectedTone(parseUserID int64, parseTone string) error {
	_, parseErr := parseS.db.Exec(
		parseS.queries.setSelectedTone,
		parseUserID, parseTone,
	)
	return parseErr
}

func (parseS *Store) getSelectedTone(parseUserID int64, parseFallback string) (string, error) {
	parseRow := parseS.db.QueryRow(parseS.queries.getSelectedTone, parseUserID)
	var parseSelectedTone string
	if parseScanErr := parseRow.Scan(&parseSelectedTone); parseScanErr != nil {
		if errors.Is(parseScanErr, sql.ErrNoRows) {
			return parseFallback, nil
		}
		return "", parseScanErr
	}
	if parseSelectedTone == "" {
		return parseFallback, nil
	}
	return parseSelectedTone, nil
}

func (parseS *Store) setSelectedThinkingEnabled(parseUserID int64, isEnabled bool) error {
	parseEnabledValue := 0
	if isEnabled {
		parseEnabledValue = 1
	}
	_, parseErr := parseS.db.Exec(
		parseS.queries.setSelectedThinkingEnabled,
		parseUserID, parseEnabledValue,
	)
	return parseErr
}

func (parseS *Store) getSelectedThinkingEnabled(parseUserID int64, isFallback bool) (bool, error) {
	parseRow := parseS.db.QueryRow(parseS.queries.getSelectedThinkingEnabled, parseUserID)
	var parseEnabledValue int64
	if parseScanErr := parseRow.Scan(&parseEnabledValue); parseScanErr != nil {
		if errors.Is(parseScanErr, sql.ErrNoRows) {
			return isFallback, nil
		}
		return false, parseScanErr
	}
	return parseEnabledValue != 0, nil
}

func (parseS *Store) setSelectedThinkingEffort(parseUserID int64, parseEffort string) error {
	_, parseErr := parseS.db.Exec(
		parseS.queries.setSelectedThinkingEffort,
		parseUserID, parseEffort,
	)
	return parseErr
}

func (parseS *Store) getSelectedThinkingEffort(parseUserID int64, parseFallback string) (string, error) {
	parseRow := parseS.db.QueryRow(parseS.queries.getSelectedThinkingEffort, parseUserID)
	var parseEffort string
	if parseScanErr := parseRow.Scan(&parseEffort); parseScanErr != nil {
		if errors.Is(parseScanErr, sql.ErrNoRows) {
			return parseFallback, nil
		}
		return "", parseScanErr
	}
	if parseEffort == "" {
		return parseFallback, nil
	}
	return parseEffort, nil
}

func (parseS *Store) setSelectedSystemPrompt(parseUserID int64, parsePrompt string) error {
	_, parseErr := parseS.db.Exec(
		parseS.queries.setSelectedSystemPrompt,
		parseUserID, parsePrompt,
	)
	return parseErr
}

func (parseS *Store) getSelectedSystemPrompt(parseUserID int64, parseFallback string) (string, error) {
	parseRow := parseS.db.QueryRow(parseS.queries.getSelectedSystemPrompt, parseUserID)
	var parsePrompt string
	if parseScanErr := parseRow.Scan(&parsePrompt); parseScanErr != nil {
		if errors.Is(parseScanErr, sql.ErrNoRows) {
			return parseFallback, nil
		}
		return "", parseScanErr
	}
	if parsePrompt == "" {
		return parseFallback, nil
	}
	return parsePrompt, nil
}

type userMemoryRow struct {
	Key             string
	Category        string
	Summary         string
	Detail          string
	SourceMessage   string
	UsefulnessScore int
	ConfidenceScore float64
	RubricReason    string
	UpdatedAt       string
}

func (parseS *Store) parseUpsertUserMemory(parseUserID int64, parseMemory userMemoryRow) error {
	parseNow := time.Now().UTC().Format(time.RFC3339)
	_, parseErr := parseS.db.Exec(
		parseS.queries.upsertUserMemory,
		parseUserID,
		parseMemory.Key,
		parseMemory.Category,
		parseMemory.Summary,
		parseMemory.Detail,
		parseMemory.SourceMessage,
		parseMemory.UsefulnessScore,
		parseMemory.ConfidenceScore,
		parseMemory.RubricReason,
		parseNow,
		parseNow,
	)
	return parseErr
}

func (parseS *Store) parseListUserMemories(parseUserID int64) ([]userMemoryRow, error) {
	parseRows, parseErr := parseS.db.Query(parseS.queries.listUserMemories, parseUserID)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseMemories := make([]userMemoryRow, 0)
	for parseRows.Next() {
		var parseMemory userMemoryRow
		if parseErr2 := parseRows.Scan(
			&parseMemory.Key,
			&parseMemory.Category,
			&parseMemory.Summary,
			&parseMemory.Detail,
			&parseMemory.SourceMessage,
			&parseMemory.UsefulnessScore,
			&parseMemory.ConfidenceScore,
			&parseMemory.RubricReason,
			&parseMemory.UpdatedAt,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseMemories = append(parseMemories, parseMemory)
	}
	return parseMemories, parseRows.Err()
}

func (parseS *Store) parseDeleteUserMemory(parseUserID int64, parseKey string) error {
	_, parseErr := parseS.db.Exec(parseS.queries.deleteUserMemory, parseUserID, strings.TrimSpace(parseKey))
	return parseErr
}

func (parseS *Store) parseListModelCatalog() ([]modelCatalogRow, error) {
	parseRows, parseErr := parseS.db.Query(parseS.queries.listModelCatalog)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseRows.Close()

	parseCatalog := make([]modelCatalogRow, 0)
	for parseRows.Next() {
		var parseRow modelCatalogRow
		var parseSupportsThinking int64
		var parseSupportsSpeech int64
		var parseOnboardingReady int64
		var parseIsDefault int64
		var parseUseForTitleGeneration int64
		var parseUseForMemoryExtraction int64
		if parseErr2 := parseRows.Scan(
			&parseRow.ID,
			&parseRow.ProviderID,
			&parseRow.ProviderLabel,
			&parseRow.Label,
			&parseRow.Note,
			&parseRow.Description,
			&parseSupportsThinking,
			&parseSupportsSpeech,
			&parseRow.InputPerMillionUSD,
			&parseRow.OutputPerMillionUSD,
			&parseRow.PricingCurrency,
			&parseRow.MaxOutputTokens,
			&parseRow.ThroughputTokensPerSec,
			&parseOnboardingReady,
			&parseIsDefault,
			&parseUseForTitleGeneration,
			&parseUseForMemoryExtraction,
			&parseRow.SortOrder,
		); parseErr2 != nil {
			return nil, parseErr2
		}
		parseRow.SupportsThinking = parseSupportsThinking != 0
		parseRow.SupportsSpeech = parseSupportsSpeech != 0
		parseRow.OnboardingReady = parseOnboardingReady != 0
		parseRow.IsDefault = parseIsDefault != 0
		parseRow.UseForTitleGeneration = parseUseForTitleGeneration != 0
		parseRow.UseForMemoryExtraction = parseUseForMemoryExtraction != 0
		parseCatalog = append(parseCatalog, parseRow)
	}
	return parseCatalog, parseRows.Err()
}
