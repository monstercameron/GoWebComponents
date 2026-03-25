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
	_ "github.com/ncruces/go-sqlite3/driver"
)

var errUserAlreadyExists = errors.New("user already exists")
var errStoreUserMissing = errors.New("store user missing")
var errStoreConversationMissing = errors.New("store conversation missing")
var newConversationPublicID = func() string { return uuid.NewString() }

// Store wraps a SQLite database for chat history persistence.
type Store struct {
	db      *sql.DB
	queries storeQueries
}

type authUserRecord struct {
	ID           int64
	Email        string
	PasswordHash string
}

const maxDBOpenConns = 8

// openChatStore opens (or creates) the SQLite database at path and initialises the schema.
func openChatStore(path string) (*Store, error) {
	return openChatStoreWithRecovery(path, true)
}

func openChatStoreWithRecovery(path string, allowRecovery bool) (*Store, error) {
	queries, err := loadStoreQueries()
	if err != nil {
		return nil, err
	}
	dbDir := strings.TrimSpace(filepath.Dir(path))
	if dbDir != "" && dbDir != "." {
		if err := os.MkdirAll(dbDir, 0o755); err != nil {
			return nil, fmt.Errorf("create db directory %q: %w", dbDir, err)
		}
	}
	dsn := "file:" + path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	maxOpenConns := maxDBOpenConns
	if cpuCount := runtime.GOMAXPROCS(0); cpuCount > 0 && cpuCount < maxOpenConns {
		maxOpenConns = cpuCount
	}
	if maxOpenConns < 4 {
		maxOpenConns = 4
	}
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxOpenConns)
	db.SetConnMaxIdleTime(5 * time.Minute)
	if _, err := db.Exec(queries.schema); err != nil {
		_ = db.Close()
		if allowRecovery && shouldResetIncompatibleStore(path, err) {
			if backupPath, recoveryErr := backupIncompatibleStoreFiles(path); recoveryErr == nil {
				return openChatStoreWithRecovery(path, false)
			} else {
				return nil, fmt.Errorf("backup incompatible store %q: %w", backupPath, recoveryErr)
			}
		}
		return nil, err
	}
	runBestEffortStatements(db, queries.migrations)
	if err := ensureConversationPublicIDs(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db, queries: queries}, nil
}

func ensureConversationPublicIDs(db *sql.DB) error {
	rows, err := db.Query(`SELECT id FROM conversations WHERE TRIM(COALESCE(public_id, '')) = '' ORDER BY id`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var conversationIDs []int64
	for rows.Next() {
		var conversationID int64
		if err := rows.Scan(&conversationID); err != nil {
			return err
		}
		conversationIDs = append(conversationIDs, conversationID)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, conversationID := range conversationIDs {
		if err := assignConversationPublicID(db, conversationID); err != nil {
			return err
		}
	}

	_, err = db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_conversations_public_id ON conversations(public_id)`)
	return err
}

func assignConversationPublicID(db *sql.DB, conversationID int64) error {
	for attempts := 0; attempts < 8; attempts++ {
		publicID := strings.TrimSpace(newConversationPublicID())
		if publicID == "" {
			continue
		}
		if _, err := db.Exec(`UPDATE conversations SET public_id = ? WHERE id = ? AND TRIM(COALESCE(public_id, '')) = ''`, publicID, conversationID); err != nil {
			if isConversationPublicIDConflict(err) {
				continue
			}
			return err
		}
		return nil
	}
	return fmt.Errorf("assign public_id for conversation %d: exhausted retries", conversationID)
}

func isConversationPublicIDConflict(err error) bool {
	if err == nil {
		return false
	}
	lowered := strings.ToLower(err.Error())
	return strings.Contains(lowered, "unique") && strings.Contains(lowered, "public_id")
}

func shouldResetIncompatibleStore(path string, err error) bool {
	if strings.TrimSpace(path) == "" || err == nil {
		return false
	}
	if _, statErr := os.Stat(path); statErr != nil {
		return false
	}
	lowered := strings.ToLower(err.Error())
	return strings.Contains(lowered, "no such column") ||
		strings.Contains(lowered, "has no column named") ||
		strings.Contains(lowered, "table ") ||
		strings.Contains(lowered, "malformed")
}

func backupIncompatibleStoreFiles(path string) (string, error) {
	suffix := ".incompatible-" + time.Now().UTC().Format("20060102T150405") + ".bak"
	backupPath := path + suffix
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return backupPath, err
	}
	if err := os.Rename(path, backupPath); err != nil {
		return backupPath, err
	}
	for _, ext := range []string{"-wal", "-shm"} {
		src := path + ext
		if _, err := os.Stat(src); err == nil {
			_ = os.Rename(src, backupPath+ext)
		}
	}
	return backupPath, nil
}

func runBestEffortStatements(db *sql.DB, statements string) {
	for _, rawStatement := range strings.Split(statements, ";") {
		statement := strings.TrimSpace(rawStatement)
		if statement == "" {
			continue
		}
		_, _ = db.Exec(statement)
	}
}

func (s *Store) close() { _ = s.db.Close() }

func (s *Store) createUser(email, passwordHash, displayName string) (int64, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	now := time.Now().UTC()
	insertResult, err := tx.Exec(
		s.queries.createUser,
		normalizeAuthEmail(email), passwordHash, now.Format(time.RFC3339),
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return 0, errUserAlreadyExists
		}
		return 0, err
	}
	userID, err := insertResult.LastInsertId()
	if err != nil {
		return 0, err
	}
	resolvedName := strings.TrimSpace(displayName)
	if resolvedName == "" {
		resolvedName = defaultDisplayNameFromEmail(email)
	}
	if _, err := tx.Exec(
		s.queries.upsertUserProfileName,
		userID, resolvedName, now.Unix(),
	); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return userID, nil
}

func (s *Store) getUserAuthByEmail(email string) (authUserRecord, error) {
	row := s.db.QueryRow(s.queries.getUserAuthByEmail, normalizeAuthEmail(email))
	record := authUserRecord{}
	if err := row.Scan(&record.ID, &record.Email, &record.PasswordHash); err != nil {
		return authUserRecord{}, err
	}
	record.Email = normalizeAuthEmail(record.Email)
	return record, nil
}

func (s *Store) userExists(userID int64) (bool, error) {
	row := s.db.QueryRow(s.queries.userExists, userID)
	var exists int
	if err := row.Scan(&exists); err != nil {
		return false, err
	}
	return exists != 0, nil
}

func (s *Store) conversationOwnedByUser(userID, conversationID int64) (bool, error) {
	row := s.db.QueryRow(s.queries.conversationOwnedByUser, conversationID, userID)
	var exists int
	if err := row.Scan(&exists); err != nil {
		return false, err
	}
	return exists != 0, nil
}

// resolveConversationRoute returns the owner-scoped internal conversation id for
// one public UUID. This is the server authority for browser routes; future
// shared-thread visibility should extend this resolver instead of bypassing it.
func (s *Store) resolveConversationRoute(userID int64, publicID string) (conversationSummaryRow, bool, error) {
	publicID = strings.TrimSpace(publicID)
	if publicID == "" {
		return conversationSummaryRow{}, false, nil
	}
	row := s.db.QueryRow(s.queries.resolveConversationRoute, userID, publicID)
	var summary conversationSummaryRow
	if err := row.Scan(&summary.ID, &summary.PublicID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return conversationSummaryRow{}, false, nil
		}
		return conversationSummaryRow{}, false, err
	}
	return summary, true, nil
}

// createConversation inserts a new conversation row and returns its id.
func (s *Store) createConversation(userID int64) (int64, error) {
	for attempts := 0; attempts < 8; attempts++ {
		publicID := strings.TrimSpace(newConversationPublicID())
		if publicID == "" {
			continue
		}
		insertResult, err := s.db.Exec(
			s.queries.createConversation,
			userID, publicID, time.Now().UTC().Format(time.RFC3339), userID,
		)
		if err != nil {
			if isConversationPublicIDConflict(err) {
				continue
			}
			return 0, err
		}
		rowsAffected, err := insertResult.RowsAffected()
		if err == nil && rowsAffected == 0 {
			return 0, errStoreUserMissing
		}
		return insertResult.LastInsertId()
	}
	return 0, errors.New("create conversation: exhausted public id retries")
}

// saveConversationMessage appends a message to the given conversation.
func (s *Store) saveConversationMessage(userID, conversationID int64, role, content, modelID string, promptTokens, completionTokens int64) error {
	result, err := s.db.Exec(
		s.queries.saveConversationMessage,
		conversationID, role, content, modelID, promptTokens, completionTokens, time.Now().UTC().Format(time.RFC3339), conversationID, userID,
	)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err == nil && rowsAffected == 0 {
		return errStoreConversationMissing
	}
	return nil
}

// saveConversationTitle persists an AI-generated title for a conversation.
func (s *Store) saveConversationTitle(userID, conversationID int64, title string) error {
	_, err := s.db.Exec(s.queries.saveConversationTitle, title, conversationID, userID)
	return err
}

// conversationSummaryRow is a lightweight view of a conversation for the sidebar list.
type conversationSummaryRow struct {
	ID        int64
	PublicID  string
	StartedAt string
	Preview   string
}

// listConversations returns all conversations for one user ordered newest-first.
func (s *Store) listConversations(userID int64) ([]conversationSummaryRow, error) {
	rows, err := s.db.Query(s.queries.listConversations, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var conversationSummaries []conversationSummaryRow
	for rows.Next() {
		var conversationSummary conversationSummaryRow
		if err := rows.Scan(&conversationSummary.ID, &conversationSummary.PublicID, &conversationSummary.StartedAt, &conversationSummary.Preview); err != nil {
			return nil, err
		}
		conversationSummaries = append(conversationSummaries, conversationSummary)
	}
	return conversationSummaries, rows.Err()
}

// loadConversation returns all messages for one user-owned conversation in order.
func (s *Store) loadConversation(userID, conversationID int64) ([]struct {
	Role             string
	Content          string
	ModelID          string
	PromptTokens     int64
	CompletionTokens int64
}, error) {
	rows, err := s.db.Query(
		s.queries.loadConversation, conversationID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var conversationMessages []struct {
		Role             string
		Content          string
		ModelID          string
		PromptTokens     int64
		CompletionTokens int64
	}
	for rows.Next() {
		var messageRow struct {
			Role             string
			Content          string
			ModelID          string
			PromptTokens     int64
			CompletionTokens int64
		}
		if err := rows.Scan(&messageRow.Role, &messageRow.Content, &messageRow.ModelID, &messageRow.PromptTokens, &messageRow.CompletionTokens); err != nil {
			return nil, err
		}
		conversationMessages = append(conversationMessages, messageRow)
	}
	return conversationMessages, rows.Err()
}

// deleteConversation removes one user-owned conversation and all its messages.
func (s *Store) deleteConversation(userID, conversationID int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		s.queries.deleteConversationMessages,
		conversationID, userID,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(s.queries.deleteConversation, conversationID, userID); err != nil {
		return err
	}
	return tx.Commit()
}

// setUserName upserts one user's profile row.
func (s *Store) setUserName(userID int64, name string, updatedAt int64) error {
	_, err := s.db.Exec(
		s.queries.upsertUserProfileName,
		userID, name, updatedAt,
	)
	return err
}

func (s *Store) getUserName(userID int64) (name string, updatedAt int64, err error) {
	row := s.db.QueryRow(s.queries.getUserName, userID)
	if scanErr := row.Scan(&name, &updatedAt); scanErr != nil {
		if errors.Is(scanErr, sql.ErrNoRows) {
			return "User", 0, nil
		}
		return "", 0, scanErr
	}
	return name, updatedAt, nil
}

func (s *Store) setSelectedModel(userID int64, model string) error {
	_, err := s.db.Exec(
		s.queries.setSelectedModel,
		userID, model,
	)
	return err
}

func (s *Store) getSelectedModel(userID int64, fallback string) (string, error) {
	row := s.db.QueryRow(s.queries.getSelectedModel, userID)
	var selectedModel string
	if scanErr := row.Scan(&selectedModel); scanErr != nil {
		if errors.Is(scanErr, sql.ErrNoRows) {
			return fallback, nil
		}
		return "", scanErr
	}
	if selectedModel == "" {
		return fallback, nil
	}
	return selectedModel, nil
}

func (s *Store) setSelectedTone(userID int64, tone string) error {
	_, err := s.db.Exec(
		s.queries.setSelectedTone,
		userID, tone,
	)
	return err
}

func (s *Store) getSelectedTone(userID int64, fallback string) (string, error) {
	row := s.db.QueryRow(s.queries.getSelectedTone, userID)
	var selectedTone string
	if scanErr := row.Scan(&selectedTone); scanErr != nil {
		if errors.Is(scanErr, sql.ErrNoRows) {
			return fallback, nil
		}
		return "", scanErr
	}
	if selectedTone == "" {
		return fallback, nil
	}
	return selectedTone, nil
}

func (s *Store) setSelectedThinkingEnabled(userID int64, enabled bool) error {
	enabledValue := 0
	if enabled {
		enabledValue = 1
	}
	_, err := s.db.Exec(
		s.queries.setSelectedThinkingEnabled,
		userID, enabledValue,
	)
	return err
}

func (s *Store) getSelectedThinkingEnabled(userID int64, fallback bool) (bool, error) {
	row := s.db.QueryRow(s.queries.getSelectedThinkingEnabled, userID)
	var enabledValue int64
	if scanErr := row.Scan(&enabledValue); scanErr != nil {
		if errors.Is(scanErr, sql.ErrNoRows) {
			return fallback, nil
		}
		return false, scanErr
	}
	return enabledValue != 0, nil
}

func (s *Store) setSelectedThinkingEffort(userID int64, effort string) error {
	_, err := s.db.Exec(
		s.queries.setSelectedThinkingEffort,
		userID, effort,
	)
	return err
}

func (s *Store) getSelectedThinkingEffort(userID int64, fallback string) (string, error) {
	row := s.db.QueryRow(s.queries.getSelectedThinkingEffort, userID)
	var effort string
	if scanErr := row.Scan(&effort); scanErr != nil {
		if errors.Is(scanErr, sql.ErrNoRows) {
			return fallback, nil
		}
		return "", scanErr
	}
	if effort == "" {
		return fallback, nil
	}
	return effort, nil
}

func (s *Store) setSelectedSystemPrompt(userID int64, prompt string) error {
	_, err := s.db.Exec(
		s.queries.setSelectedSystemPrompt,
		userID, prompt,
	)
	return err
}

func (s *Store) getSelectedSystemPrompt(userID int64, fallback string) (string, error) {
	row := s.db.QueryRow(s.queries.getSelectedSystemPrompt, userID)
	var prompt string
	if scanErr := row.Scan(&prompt); scanErr != nil {
		if errors.Is(scanErr, sql.ErrNoRows) {
			return fallback, nil
		}
		return "", scanErr
	}
	if prompt == "" {
		return fallback, nil
	}
	return prompt, nil
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

func (s *Store) upsertUserMemory(userID int64, memory userMemoryRow) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(
		s.queries.upsertUserMemory,
		userID,
		memory.Key,
		memory.Category,
		memory.Summary,
		memory.Detail,
		memory.SourceMessage,
		memory.UsefulnessScore,
		memory.ConfidenceScore,
		memory.RubricReason,
		now,
		now,
	)
	return err
}

func (s *Store) listUserMemories(userID int64) ([]userMemoryRow, error) {
	rows, err := s.db.Query(s.queries.listUserMemories, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	memories := make([]userMemoryRow, 0)
	for rows.Next() {
		var memory userMemoryRow
		if err := rows.Scan(
			&memory.Key,
			&memory.Category,
			&memory.Summary,
			&memory.Detail,
			&memory.SourceMessage,
			&memory.UsefulnessScore,
			&memory.ConfidenceScore,
			&memory.RubricReason,
			&memory.UpdatedAt,
		); err != nil {
			return nil, err
		}
		memories = append(memories, memory)
	}
	return memories, rows.Err()
}

func (s *Store) deleteUserMemory(userID int64, key string) error {
	_, err := s.db.Exec(s.queries.deleteUserMemory, userID, strings.TrimSpace(key))
	return err
}

func (s *Store) listModelCatalog() ([]modelCatalogRow, error) {
	rows, err := s.db.Query(s.queries.listModelCatalog)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	catalog := make([]modelCatalogRow, 0)
	for rows.Next() {
		var row modelCatalogRow
		var supportsThinking int64
		var supportsSpeech int64
		var onboardingReady int64
		var isDefault int64
		var useForTitleGeneration int64
		var useForMemoryExtraction int64
		if err := rows.Scan(
			&row.ID,
			&row.ProviderID,
			&row.ProviderLabel,
			&row.Label,
			&row.Note,
			&row.Description,
			&supportsThinking,
			&supportsSpeech,
			&row.InputPerMillionUSD,
			&row.OutputPerMillionUSD,
			&row.PricingCurrency,
			&row.MaxOutputTokens,
			&row.ThroughputTokensPerSec,
			&onboardingReady,
			&isDefault,
			&useForTitleGeneration,
			&useForMemoryExtraction,
			&row.SortOrder,
		); err != nil {
			return nil, err
		}
		row.SupportsThinking = supportsThinking != 0
		row.SupportsSpeech = supportsSpeech != 0
		row.OnboardingReady = onboardingReady != 0
		row.IsDefault = isDefault != 0
		row.UseForTitleGeneration = useForTitleGeneration != 0
		row.UseForMemoryExtraction = useForMemoryExtraction != 0
		catalog = append(catalog, row)
	}
	return catalog, rows.Err()
}
