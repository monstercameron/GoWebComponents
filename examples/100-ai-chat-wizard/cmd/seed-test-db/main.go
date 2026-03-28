package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"golang.org/x/crypto/bcrypt"
)

func parseInsertSeedUser(parseDb *sql.DB, parseQueries seedQueries, parseNow time.Time, parseEmail, parsePassword, parseDisplayName, parseModel, parseTone, parseThinkingEffort string, parseThinkingEnabled int) (int64, error) {
	parsePasswordHash, parseErr := bcrypt.GenerateFromPassword([]byte(parsePassword), bcrypt.DefaultCost)
	if parseErr != nil {
		return 0, parseErr
	}
	parseUserResult, parseErr := parseDb.Exec(
		parseQueries.parseCreateUser,
		parseEmail, string(parsePasswordHash), parseNow.Format(time.RFC3339),
	)
	if parseErr != nil {
		return 0, parseErr
	}
	parseUserID, parseErr := parseUserResult.LastInsertId()
	if parseErr != nil {
		return 0, parseErr
	}
	if _, parseErr2 := parseDb.Exec(
		parseQueries.insertUserProfile,
		parseUserID, parseDisplayName, parseNow.Unix(), parseModel, parseTone, parseThinkingEnabled, parseThinkingEffort,
	); parseErr2 != nil {
		return 0, parseErr2
	}
	return parseUserID, nil
}

func parseUpsertSeedBillingCustomer(parseDb *sql.DB, parseQueries seedQueries, parseNow time.Time, parseUserID int64, parseEmail, parseDisplayName string) error {
	parseProviderCustomerID := fmt.Sprintf("seed-user-%d", parseUserID)
	_, parseErr := parseDb.Exec(
		parseQueries.upsertBillingUser,
		parseUserID,
		"seed",
		parseProviderCustomerID,
		parseEmail,
		parseDisplayName,
		"US",
		"NA",
		"USD",
		"none",
		"{}",
		parseNow.Format(time.RFC3339),
		parseNow.Format(time.RFC3339),
		parseUserID,
	)
	return parseErr
}

func main() {
	parseSummary, parseErr := runSeedTestDB()
	if parseErr != nil {
		fmt.Fprintln(os.Stderr, parseErr)
		os.Exit(1)
	}
	fmt.Println(parseSummary)
}

// runSeedTestDB creates and seeds the demo chat database, returning a summary line.
func runSeedTestDB() (string, error) {
	parseDbPath := os.Getenv("CHAT_DB_PATH")
	if parseDbPath == "" {
		parseDbPath = "examples/100-ai-chat-wizard/bin/runtime/test_chat.db"
	}
	if parseDir := filepath.Dir(parseDbPath); parseDir != "." && parseDir != "" {
		if parseErr := os.MkdirAll(parseDir, 0o755); parseErr != nil {
			return "", fmt.Errorf("mkdir: %w", parseErr)
		}
	}
	parseQueries, parseErr2 := parseLoadSeedQueries()
	if parseErr2 != nil {
		return "", fmt.Errorf("load sql: %w", parseErr2)
	}

	parseDsn := "file:" + parseDbPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	parseDb, parseErr2 := sql.Open("sqlite3", parseDsn)
	if parseErr2 != nil {
		return "", fmt.Errorf("open: %w", parseErr2)
	}
	defer parseDb.Close()
	parseDb.SetMaxOpenConns(1)

	if _, parseErr3 := parseDb.Exec(parseQueries.schema); parseErr3 != nil {
		return "", fmt.Errorf("schema: %w", parseErr3)
	}

	parseNow := time.Now().UTC()
	parseUserID, parseErr2 := parseInsertSeedUser(parseDb, parseQueries, parseNow, "customer@email.com", "password", "Customer User", "gpt-5.4-mini", "balanced", "medium", 1)
	if parseErr2 != nil {
		return "", fmt.Errorf("customer user: %w", parseErr2)
	}
	parseAdminUserID, parseErr2 := parseInsertSeedUser(parseDb, parseQueries, parseNow, "admin@email.com", "password", "Admin User", "gpt-5.4", "professional", "high", 1)
	if parseErr2 != nil {
		return "", fmt.Errorf("admin user: %w", parseErr2)
	}
	if parseErr2 = parseUpsertSeedBillingCustomer(parseDb, parseQueries, parseNow, parseUserID, "customer@email.com", "Customer User"); parseErr2 != nil {
		return "", fmt.Errorf("customer billing customer: %w", parseErr2)
	}
	if parseErr2 = parseUpsertSeedBillingCustomer(parseDb, parseQueries, parseNow, parseAdminUserID, "admin@email.com", "Admin User"); parseErr2 != nil {
		return "", fmt.Errorf("admin billing customer: %w", parseErr2)
	}

	parseR1, parseErr2 := parseDb.Exec(
		parseQueries.insertConversation,
		parseUserID, uuid.NewString(), parseNow.Add(-2*time.Hour).Format(time.RFC3339), "Golang Goroutines Explained",
	)
	if parseErr2 != nil {
		return "", fmt.Errorf("conv1: %w", parseErr2)
	}
	parseId1, _ := parseR1.LastInsertId()

	for _, parseM := range []struct {
		role, content    string
		modelID          string
		promptTokens     int
		completionTokens int
	}{
		{"user", "How do goroutines work in Go?", "", 0, 0},
		{"assistant", "Goroutines are lightweight threads managed by the Go runtime. Unlike OS threads, goroutines are multiplexed onto a small number of OS threads automatically by the scheduler. You create one with the `go` keyword: `go myFunc()`.", "gpt-5.4", 120, 60},
	} {
		if _, parseErr4 := parseDb.Exec(
			parseQueries.insertMessage,
			parseId1, parseM.role, parseM.content, parseM.modelID, parseM.promptTokens, parseM.completionTokens, parseNow.Add(-2*time.Hour).Format(time.RFC3339), parseId1, parseUserID,
		); parseErr4 != nil {
			return "", fmt.Errorf("msg conv1: %w", parseErr4)
		}
	}

	parseR2, parseErr2 := parseDb.Exec(
		parseQueries.insertConversation,
		parseUserID, uuid.NewString(), parseNow.Add(-1*time.Hour).Format(time.RFC3339), "WebAssembly and Go",
	)
	if parseErr2 != nil {
		return "", fmt.Errorf("conv2: %w", parseErr2)
	}
	parseId2, _ := parseR2.LastInsertId()

	for _, parseM2 := range []struct {
		role, content    string
		modelID          string
		promptTokens     int
		completionTokens int
	}{
		{"user", "What is WebAssembly?", "", 0, 0},
		{"assistant", "WebAssembly (WASM) is a binary instruction format for a stack-based virtual machine. It allows code written in languages like C, Rust, and Go to run in the browser at near-native speed.", "gpt-5.4-mini", 160, 80},
	} {
		if _, parseErr5 := parseDb.Exec(
			parseQueries.insertMessage,
			parseId2, parseM2.role, parseM2.content, parseM2.modelID, parseM2.promptTokens, parseM2.completionTokens, parseNow.Add(-1*time.Hour).Format(time.RFC3339), parseId2, parseUserID,
		); parseErr5 != nil {
			return "", fmt.Errorf("msg conv2: %w", parseErr5)
		}
	}

	parseR3, parseErr2 := parseDb.Exec(
		parseQueries.insertConversation,
		parseUserID, uuid.NewString(), parseNow.Add(-30*time.Minute).Format(time.RFC3339), "Canvas Preview Demo",
	)
	if parseErr2 != nil {
		return "", fmt.Errorf("conv3: %w", parseErr2)
	}
	parseId3, _ := parseR3.LastInsertId()

	for _, parseM3 := range []struct {
		role, content    string
		modelID          string
		promptTokens     int
		completionTokens int
	}{
		{"user", "Show me a small interactive canvas preview.", "", 0, 0},
		{"assistant", "Here is a runnable preview.\n\n```canvas\n<style>\n  #canvas {\n    min-height: 100vh;\n    display: grid;\n    place-items: center;\n    background: radial-gradient(circle at top, #d8fff1, #f6f8f7 55%);\n  }\n  .demo-card {\n    padding: 1.25rem 1.5rem;\n    border-radius: 1.25rem;\n    background: #111827;\n    color: white;\n    box-shadow: 0 20px 50px rgba(17, 24, 39, 0.2);\n    text-align: center;\n  }\n</style>\n<div class=\"demo-card\">\n  <h1 style=\"margin:0 0 .4rem; font-size:1.5rem;\">Canvas demo</h1>\n  <p style=\"margin:0; color:rgba(255,255,255,.75);\">Rendered inside #canvas.</p>\n</div>\n<script>\n  const root = document.getElementById('canvas');\n  const card = document.querySelector('.demo-card');\n  if (root && card) {\n    root.appendChild(card);\n  }\n</script>\n```", "gpt-5.4-mini", 210, 120},
	} {
		if _, parseErr6 := parseDb.Exec(
			parseQueries.insertMessage,
			parseId3, parseM3.role, parseM3.content, parseM3.modelID, parseM3.promptTokens, parseM3.completionTokens, parseNow.Add(-30*time.Minute).Format(time.RFC3339), parseId3, parseUserID,
		); parseErr6 != nil {
			return "", fmt.Errorf("msg conv3: %w", parseErr6)
		}
	}

	return fmt.Sprintf("seeded test DB: %s (users: customer@email.com / password, admin@email.com / password; customer user id: %d, admin user id: %d; conversations: %d, %d, %d)", parseDbPath, parseUserID, parseAdminUserID, parseId1, parseId2, parseId3), nil
}
