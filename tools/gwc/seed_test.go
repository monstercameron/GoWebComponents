package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveSeedConfigDefaultsToChatWizardSeeder(t *testing.T) {
	root := t.TempDir()
	commandDir := filepath.Join(root, "examples", "100-ai-chat-wizard", "cmd", "seed-test-db")
	if err := os.MkdirAll(commandDir, 0755); err != nil {
		t.Fatalf("mkdir seed command: %v", err)
	}

	originalGetwd := seedGetwd
	t.Cleanup(func() { seedGetwd = originalGetwd })
	seedGetwd = func() (string, error) { return root, nil }

	config, err := resolveSeedConfig(seedConfig{})
	if err != nil {
		t.Fatalf("resolve seed config: %v", err)
	}
	if config.rootPath != root {
		t.Fatalf("expected root path %q, got %#v", root, config)
	}
	if config.commandPath != commandDir {
		t.Fatalf("expected command path %q, got %#v", commandDir, config)
	}
	wantDB := filepath.Join(root, "examples", "100-ai-chat-wizard", "bin", "runtime", "test_chat.db")
	if config.dbPath != wantDB {
		t.Fatalf("expected db path %q, got %#v", wantDB, config)
	}
}

func TestRunSeedJSONExecutesDefaultSeederWithKnownCredentials(t *testing.T) {
	root := t.TempDir()
	commandDir := filepath.Join(root, "examples", "100-ai-chat-wizard", "cmd", "seed-test-db")
	if err := os.MkdirAll(commandDir, 0755); err != nil {
		t.Fatalf("mkdir seed command: %v", err)
	}

	originalGetwd := seedGetwd
	originalRunCommand := launcherRunCommand
	t.Cleanup(func() {
		seedGetwd = originalGetwd
		launcherRunCommand = originalRunCommand
	})
	seedGetwd = func() (string, error) { return root, nil }

	var capturedCommand string
	var capturedArgs []string
	var capturedCWD string
	var capturedEnv []string
	launcherRunCommand = func(command string, args []string, cwd string, env []string) (string, error) {
		capturedCommand = command
		capturedArgs = append([]string(nil), args...)
		capturedCWD = cwd
		capturedEnv = append([]string(nil), env...)
		return "seeded test DB", nil
	}

	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	if err := (launcher{}).run([]string{"seed", "-json"}); err != nil {
		t.Fatalf("run seed: %v", err)
	}

	output, err := stdout()
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	var summary seedSummary
	if err := json.Unmarshal([]byte(output), &summary); err != nil {
		t.Fatalf("unmarshal seed summary: %v\n%s", err, output)
	}
	if !summary.OK {
		t.Fatalf("expected ok summary, got %#v", summary)
	}
	if capturedCommand != "go" {
		t.Fatalf("expected go command, got %q", capturedCommand)
	}
	if len(capturedArgs) != 2 || capturedArgs[0] != "run" || capturedArgs[1] != "." {
		t.Fatalf("expected go run ., got %#v", capturedArgs)
	}
	if capturedCWD != commandDir {
		t.Fatalf("expected seed cwd %q, got %q", commandDir, capturedCWD)
	}
	wantDB := filepath.Join(root, "examples", "100-ai-chat-wizard", "bin", "runtime", "test_chat.db")
	if summary.DatabasePath != wantDB {
		t.Fatalf("expected summary db path %q, got %#v", wantDB, summary)
	}
	if !envContains(capturedEnv, "CHAT_DB_PATH="+wantDB) {
		t.Fatalf("expected CHAT_DB_PATH in env, got %#v", capturedEnv)
	}
	if len(summary.Credentials) != 2 {
		t.Fatalf("expected known credentials, got %#v", summary)
	}
	if summary.Credentials[0].Email != "demo@example.com" || summary.Credentials[1].Email != "admin@example.com" {
		t.Fatalf("unexpected credential summary: %#v", summary.Credentials)
	}
}

func TestRunSeedCommandOverrideUsesProvidedPathAndDB(t *testing.T) {
	root := t.TempDir()
	commandDir := filepath.Join(root, "cmd", "seed")
	mainPath := filepath.Join(commandDir, "main.go")
	if err := os.MkdirAll(commandDir, 0755); err != nil {
		t.Fatalf("mkdir seed command: %v", err)
	}
	if err := os.WriteFile(mainPath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write seed main.go: %v", err)
	}

	originalRunCommand := launcherRunCommand
	t.Cleanup(func() { launcherRunCommand = originalRunCommand })

	var capturedCWD string
	var capturedEnv []string
	launcherRunCommand = func(command string, args []string, cwd string, env []string) (string, error) {
		capturedCWD = cwd
		capturedEnv = append([]string(nil), env...)
		return "seed ok", nil
	}

	dbPath := filepath.Join(root, "fixtures", "dev.db")
	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	if err := (launcher{}).run([]string{"seed", "-root", root, "-command", mainPath, "-db-path", dbPath, "-json"}); err != nil {
		t.Fatalf("run seed with command override: %v", err)
	}

	output, err := stdout()
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	var summary seedSummary
	if err := json.Unmarshal([]byte(output), &summary); err != nil {
		t.Fatalf("unmarshal seed summary: %v\n%s", err, output)
	}
	if capturedCWD != commandDir {
		t.Fatalf("expected command dir %q, got %q", commandDir, capturedCWD)
	}
	if summary.CommandPath != commandDir {
		t.Fatalf("expected summary command dir %q, got %#v", commandDir, summary)
	}
	if summary.DatabasePath != dbPath {
		t.Fatalf("expected summary db path %q, got %#v", dbPath, summary)
	}
	if !envContains(capturedEnv, "CHAT_DB_PATH="+dbPath) {
		t.Fatalf("expected db override in env, got %#v", capturedEnv)
	}
	if len(summary.Credentials) != 0 {
		t.Fatalf("expected no built-in credentials for custom seeder, got %#v", summary)
	}
}

func TestRunSeedReportsMissingSeeder(t *testing.T) {
	err := (launcher{}).run([]string{"seed", "-root", t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "no seed command found") {
		t.Fatalf("expected missing seeder error, got %v", err)
	}
}

func TestRunSeedHandlesHelpAndInvalidFlags(t *testing.T) {
	if err := (launcher{}).runSeed([]string{"-help"}); err != nil {
		t.Fatalf("expected help to succeed, got %v", err)
	}
	if err := (launcher{}).runSeed([]string{"-definitely-invalid"}); err == nil || !strings.Contains(err.Error(), "flag provided but not defined") {
		t.Fatalf("expected invalid flag error, got %v", err)
	}
}

func envContains(env []string, want string) bool {
	for _, entry := range env {
		if entry == want {
			return true
		}
	}
	return false
}
