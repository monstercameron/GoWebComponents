package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	"golang.org/x/crypto/bcrypt"
)

func TestLoadSeedQueriesLoadsExpectedStatements(parseT *testing.T) {
	parseQueries, parseErr := parseLoadSeedQueries()
	if parseErr != nil {
		parseT.Fatalf("loadSeedQueries() error = %v", parseErr)
	}
	if !strings.Contains(parseQueries.schema, "CREATE TABLE IF NOT EXISTS users") {
		parseT.Fatal("schema query missing users table")
	}
	if !strings.Contains(parseQueries.parseCreateUser, "INSERT INTO users") {
		parseT.Fatal("createUser query missing insert")
	}
	if !strings.Contains(parseQueries.insertUserProfile, "INSERT INTO user_profile") {
		parseT.Fatal("insertUserProfile query missing insert")
	}
	if !strings.Contains(parseQueries.insertConversation, "INSERT INTO conversations") {
		parseT.Fatal("insertConversation query missing insert")
	}
	if !strings.Contains(parseQueries.insertMessage, "INSERT INTO messages") {
		parseT.Fatal("insertMessage query missing insert")
	}
}

func TestInsertSeedUserCreatesUserAndProfile(parseT *testing.T) {
	parseQueries, parseErr := parseLoadSeedQueries()
	if parseErr != nil {
		parseT.Fatalf("loadSeedQueries() error = %v", parseErr)
	}
	parseDb, parseErr := sql.Open("sqlite3", "file::memory:?cache=shared")
	if parseErr != nil {
		parseT.Fatalf("sql.Open() error = %v", parseErr)
	}
	defer parseDb.Close()

	if _, parseErr2 := parseDb.Exec(parseQueries.schema); parseErr2 != nil {
		parseT.Fatalf("db.Exec(schema) error = %v", parseErr2)
	}

	parseNow := time.Date(2026, 3, 25, 14, 30, 0, 0, time.UTC)
	parseUserID, parseErr := parseInsertSeedUser(parseDb, parseQueries, parseNow, "customer@email.com", "password", "Customer User", "gpt-5.4-mini", "balanced", "medium", 1)
	if parseErr != nil {
		parseT.Fatalf("insertSeedUser() error = %v", parseErr)
	}
	if parseUserID <= 0 {
		parseT.Fatalf("insertSeedUser() userID = %d, want positive ID", parseUserID)
	}

	var parseEmail string
	var parsePasswordHash string
	var parseCreatedAt string
	if parseErr3 := parseDb.QueryRow(`SELECT email, password_hash, created_at FROM users WHERE id = ?`, parseUserID).Scan(&parseEmail, &parsePasswordHash, &parseCreatedAt); parseErr3 != nil {
		parseT.Fatalf("QueryRow(users) error = %v", parseErr3)
	}
	if parseEmail != "customer@email.com" || parseCreatedAt != parseNow.Format(time.RFC3339) {
		parseT.Fatalf("users row = (%q, %q), want customer@email.com and %q", parseEmail, parseCreatedAt, parseNow.Format(time.RFC3339))
	}
	if parsePasswordHash == "password" || parsePasswordHash == "" {
		parseT.Fatalf("password hash = %q, want non-empty bcrypt hash", parsePasswordHash)
	}

	var parseName, parseModel, parseTone, parseEffort string
	var parseUpdatedAt int64
	var parseEnabled int
	if parseErr4 := parseDb.QueryRow(`SELECT name, updated_at, selected_model, selected_tone, selected_thinking_enabled, selected_thinking_effort FROM user_profile WHERE user_id = ?`, parseUserID).Scan(&parseName, &parseUpdatedAt, &parseModel, &parseTone, &parseEnabled, &parseEffort); parseErr4 != nil {
		parseT.Fatalf("QueryRow(user_profile) error = %v", parseErr4)
	}
	if parseName != "Customer User" || parseUpdatedAt != parseNow.Unix() || parseModel != "gpt-5.4-mini" || parseTone != "balanced" || parseEnabled != 1 || parseEffort != "medium" {
		parseT.Fatalf("user_profile row = (%q, %d, %q, %q, %d, %q), want Customer User/%d/gpt-5.4-mini/balanced/1/medium", parseName, parseUpdatedAt, parseModel, parseTone, parseEnabled, parseEffort, parseNow.Unix())
	}

	if _, parseErr5 := parseInsertSeedUser(parseDb, parseQueries, parseNow, "customer@email.com", "password", "Customer User", "gpt-5.4-mini", "balanced", "medium", 1); parseErr5 == nil {
		parseT.Fatal("insertSeedUser() duplicate email error = nil, want unique-constraint failure")
	}
}

// TestRunSeedTestDBSeedsExpectedRows verifies the end-to-end seed workflow against a temporary sqlite file.
func TestRunSeedTestDBSeedsExpectedRows(parseT *testing.T) {
	parseDbPath := filepath.Join(parseT.TempDir(), "runtime", "seed.db")
	parseT.Setenv("CHAT_DB_PATH", parseDbPath)

	parseSummary, parseErr := runSeedTestDB()
	if parseErr != nil {
		parseT.Fatalf("runSeedTestDB(): %v", parseErr)
	}
	if !strings.Contains(parseSummary, parseDbPath) ||
		!strings.Contains(parseSummary, "customer@email.com / password") ||
		!strings.Contains(parseSummary, "admin@email.com / password") {
		parseT.Fatalf("unexpected seed summary %q", parseSummary)
	}

	parseDb, parseErr := sql.Open("sqlite3", "file:"+parseDbPath)
	if parseErr != nil {
		parseT.Fatalf("sql.Open(): %v", parseErr)
	}
	defer parseDb.Close()

	var parseUserCount int
	if parseErr2 := parseDb.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&parseUserCount); parseErr2 != nil {
		parseT.Fatalf("QueryRow(users count): %v", parseErr2)
	}
	if parseUserCount != 2 {
		parseT.Fatalf("expected 2 users, got %d", parseUserCount)
	}
	parseExpectedUsers := map[string]bool{
		"admin@email.com":    false,
		"customer@email.com": false,
	}
	parseRows, parseErr := parseDb.Query(`SELECT email, password_hash FROM users`)
	if parseErr != nil {
		parseT.Fatalf("Query(users credentials): %v", parseErr)
	}
	defer parseRows.Close()
	for parseRows.Next() {
		var parseEmail string
		var parsePasswordHash string
		if parseErr2 := parseRows.Scan(&parseEmail, &parsePasswordHash); parseErr2 != nil {
			parseT.Fatalf("Scan(users credentials): %v", parseErr2)
		}
		if _, isParseExpected := parseExpectedUsers[parseEmail]; !isParseExpected {
			parseT.Fatalf("unexpected seeded user email %q", parseEmail)
		}
		if parseErr2 := bcrypt.CompareHashAndPassword([]byte(parsePasswordHash), []byte("password")); parseErr2 != nil {
			parseT.Fatalf("expected seeded password hash to validate for %s: %v", parseEmail, parseErr2)
		}
		parseExpectedUsers[parseEmail] = true
	}
	if parseErr2 := parseRows.Err(); parseErr2 != nil {
		parseT.Fatalf("Rows(users credentials): %v", parseErr2)
	}
	for parseEmail, isParseFound := range parseExpectedUsers {
		if !isParseFound {
			parseT.Fatalf("expected seeded user %q to exist", parseEmail)
		}
	}

	var parseConversationCount int
	if parseErr3 := parseDb.QueryRow(`SELECT COUNT(*) FROM conversations`).Scan(&parseConversationCount); parseErr3 != nil {
		parseT.Fatalf("QueryRow(conversations count): %v", parseErr3)
	}
	if parseConversationCount != 3 {
		parseT.Fatalf("expected 3 conversations, got %d", parseConversationCount)
	}

	var parseMessageCount int
	if parseErr4 := parseDb.QueryRow(`SELECT COUNT(*) FROM messages`).Scan(&parseMessageCount); parseErr4 != nil {
		parseT.Fatalf("QueryRow(messages count): %v", parseErr4)
	}
	if parseMessageCount != 6 {
		parseT.Fatalf("expected 6 messages, got %d", parseMessageCount)
	}
}

// TestRunSeedTestDBReturnsDuplicateFailure verifies rerunning against the same file surfaces the insert error.
func TestRunSeedTestDBReturnsDuplicateFailure(parseT *testing.T) {
	parseDbPath := filepath.Join(parseT.TempDir(), "runtime", "seed.db")
	parseOriginalPath := os.Getenv("CHAT_DB_PATH")
	parseT.Cleanup(func() {
		_ = os.Setenv("CHAT_DB_PATH", parseOriginalPath)
	})
	if parseErr := os.Setenv("CHAT_DB_PATH", parseDbPath); parseErr != nil {
		parseT.Fatalf("Setenv(CHAT_DB_PATH): %v", parseErr)
	}

	if _, parseErr := runSeedTestDB(); parseErr != nil {
		parseT.Fatalf("first runSeedTestDB(): %v", parseErr)
	}
	if _, parseErr := runSeedTestDB(); parseErr == nil || !strings.Contains(parseErr.Error(), "customer user:") {
		parseT.Fatalf("expected duplicate seed failure, got %v", parseErr)
	}
}
