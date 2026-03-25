package main

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
)

func TestLoadSeedQueriesLoadsExpectedStatements(t *testing.T) {
	queries, err := loadSeedQueries()
	if err != nil {
		t.Fatalf("loadSeedQueries() error = %v", err)
	}
	if !strings.Contains(queries.schema, "CREATE TABLE IF NOT EXISTS users") {
		t.Fatal("schema query missing users table")
	}
	if !strings.Contains(queries.createUser, "INSERT INTO users") {
		t.Fatal("createUser query missing insert")
	}
	if !strings.Contains(queries.insertUserProfile, "INSERT INTO user_profile") {
		t.Fatal("insertUserProfile query missing insert")
	}
	if !strings.Contains(queries.insertConversation, "INSERT INTO conversations") {
		t.Fatal("insertConversation query missing insert")
	}
	if !strings.Contains(queries.insertMessage, "INSERT INTO messages") {
		t.Fatal("insertMessage query missing insert")
	}
}

func TestInsertSeedUserCreatesUserAndProfile(t *testing.T) {
	queries, err := loadSeedQueries()
	if err != nil {
		t.Fatalf("loadSeedQueries() error = %v", err)
	}
	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(queries.schema); err != nil {
		t.Fatalf("db.Exec(schema) error = %v", err)
	}

	now := time.Date(2026, 3, 25, 14, 30, 0, 0, time.UTC)
	userID, err := insertSeedUser(db, queries, now, "demo@example.com", "password123", "Demo User", "gpt-5.4-mini", "balanced", "medium", 1)
	if err != nil {
		t.Fatalf("insertSeedUser() error = %v", err)
	}
	if userID <= 0 {
		t.Fatalf("insertSeedUser() userID = %d, want positive ID", userID)
	}

	var email string
	var passwordHash string
	var createdAt string
	if err := db.QueryRow(`SELECT email, password_hash, created_at FROM users WHERE id = ?`, userID).Scan(&email, &passwordHash, &createdAt); err != nil {
		t.Fatalf("QueryRow(users) error = %v", err)
	}
	if email != "demo@example.com" || createdAt != now.Format(time.RFC3339) {
		t.Fatalf("users row = (%q, %q), want demo@example.com and %q", email, createdAt, now.Format(time.RFC3339))
	}
	if passwordHash == "password123" || passwordHash == "" {
		t.Fatalf("password hash = %q, want non-empty bcrypt hash", passwordHash)
	}

	var name, model, tone, effort string
	var updatedAt int64
	var enabled int
	if err := db.QueryRow(`SELECT name, updated_at, selected_model, selected_tone, selected_thinking_enabled, selected_thinking_effort FROM user_profile WHERE user_id = ?`, userID).Scan(&name, &updatedAt, &model, &tone, &enabled, &effort); err != nil {
		t.Fatalf("QueryRow(user_profile) error = %v", err)
	}
	if name != "Demo User" || updatedAt != now.Unix() || model != "gpt-5.4-mini" || tone != "balanced" || enabled != 1 || effort != "medium" {
		t.Fatalf("user_profile row = (%q, %d, %q, %q, %d, %q), want Demo User/%d/gpt-5.4-mini/balanced/1/medium", name, updatedAt, model, tone, enabled, effort, now.Unix())
	}

	if _, err := insertSeedUser(db, queries, now, "demo@example.com", "password123", "Demo User", "gpt-5.4-mini", "balanced", "medium", 1); err == nil {
		t.Fatal("insertSeedUser() duplicate email error = nil, want unique-constraint failure")
	}
}
