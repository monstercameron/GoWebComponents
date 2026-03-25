package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	_ "github.com/ncruces/go-sqlite3/driver"
	"golang.org/x/crypto/bcrypt"
)

func insertSeedUser(db *sql.DB, queries seedQueries, now time.Time, email, password, displayName, model, tone, thinkingEffort string, thinkingEnabled int) (int64, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	userResult, err := db.Exec(
		queries.createUser,
		email, string(passwordHash), now.Format(time.RFC3339),
	)
	if err != nil {
		return 0, err
	}
	userID, err := userResult.LastInsertId()
	if err != nil {
		return 0, err
	}
	if _, err := db.Exec(
		queries.insertUserProfile,
		userID, displayName, now.Unix(), model, tone, thinkingEnabled, thinkingEffort,
	); err != nil {
		return 0, err
	}
	return userID, nil
}

func main() {
	dbPath := os.Getenv("CHAT_DB_PATH")
	if dbPath == "" {
		dbPath = "examples/100-ai-chat-wizard/bin/runtime/test_chat.db"
	}
	if dir := filepath.Dir(dbPath); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintln(os.Stderr, "mkdir:", err)
			os.Exit(1)
		}
	}
	queries, err := loadSeedQueries()
	if err != nil {
		fmt.Fprintln(os.Stderr, "load sql:", err)
		os.Exit(1)
	}

	dsn := "file:" + dbPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "open:", err)
		os.Exit(1)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(queries.schema); err != nil {
		fmt.Fprintln(os.Stderr, "schema:", err)
		os.Exit(1)
	}

	now := time.Now().UTC()
	userID, err := insertSeedUser(db, queries, now, "demo@example.com", "password123", "Demo User", "gpt-5.4-mini", "balanced", "medium", 1)
	if err != nil {
		fmt.Fprintln(os.Stderr, "demo user:", err)
		os.Exit(1)
	}
	adminUserID, err := insertSeedUser(db, queries, now, "admin@example.com", "password", "Admin User", "gpt-5.4", "professional", "high", 1)
	if err != nil {
		fmt.Fprintln(os.Stderr, "admin user:", err)
		os.Exit(1)
	}

	r1, err := db.Exec(
		queries.insertConversation,
		userID, uuid.NewString(), now.Add(-2*time.Hour).Format(time.RFC3339), "Golang Goroutines Explained",
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "conv1:", err)
		os.Exit(1)
	}
	id1, _ := r1.LastInsertId()

	for _, m := range []struct {
		role, content    string
		modelID          string
		promptTokens     int
		completionTokens int
	}{
		{"user", "How do goroutines work in Go?", "", 0, 0},
		{"assistant", "Goroutines are lightweight threads managed by the Go runtime. Unlike OS threads, goroutines are multiplexed onto a small number of OS threads automatically by the scheduler. You create one with the `go` keyword: `go myFunc()`.", "gpt-5.4", 120, 60},
	} {
		if _, err := db.Exec(
			queries.insertMessage,
			id1, m.role, m.content, m.modelID, m.promptTokens, m.completionTokens, now.Add(-2*time.Hour).Format(time.RFC3339), id1, userID,
		); err != nil {
			fmt.Fprintln(os.Stderr, "msg conv1:", err)
			os.Exit(1)
		}
	}

	r2, err := db.Exec(
		queries.insertConversation,
		userID, uuid.NewString(), now.Add(-1*time.Hour).Format(time.RFC3339), "WebAssembly and Go",
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "conv2:", err)
		os.Exit(1)
	}
	id2, _ := r2.LastInsertId()

	for _, m := range []struct {
		role, content    string
		modelID          string
		promptTokens     int
		completionTokens int
	}{
		{"user", "What is WebAssembly?", "", 0, 0},
		{"assistant", "WebAssembly (WASM) is a binary instruction format for a stack-based virtual machine. It allows code written in languages like C, Rust, and Go to run in the browser at near-native speed.", "gpt-5.4-mini", 160, 80},
	} {
		if _, err := db.Exec(
			queries.insertMessage,
			id2, m.role, m.content, m.modelID, m.promptTokens, m.completionTokens, now.Add(-1*time.Hour).Format(time.RFC3339), id2, userID,
		); err != nil {
			fmt.Fprintln(os.Stderr, "msg conv2:", err)
			os.Exit(1)
		}
	}

	r3, err := db.Exec(
		queries.insertConversation,
		userID, uuid.NewString(), now.Add(-30*time.Minute).Format(time.RFC3339), "Canvas Preview Demo",
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "conv3:", err)
		os.Exit(1)
	}
	id3, _ := r3.LastInsertId()

	for _, m := range []struct {
		role, content    string
		modelID          string
		promptTokens     int
		completionTokens int
	}{
		{"user", "Show me a small interactive canvas preview.", "", 0, 0},
		{"assistant", "Here is a runnable preview.\n\n```canvas\n<style>\n  #canvas {\n    min-height: 100vh;\n    display: grid;\n    place-items: center;\n    background: radial-gradient(circle at top, #d8fff1, #f6f8f7 55%);\n  }\n  .demo-card {\n    padding: 1.25rem 1.5rem;\n    border-radius: 1.25rem;\n    background: #111827;\n    color: white;\n    box-shadow: 0 20px 50px rgba(17, 24, 39, 0.2);\n    text-align: center;\n  }\n</style>\n<div class=\"demo-card\">\n  <h1 style=\"margin:0 0 .4rem; font-size:1.5rem;\">Canvas demo</h1>\n  <p style=\"margin:0; color:rgba(255,255,255,.75);\">Rendered inside #canvas.</p>\n</div>\n<script>\n  const root = document.getElementById('canvas');\n  const card = document.querySelector('.demo-card');\n  if (root && card) {\n    root.appendChild(card);\n  }\n</script>\n```", "gpt-5.4-mini", 210, 120},
	} {
		if _, err := db.Exec(
			queries.insertMessage,
			id3, m.role, m.content, m.modelID, m.promptTokens, m.completionTokens, now.Add(-30*time.Minute).Format(time.RFC3339), id3, userID,
		); err != nil {
			fmt.Fprintln(os.Stderr, "msg conv3:", err)
			os.Exit(1)
		}
	}

	fmt.Printf("seeded test DB: %s (users: demo@example.com / password123, admin@example.com / password; demo user id: %d, admin user id: %d; conversations: %d, %d, %d)\n", dbPath, userID, adminUserID, id1, id2, id3)
}
