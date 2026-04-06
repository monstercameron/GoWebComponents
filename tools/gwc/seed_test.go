package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveSeedConfigDefaultsToChatWizardSeeder(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseCommandDir := filepath.Join(parseRoot, "examples", "100-ai-chat-wizard", "cmd", "seed-test-db")
	if parseErr := os.MkdirAll(parseCommandDir, 0755); parseErr != nil {
		parseT.Fatalf("mkdir seed command: %v", parseErr)
	}

	parseOriginalGetwd := seedGetwd
	parseT.Cleanup(func() { seedGetwd = parseOriginalGetwd })
	seedGetwd = func() (string, error) { return parseRoot, nil }

	parseConfig, parseErr2 := resolveSeedConfig(seedConfig{})
	if parseErr2 != nil {
		parseT.Fatalf("resolve seed config: %v", parseErr2)
	}
	if parseConfig.rootPath != parseRoot {
		parseT.Fatalf("expected root path %q, got %#v", parseRoot, parseConfig)
	}
	if parseConfig.commandPath != parseCommandDir {
		parseT.Fatalf("expected command path %q, got %#v", parseCommandDir, parseConfig)
	}
	parseWantDB := filepath.Join(parseRoot, "examples", "100-ai-chat-wizard", "bin", "runtime", "chat_history.db")
	if parseConfig.dbPath != parseWantDB {
		parseT.Fatalf("expected db path %q, got %#v", parseWantDB, parseConfig)
	}
}

func TestRunSeedJSONExecutesDefaultSeederWithKnownCredentials(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseCommandDir := filepath.Join(parseRoot, "examples", "100-ai-chat-wizard", "cmd", "seed-test-db")
	if parseErr := os.MkdirAll(parseCommandDir, 0755); parseErr != nil {
		parseT.Fatalf("mkdir seed command: %v", parseErr)
	}

	parseOriginalGetwd := seedGetwd
	parseOriginalRunCommand := launcherRunCommand
	parseT.Cleanup(func() {
		seedGetwd = parseOriginalGetwd
		launcherRunCommand = parseOriginalRunCommand
	})
	seedGetwd = func() (string, error) { return parseRoot, nil }

	var parseCapturedCommand string
	var parseCapturedArgs []string
	var parseCapturedCWD string
	var parseCapturedEnv []string
	launcherRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
		parseCapturedCommand = parseCommand
		parseCapturedArgs = append([]string(nil), parseArgs...)
		parseCapturedCWD = parseCwd
		parseCapturedEnv = append([]string(nil), parseEnv...)
		return "seeded test DB", nil
	}

	parseStdout, parseRestoreStdout, parseErr2 := captureExamplesStdout()
	if parseErr2 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr2)
	}
	defer parseRestoreStdout()

	if parseErr3 := (launcher{}).run([]string{"seed", "-json"}); parseErr3 != nil {
		parseT.Fatalf("run seed: %v", parseErr3)
	}

	parseOutput, parseErr2 := parseStdout()
	if parseErr2 != nil {
		parseT.Fatalf("read stdout: %v", parseErr2)
	}
	var parseSummary seedSummary
	if parseErr4 := json.Unmarshal([]byte(parseOutput), &parseSummary); parseErr4 != nil {
		parseT.Fatalf("unmarshal seed summary: %v\n%s", parseErr4, parseOutput)
	}
	if !parseSummary.OK {
		parseT.Fatalf("expected ok summary, got %#v", parseSummary)
	}
	if parseCapturedCommand != "go" {
		parseT.Fatalf("expected go command, got %q", parseCapturedCommand)
	}
	if len(parseCapturedArgs) != 2 || parseCapturedArgs[0] != "run" || parseCapturedArgs[1] != "." {
		parseT.Fatalf("expected go run ., got %#v", parseCapturedArgs)
	}
	if parseCapturedCWD != parseCommandDir {
		parseT.Fatalf("expected seed cwd %q, got %q", parseCommandDir, parseCapturedCWD)
	}
	parseWantDB := filepath.Join(parseRoot, "examples", "100-ai-chat-wizard", "bin", "runtime", "chat_history.db")
	if parseSummary.DatabasePath != parseWantDB {
		parseT.Fatalf("expected summary db path %q, got %#v", parseWantDB, parseSummary)
	}
	if !envContains(parseCapturedEnv, "CHAT_DB_PATH="+parseWantDB) {
		parseT.Fatalf("expected CHAT_DB_PATH in env, got %#v", parseCapturedEnv)
	}
	if len(parseSummary.Credentials) != 2 {
		parseT.Fatalf("expected known credentials, got %#v", parseSummary)
	}
	if parseSummary.Credentials[0].Email != "customer@email.com" || parseSummary.Credentials[1].Email != "admin@email.com" {
		parseT.Fatalf("unexpected credential summary: %#v", parseSummary.Credentials)
	}
}

func TestRunSeedCommandOverrideUsesProvidedPathAndDB(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseCommandDir := filepath.Join(parseRoot, "cmd", "seed")
	parseMainPath := filepath.Join(parseCommandDir, "main.go")
	if parseErr := os.MkdirAll(parseCommandDir, 0755); parseErr != nil {
		parseT.Fatalf("mkdir seed command: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseMainPath, []byte("package main\nfunc main() {}\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write seed main.go: %v", parseErr2)
	}

	parseOriginalRunCommand := launcherRunCommand
	parseT.Cleanup(func() { launcherRunCommand = parseOriginalRunCommand })

	var parseCapturedCWD string
	var parseCapturedEnv []string
	launcherRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
		parseCapturedCWD = parseCwd
		parseCapturedEnv = append([]string(nil), parseEnv...)
		return "seed ok", nil
	}

	parseDbPath := filepath.Join(parseRoot, "fixtures", "dev.db")
	parseStdout, parseRestoreStdout, parseErr3 := captureExamplesStdout()
	if parseErr3 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr3)
	}
	defer parseRestoreStdout()

	if parseErr4 := (launcher{}).run([]string{"seed", "-root", parseRoot, "-command", parseMainPath, "-db-path", parseDbPath, "-json"}); parseErr4 != nil {
		parseT.Fatalf("run seed with command override: %v", parseErr4)
	}

	parseOutput, parseErr3 := parseStdout()
	if parseErr3 != nil {
		parseT.Fatalf("read stdout: %v", parseErr3)
	}
	var parseSummary seedSummary
	if parseErr5 := json.Unmarshal([]byte(parseOutput), &parseSummary); parseErr5 != nil {
		parseT.Fatalf("unmarshal seed summary: %v\n%s", parseErr5, parseOutput)
	}
	if parseCapturedCWD != parseCommandDir {
		parseT.Fatalf("expected command dir %q, got %q", parseCommandDir, parseCapturedCWD)
	}
	if parseSummary.CommandPath != parseCommandDir {
		parseT.Fatalf("expected summary command dir %q, got %#v", parseCommandDir, parseSummary)
	}
	if parseSummary.DatabasePath != parseDbPath {
		parseT.Fatalf("expected summary db path %q, got %#v", parseDbPath, parseSummary)
	}
	if !envContains(parseCapturedEnv, "CHAT_DB_PATH="+parseDbPath) {
		parseT.Fatalf("expected db override in env, got %#v", parseCapturedEnv)
	}
	if len(parseSummary.Credentials) != 0 {
		parseT.Fatalf("expected no built-in credentials for custom seeder, got %#v", parseSummary)
	}
}

func TestRunSeedReportsMissingSeeder(parseT *testing.T) {
	parseErr := (launcher{}).run([]string{"seed", "-root", parseT.TempDir()})
	if parseErr == nil || !strings.Contains(parseErr.Error(), "no seed command found") {
		parseT.Fatalf("expected missing seeder error, got %v", parseErr)
	}
}

func TestRunSeedHandlesHelpAndInvalidFlags(parseT *testing.T) {
	if parseErr := (launcher{}).runSeed([]string{"-help"}); parseErr != nil {
		parseT.Fatalf("expected help to succeed, got %v", parseErr)
	}
	if parseErr2 := (launcher{}).runSeed([]string{"-definitely-invalid"}); parseErr2 == nil || !strings.Contains(parseErr2.Error(), "flag provided but not defined") {
		parseT.Fatalf("expected invalid flag error, got %v", parseErr2)
	}
}

func TestPrintSeedSummary(parseT *testing.T) {
	parseStdout, parseRestoreStdout, parseErr := captureExamplesStdout()
	if parseErr != nil {
		parseT.Fatalf("capture stdout: %v", parseErr)
	}
	defer parseRestoreStdout()

	printSeedSummary(seedSummary{
		ProjectRoot:  "/repo",
		CommandPath:  "/repo/cmd/seed",
		DatabasePath: "/repo/bin/runtime/test.db",
		Credentials: []seedCredentialRecord{
			{Email: "customer@email.com", Password: "password", Role: "customer"},
			{Email: "admin@email.com", Password: "password", Role: "admin"},
		},
		Output: "seed complete",
	})

	parseOutput, parseErr := parseStdout()
	if parseErr != nil {
		parseT.Fatalf("read stdout: %v", parseErr)
	}
	for _, parseWant := range []string{
		"GWC seed",
		"project root: /repo",
		"command:      /repo/cmd/seed",
		"database:     /repo/bin/runtime/test.db",
		"account:      customer@email.com / password (customer)",
		"account:      admin@email.com / password (admin)",
		"output:       seed complete",
	} {
		if !strings.Contains(parseOutput, parseWant) {
			parseT.Fatalf("expected seed summary output to contain %q\n%s", parseWant, parseOutput)
		}
	}
}

func envContains(parseEnv []string, parseWant string) bool {
	for _, parseEntry := range parseEnv {
		if parseEntry == parseWant {
			return true
		}
	}
	return false
}
