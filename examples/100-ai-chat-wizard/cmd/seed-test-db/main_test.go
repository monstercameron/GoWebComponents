package main

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
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
	parseUserID, parseErr := parseInsertSeedUser(parseDb, parseQueries, parseNow, "demo@example.com", "password123", "Demo User", "gpt-5.4-mini", "balanced", "medium", 1)
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
	if parseEmail != "demo@example.com" || parseCreatedAt != parseNow.Format(time.RFC3339) {
		parseT.Fatalf("users row = (%q, %q), want demo@example.com and %q", parseEmail, parseCreatedAt, parseNow.Format(time.RFC3339))
	}
	if parsePasswordHash == "password123" || parsePasswordHash == "" {
		parseT.Fatalf("password hash = %q, want non-empty bcrypt hash", parsePasswordHash)
	}

	var parseName, parseModel, parseTone, parseEffort string
	var parseUpdatedAt int64
	var parseEnabled int
	if parseErr4 := parseDb.QueryRow(`SELECT name, updated_at, selected_model, selected_tone, selected_thinking_enabled, selected_thinking_effort FROM user_profile WHERE user_id = ?`, parseUserID).Scan(&parseName, &parseUpdatedAt, &parseModel, &parseTone, &parseEnabled, &parseEffort); parseErr4 != nil {
		parseT.Fatalf("QueryRow(user_profile) error = %v", parseErr4)
	}
	if parseName != "Demo User" || parseUpdatedAt != parseNow.Unix() || parseModel != "gpt-5.4-mini" || parseTone != "balanced" || parseEnabled != 1 || parseEffort != "medium" {
		parseT.Fatalf("user_profile row = (%q, %d, %q, %q, %d, %q), want Demo User/%d/gpt-5.4-mini/balanced/1/medium", parseName, parseUpdatedAt, parseModel, parseTone, parseEnabled, parseEffort, parseNow.Unix())
	}

	if _, parseErr5 := parseInsertSeedUser(parseDb, parseQueries, parseNow, "demo@example.com", "password123", "Demo User", "gpt-5.4-mini", "balanced", "medium", 1); parseErr5 == nil {
		parseT.Fatal("insertSeedUser() duplicate email error = nil, want unique-constraint failure")
	}
}
