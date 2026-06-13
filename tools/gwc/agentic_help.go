package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
)

// runHelpCommand routes the structured help command.
var runHelpCommand = func(parseL launcher, parseArgs []string) error {
	return parseL.runHelp(parseArgs)
}

type gwcHelpFlag struct {
	Name        string `json:"name"`
	Type        string `json:"type,omitempty"`
	Default     string `json:"default,omitempty"`
	Description string `json:"description"`
}

type gwcHelpCommand struct {
	Name         string        `json:"name"`
	Aliases      []string      `json:"aliases,omitempty"`
	Category     string        `json:"category"`
	Summary      string        `json:"summary"`
	Usage        string        `json:"usage"`
	Examples     []string      `json:"examples"`
	Flags        []gwcHelpFlag `json:"flags,omitempty"`
	JSON         bool          `json:"json"`
	ReadOnly     bool          `json:"readOnly"`
	Mutating     bool          `json:"mutating"`
	Experimental bool          `json:"experimental,omitempty"`
}

type gwcHelpReport struct {
	SchemaVersion string           `json:"schemaVersion"`
	Command       string           `json:"command,omitempty"`
	Commands      []gwcHelpCommand `json:"commands,omitempty"`
}

// runHelp prints root or command-specific help in human or JSON form.
func (parseL launcher) runHelp(parseArgs []string) error {
	parseArgs = reorderAgenticKnownFlags(parseArgs, []string{"command"}, []string{"json"})
	parseFlags := flag.NewFlagSet("help", flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parseJSON := parseFlags.Bool("json", false, "Emit structured help metadata")
	parseCommandFlag := parseFlags.String("command", "", "Optional command name to describe")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}

	parseCommand := strings.TrimSpace(*parseCommandFlag)
	if parseCommand == "" && len(parseFlags.Args()) > 0 {
		parseCommand = strings.TrimSpace(parseFlags.Args()[0])
	}
	parseReport, parseErr := buildGwcHelpReport(parseCommand)
	if parseErr != nil {
		return parseErr
	}
	if *parseJSON {
		return writeAgenticEnvelope("help", true, parseReport, nil, nil)
	}
	if parseCommand != "" && len(parseReport.Commands) == 1 {
		printGwcCommandHelp(parseReport.Commands[0])
		return nil
	}
	printUsage()
	return nil
}

// buildGwcHelpReport returns deterministic help metadata for all known commands.
func buildGwcHelpReport(parseCommand string) (gwcHelpReport, error) {
	parseCommands := listGwcHelpCommands()
	parseReport := gwcHelpReport{
		SchemaVersion: agenticEnvelopeSchemaVersion,
	}
	parseCommand = strings.ToLower(strings.TrimSpace(parseCommand))
	if parseCommand == "" {
		parseReport.Commands = parseCommands
		return parseReport, nil
	}
	for _, parseCandidate := range parseCommands {
		if parseCandidate.Name == parseCommand || stringSliceContains(parseCandidate.Aliases, parseCommand) {
			parseReport.Command = parseCandidate.Name
			parseReport.Commands = []gwcHelpCommand{parseCandidate}
			return parseReport, nil
		}
	}
	parseSuggestion := suggestGwcHelpCommand(parseCommand, parseCommands)
	if parseSuggestion != "" {
		return gwcHelpReport{}, fmt.Errorf("unknown help command %q; did you mean %q", parseCommand, parseSuggestion)
	}
	return gwcHelpReport{}, fmt.Errorf("unknown help command %q", parseCommand)
}

// printGwcCommandHelp renders a single command's human help page.
func printGwcCommandHelp(parseCommand gwcHelpCommand) {
	fmt.Printf("GWC %s\n\n", parseCommand.Name)
	fmt.Println(parseCommand.Summary)
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Printf("  %s\n", parseCommand.Usage)
	if len(parseCommand.Flags) > 0 {
		fmt.Println()
		fmt.Println("Flags:")
		for _, parseFlag := range parseCommand.Flags {
			parseDefault := ""
			if parseFlag.Default != "" {
				parseDefault = fmt.Sprintf(" (default %s)", parseFlag.Default)
			}
			fmt.Printf("  -%s <%s>%s\n      %s\n", parseFlag.Name, firstNonEmpty(parseFlag.Type, "value"), parseDefault, parseFlag.Description)
		}
	}
	if len(parseCommand.Examples) > 0 {
		fmt.Println()
		fmt.Println("Examples:")
		for _, parseExample := range parseCommand.Examples {
			fmt.Printf("  %s\n", parseExample)
		}
	}
	if parseCommand.JSON {
		fmt.Println()
		fmt.Println("JSON:")
		fmt.Println("  Supports -json for machine-readable output.")
	}
}

// listGwcHelpCommands returns the CLI command registry used by help and MCP.
func listGwcHelpCommands() []gwcHelpCommand {
	parseCommands := []gwcHelpCommand{
		{Name: "bench", Aliases: []string{"benchmark"}, Category: "verify", Summary: "Discover, run, capture, and compare native/js-wasm benchmarks.", Usage: "go run ./tools/gwc bench [capture|compare] [flags]", Examples: []string{"go run ./tools/gwc bench -json", "go run ./tools/gwc bench compare -baseline old.txt -candidate new.txt"}, JSON: true, ReadOnly: true},
		{Name: "build", Category: "verify", Summary: "Build a js/wasm app with an explicit launcher profile.", Usage: "go run ./tools/gwc build -app ./main.go -root . -json", Examples: []string{"go run ./tools/gwc build -app ./main.go -root . -profile ci -json"}, Flags: []gwcHelpFlag{{Name: "app", Type: "path", Description: "App entrypoint path"}, {Name: "root", Type: "path", Description: "Project root"}, {Name: "profile", Type: "string", Default: "development", Description: "Build profile"}, {Name: "json", Type: "bool", Default: "false", Description: "Emit JSON output"}}, JSON: true, ReadOnly: true},
		{Name: "check", Category: "agent", Summary: "Run agent-shaped diagnostics across Go tests and source conventions.", Usage: "go run ./tools/gwc check -root . -json", Examples: []string{"go run ./tools/gwc check -root . -skip-tests -json"}, Flags: []gwcHelpFlag{{Name: "root", Type: "path", Description: "Project root to check"}, {Name: "pattern", Type: "string", Default: "./...", Description: "go test package pattern"}, {Name: "skip-tests", Type: "bool", Default: "false", Description: "Skip go test execution"}, {Name: "skip-conventions", Type: "bool", Default: "false", Description: "Skip source convention diagnostics"}, {Name: "json", Type: "bool", Default: "false", Description: "Emit JSON envelope"}}, JSON: true, ReadOnly: true, Experimental: true},
		{Name: "clean", Category: "act", Summary: "Remove launcher-owned build artifacts, caches, and generated outputs.", Usage: "go run ./tools/gwc clean -root . -dry-run -json", Examples: []string{"go run ./tools/gwc clean -root . -dry-run -json", "go run ./tools/gwc clean -root . -cache -json"}, Flags: []gwcHelpFlag{{Name: "root", Type: "path", Description: "Project root to clean"}, {Name: "artifacts", Type: "bool", Default: "false", Description: "Clean generated artifacts"}, {Name: "cache", Type: "bool", Default: "false", Description: "Clean launcher caches"}, {Name: "bin", Type: "bool", Default: "false", Description: "Clean bin outputs"}, {Name: "dry-run", Type: "bool", Default: "false", Description: "List removals without deleting"}, {Name: "json", Type: "bool", Default: "false", Description: "Emit JSON envelope"}}, JSON: true, ReadOnly: false, Mutating: true, Experimental: true},
		{Name: "deadcode", Category: "agent", Summary: "Report exported symbols with no static in-repo dependents.", Usage: "go run ./tools/gwc deadcode -root . -json", Examples: []string{"go run ./tools/gwc deadcode -root . -json"}, Flags: []gwcHelpFlag{{Name: "root", Type: "path", Description: "Project root to analyze"}, {Name: "json", Type: "bool", Default: "false", Description: "Emit JSON envelope"}}, JSON: true, ReadOnly: true, Experimental: true},
		{Name: "deps", Aliases: []string{"update"}, Category: "agent", Summary: "Inspect Go module dependencies and apply guarded dependency updates.", Usage: "go run ./tools/gwc deps -root . [-latest] [-module path -to version] -json", Examples: []string{"go run ./tools/gwc deps -root . -latest -json", "go run ./tools/gwc deps -root . -module example.com/lib -to v1.2.3 -dry-run -json"}, Flags: []gwcHelpFlag{{Name: "root", Type: "path", Description: "Project root containing go.mod"}, {Name: "latest", Type: "bool", Default: "false", Description: "Ask go list for available updates"}, {Name: "module", Type: "string", Description: "Module path to update"}, {Name: "to", Type: "version", Description: "Version to update to"}, {Name: "dry-run", Type: "bool", Default: "false", Description: "Report planned update without writing"}, {Name: "json", Type: "bool", Default: "false", Description: "Emit JSON envelope"}}, JSON: true, ReadOnly: false, Mutating: true, Experimental: true},
		{Name: "dev", Category: "run", Summary: "Run the native gwc dev orchestration path with integrated live reload.", Usage: "go run ./tools/gwc dev [flags]", Examples: []string{"go run ./tools/gwc dev -app ./main.go -root . -json"}, JSON: true, ReadOnly: false},
		{Name: "doctor", Category: "verify", Summary: "Check local toolchains, runtime assets, project signals, and optional audit anchors.", Usage: "go run ./tools/gwc doctor [flags]", Examples: []string{"go run ./tools/gwc doctor -json"}, JSON: true, ReadOnly: true},
		{Name: "docs", Category: "agent", Summary: "Generate exported API documentation from the static GWC model.", Usage: "go run ./tools/gwc docs -root . [-out docs/api.md] -json", Examples: []string{"go run ./tools/gwc docs -root . -json", "go run ./tools/gwc docs -root . -out docs/api.md -json"}, Flags: []gwcHelpFlag{{Name: "root", Type: "path", Description: "Project root to document"}, {Name: "out", Type: "path", Description: "Optional markdown output path"}, {Name: "json", Type: "bool", Default: "false", Description: "Emit JSON envelope"}}, JSON: true, ReadOnly: false, Mutating: true, Experimental: true},
		{Name: "env", Category: "inspect", Summary: "Print launcher-relevant environment variables and current values.", Usage: "go run ./tools/gwc env -json", Examples: []string{"go run ./tools/gwc env -json"}, JSON: true, ReadOnly: true},
		{Name: "examples", Category: "run", Summary: "Serve the examples catalog or manage example-server lifecycle actions.", Usage: "go run ./tools/gwc examples [flags|path action]", Examples: []string{"go run ./tools/gwc examples -json"}, JSON: true, ReadOnly: false},
		{Name: "export-test", Category: "agent", Summary: "Generate a testkit Go test from a live-session recording.", Usage: "go run ./tools/gwc export-test -recording recording.json -component App -out app_recording_test.go -json", Examples: []string{"go run ./tools/gwc export-test -recording counter.json -component CounterApp -out counter_recording_test.go -json"}, Flags: []gwcHelpFlag{{Name: "recording", Type: "path", Description: "Recording JSON path"}, {Name: "out", Type: "path", Description: "Optional Go test file to write"}, {Name: "package", Type: "identifier", Description: "Generated test package name"}, {Name: "component", Type: "expr", Description: "Root component expression"}, {Name: "test-name", Type: "identifier", Description: "Generated test function name"}, {Name: "json", Type: "bool", Default: "false", Description: "Emit JSON envelope"}}, JSON: true, ReadOnly: false, Mutating: true, Experimental: true},
		{Name: "files", Category: "inspect", Summary: "List project files with repeatable extension and directory filters.", Usage: "go run ./tools/gwc files -root . -json", Examples: []string{"go run ./tools/gwc files -root . -ext go -json"}, JSON: true, ReadOnly: true},
		{Name: "fmt", Category: "agent", Summary: "Format Go source, normalize line endings, and report/fix GWC doc-comment conventions.", Usage: "go run ./tools/gwc fmt -root . [-check] -json", Examples: []string{"go run ./tools/gwc fmt -root . -check -json", "go run ./tools/gwc fmt -root . -json"}, Flags: []gwcHelpFlag{{Name: "root", Type: "path", Description: "Project root to format"}, {Name: "check", Type: "bool", Default: "false", Description: "Report formatting changes without writing"}, {Name: "json", Type: "bool", Default: "false", Description: "Emit JSON envelope"}}, JSON: true, ReadOnly: false, Mutating: true, Experimental: true},
		{Name: "help", Aliases: []string{"-h", "--help"}, Category: "agent", Summary: "Print human help or structured command metadata.", Usage: "go run ./tools/gwc help [command] [--json]", Examples: []string{"go run ./tools/gwc help --json", "go run ./tools/gwc help mutate"}, Flags: []gwcHelpFlag{{Name: "command", Type: "string", Description: "Optional command to describe"}, {Name: "json", Type: "bool", Default: "false", Description: "Emit JSON envelope"}}, JSON: true, ReadOnly: true},
		{Name: "import", Category: "act", Summary: "Convert a static HTML or JSX file into an inspectable GWC project.", Usage: "go run ./tools/gwc import -src input.html -out ./generated -json", Examples: []string{"go run ./tools/gwc import -src ./page.html -out ./generated -json"}, JSON: true, ReadOnly: false, Mutating: true},
		{Name: "init", Category: "act", Summary: "Write lifecycle metadata and starter defaults without the scaffold TUI.", Usage: "go run ./tools/gwc init -root . -json", Examples: []string{"go run ./tools/gwc init -root . -force -json"}, JSON: true, ReadOnly: false, Mutating: true},
		{Name: "inspect", Category: "inspect", Summary: "Build higher-level route, dependency, ownership, file-type, and symbol-impact reports.", Usage: "go run ./tools/gwc inspect -root . [-impact Symbol] -json", Examples: []string{"go run ./tools/gwc inspect -root . -json", "go run ./tools/gwc inspect -root . -impact App -json"}, Flags: []gwcHelpFlag{{Name: "root", Type: "path", Description: "Project root to inspect"}, {Name: "impact", Type: "symbol", Description: "Return direct and transitive dependents for a component, atom, or exported symbol"}, {Name: "json", Type: "bool", Default: "false", Description: "Emit JSON output"}}, JSON: true, ReadOnly: true},
		{Name: "lint", Aliases: []string{"review"}, Category: "verify", Summary: "Run golangci-lint plus built-in GWC hook rules.", Usage: "go run ./tools/gwc lint -root . -json", Examples: []string{"go run ./tools/gwc lint -root . -json"}, JSON: true, ReadOnly: true},
		{Name: "mcp", Category: "agent", Summary: "Serve JSON-capable gwc commands as MCP tools over stdio.", Usage: "go run ./tools/gwc mcp [--json]", Examples: []string{"go run ./tools/gwc mcp --json", "go run ./tools/gwc mcp"}, Flags: []gwcHelpFlag{{Name: "json", Type: "bool", Default: "false", Description: "Emit the MCP tool manifest instead of starting the server"}}, JSON: true, ReadOnly: true, Experimental: true},
		{Name: "migrate", Category: "act", Summary: "Run compatibility API findings and optional safe rewrites.", Usage: "go run ./tools/gwc migrate -root . -json", Examples: []string{"go run ./tools/gwc migrate -root . -apply -json"}, JSON: true, ReadOnly: false, Mutating: true},
		{Name: "model", Category: "agent", Summary: "Emit a static component/API manifest with props, hooks, atoms, events, symbols, and references.", Usage: "go run ./tools/gwc model -root . -json", Examples: []string{"go run ./tools/gwc model -root . -json"}, Flags: []gwcHelpFlag{{Name: "root", Type: "path", Description: "Root directory to model"}, {Name: "json", Type: "bool", Default: "false", Description: "Emit JSON output"}}, JSON: true, ReadOnly: true, Experimental: true},
		{Name: "mutate", Category: "agent", Summary: "Apply safe AST-backed source mutations with dry-run and JSON diff output.", Usage: "go run ./tools/gwc mutate rename-ident -root . -from Old -to New -json", Examples: []string{"go run ./tools/gwc mutate rename-ident -root . -from OldName -to NewName -dry-run -json"}, Flags: []gwcHelpFlag{{Name: "root", Type: "path", Description: "Project root to mutate"}, {Name: "from", Type: "identifier", Description: "Identifier to rename"}, {Name: "to", Type: "identifier", Description: "Replacement identifier"}, {Name: "dry-run", Type: "bool", Default: "false", Description: "Report edits without writing files"}, {Name: "json", Type: "bool", Default: "false", Description: "Emit JSON envelope"}}, JSON: true, ReadOnly: false, Mutating: true, Experimental: true},
		{Name: "explain", Category: "agent", Summary: "Resolve a GWC diagnostic code or capability to docs, cause, and remediation.", Usage: "go run ./tools/gwc explain GWC-HYDRATION-MISMATCH -json", Examples: []string{"go run ./tools/gwc explain GWC-RUNTIME-PANIC-RENDER -json", "go run ./tools/gwc explain \"SSR & hydration\" -json"}, Flags: []gwcHelpFlag{{Name: "root", Type: "path", Description: "Repository root used for generated error-code metadata"}, {Name: "json", Type: "bool", Default: "false", Description: "Emit JSON output"}}, JSON: true, ReadOnly: true, Experimental: true},
		{Name: "observe", Category: "agent", Summary: "Query redacted runtime/crash telemetry records by route, build, and severity.", Usage: "go run ./tools/gwc observe -source telemetry.ndjson -json", Examples: []string{"go run ./tools/gwc observe -source telemetry.ndjson -severity error -json", "go run ./tools/gwc observe -source telemetry.ndjson -agent"}, Flags: []gwcHelpFlag{{Name: "source", Type: "path", Description: "JSONL/NDJSON telemetry source; repeatable"}, {Name: "route", Type: "string", Description: "Route filter"}, {Name: "build", Type: "string", Description: "Build id/version filter"}, {Name: "severity", Type: "string", Description: "Minimum severity"}, {Name: "agent", Type: "bool", Default: "false", Description: "Emit NDJSON agent events"}, {Name: "json", Type: "bool", Default: "false", Description: "Emit JSON output"}}, JSON: true, ReadOnly: true, Experimental: true},
		{Name: "probe", Category: "agent", Summary: "Run a browser-oracle probe for a URL or example target and return DOM, console, and error evidence.", Usage: "go run ./tools/gwc probe -url http://127.0.0.1:8090 -json", Examples: []string{"go run ./tools/gwc probe -url http://127.0.0.1:8090 -json"}, Flags: []gwcHelpFlag{{Name: "root", Type: "path", Description: "Project root"}, {Name: "url", Type: "url", Description: "Running page to probe"}, {Name: "timeout", Type: "duration", Default: "10s", Description: "Browser probe timeout"}, {Name: "json", Type: "bool", Default: "false", Description: "Emit JSON output"}}, JSON: true, ReadOnly: true, Experimental: true},
		{Name: "prerender", Aliases: []string{"export"}, Category: "act", Summary: "Build a static export output with route HTML, wasm artifacts, and a manifest.", Usage: "go run ./tools/gwc prerender [flags]", Examples: []string{"go run ./tools/gwc prerender -json"}, JSON: true, ReadOnly: false, Mutating: true},
		{Name: "release", Category: "act", Summary: "Package a js/wasm release with manifest and compressed sidecars.", Usage: "go run ./tools/gwc release [flags]", Examples: []string{"go run ./tools/gwc release -app ./main.go -root . -json"}, JSON: true, ReadOnly: false, Mutating: true},
		{Name: "rebuild", Category: "agent", Summary: "Rebuild a live agent session and restore state through the hub JSON API.", Usage: "go run ./tools/gwc rebuild -hub http://127.0.0.1:8090 -token $GWC_AGENT_TOKEN -app ./main.go -json", Examples: []string{"go run ./tools/gwc rebuild -hub http://127.0.0.1:8090 -token dev -session sess-1 -app ./main.go -json"}, Flags: []gwcHelpFlag{{Name: "hub", Type: "url", Description: "Agent hub base URL"}, {Name: "token", Type: "string", Description: "Agent hub token"}, {Name: "session", Type: "id", Description: "Optional live session id"}, {Name: "app", Type: "path", Description: "App entrypoint path"}, {Name: "root", Type: "path", Description: "Project root"}, {Name: "out", Type: "path", Description: "WASM output path"}, {Name: "profile", Type: "string", Default: "development", Description: "Build profile"}, {Name: "timeout", Type: "duration", Default: "10s", Description: "Successor/apply timeout"}, {Name: "json", Type: "bool", Default: "false", Description: "Emit JSON envelope"}}, JSON: true, ReadOnly: false, Mutating: true, Experimental: true},
		{Name: "render", Category: "agent", Summary: "Render a component through a generated headless SSR oracle.", Usage: "go run ./tools/gwc render ComponentName -root . -props '{}' -json", Examples: []string{"go run ./tools/gwc render App -root . -props '{}' -json"}, Flags: []gwcHelpFlag{{Name: "root", Type: "path", Description: "Project root"}, {Name: "props", Type: "json", Default: "{}", Description: "Props JSON object"}, {Name: "json", Type: "bool", Default: "false", Description: "Emit JSON output"}}, JSON: true, ReadOnly: true, Experimental: true},
		{Name: "scaffold", Category: "agent", Summary: "Generate components, hooks, examples, or starter apps without prompts.", Usage: "go run ./tools/gwc scaffold component -name AppCard -root . -json --no-input", Examples: []string{"go run ./tools/gwc scaffold component -name ProfileCard -root . -json --no-input", "go run ./tools/gwc scaffold hook -name Search -root . -dry-run -json --no-input"}, Flags: []gwcHelpFlag{{Name: "root", Type: "path", Description: "Project root for generated files"}, {Name: "dir", Type: "path", Description: "Output directory relative to root"}, {Name: "name", Type: "identifier", Description: "Generated symbol name"}, {Name: "package", Type: "identifier", Description: "Go package name"}, {Name: "dry-run", Type: "bool", Default: "false", Description: "Report planned files without writing"}, {Name: "no-input", Type: "bool", Default: "false", Description: "Confirm non-interactive mode"}, {Name: "json", Type: "bool", Default: "false", Description: "Emit JSON envelope"}}, JSON: true, ReadOnly: false, Mutating: true, Experimental: true},
		{Name: "search", Category: "agent", Summary: "Search exported APIs by intent using the static manifest and capability tags.", Usage: "go run ./tools/gwc search \"persist state across reload\" -root . -json", Examples: []string{"go run ./tools/gwc search \"persist state across reload\" -root . -json", "go run ./tools/gwc search \"trap focus\" -root . -json"}, Flags: []gwcHelpFlag{{Name: "root", Type: "path", Description: "Root directory to index"}, {Name: "limit", Type: "int", Default: "10", Description: "Maximum result count"}, {Name: "json", Type: "bool", Default: "false", Description: "Emit JSON output"}}, JSON: true, ReadOnly: true, Experimental: true},
		{Name: "seed", Category: "act", Summary: "Provision local dev identities and fixture data through a seed package.", Usage: "go run ./tools/gwc seed -root . -json", Examples: []string{"go run ./tools/gwc seed -root . -json"}, JSON: true, ReadOnly: false, Mutating: true},
		{Name: "serve", Category: "run", Summary: "Serve static assets, wasm artifacts, wasm_exec.js, and optional fixtures.", Usage: "go run ./tools/gwc serve [flags]", Examples: []string{"go run ./tools/gwc serve -root . -port 8090"}, ReadOnly: true},
		{Name: "size", Category: "agent", Summary: "Attribute wasm/native artifact size by package and symbol using go tool nm.", Usage: "go run ./tools/gwc size -artifact ./app.wasm -json", Examples: []string{"go run ./tools/gwc size -artifact ./dist/app.wasm -limit 25 -json"}, Flags: []gwcHelpFlag{{Name: "artifact", Type: "path", Description: "Wasm or native artifact path to attribute"}, {Name: "limit", Type: "int", Default: "20", Description: "Maximum symbols/packages to return"}, {Name: "json", Type: "bool", Default: "false", Description: "Emit JSON envelope"}}, JSON: true, ReadOnly: true, Experimental: true},
		{Name: "screenshot", Category: "agent", Summary: "Capture a pixel-perfect PNG of a running app via a CDP-driven headless Chromium (--no-sandbox). The DevTools/browser-proxy backend of gwc mcp for pixels the in-wasm bridge can't see. Requires building with -tags playwrightgo.", Usage: "go run -tags playwrightgo ./tools/gwc screenshot -url http://127.0.0.1:8099 -out bin/shot.png -json", Examples: []string{"go run -tags playwrightgo ./tools/gwc screenshot -url http://127.0.0.1:8099 -out bin/shot.png -json", "go run -tags playwrightgo ./tools/gwc screenshot -url http://127.0.0.1:8099 -selector '#app' -out bin/app.png -json"}, Flags: []gwcHelpFlag{{Name: "url", Type: "string", Description: "URL to open and capture (required)"}, {Name: "out", Type: "path", Default: "bin/screenshot.png", Description: "Output PNG path"}, {Name: "selector", Type: "string", Description: "Optional CSS selector to capture instead of the full page"}, {Name: "width", Type: "int", Default: "1280", Description: "Viewport width"}, {Name: "height", Type: "int", Default: "800", Description: "Viewport height"}, {Name: "headless", Type: "bool", Default: "true", Description: "Run Chromium headless"}, {Name: "json", Type: "bool", Default: "false", Description: "Emit JSON envelope"}}, JSON: true, ReadOnly: true, Experimental: true},
		{Name: "start", Category: "act", Summary: "Open the scaffold TUI for preset and project setup.", Usage: "go run ./tools/gwc start [flags]", Examples: []string{"go run ./tools/gwc start -mode standalone"}, ReadOnly: false, Mutating: true},
		{Name: "tailwind", Category: "act", Summary: "Build shared Tailwind CSS and generated class manifests.", Usage: "go run ./tools/gwc tailwind [flags]", Examples: []string{"go run ./tools/gwc tailwind -json"}, JSON: true, ReadOnly: false, Mutating: true},
		{Name: "test", Category: "verify", Summary: "Run launcher-owned test lanes such as unit, race, wasm, hydration, browser, agent, agent-browser, and release.", Usage: "go run ./tools/gwc test -lane unit -json", Examples: []string{"go run ./tools/gwc test -lane unit -lane race -lane wasm -json", "go run ./tools/gwc test -lane agent -lane agent-browser -json", "go run ./tools/gwc test -lane unit -watch"}, Flags: []gwcHelpFlag{{Name: "watch", Type: "bool", Default: "false", Description: "Rerun selected lanes when Go files change"}}, JSON: true, ReadOnly: true},
		{Name: "upgrade", Category: "act", Summary: "Upgrade gwc-start.json schema and runtime assets.", Usage: "go run ./tools/gwc upgrade -root . -json", Examples: []string{"go run ./tools/gwc upgrade -root . -json"}, JSON: true, ReadOnly: false, Mutating: true},
		{Name: "verify", Category: "verify", Summary: "Run app-local Go tests when present and perform a CI-profile wasm build.", Usage: "go run ./tools/gwc verify -app ./main.go -root . -json", Examples: []string{"go run ./tools/gwc verify -app ./main.go -root . -json"}, JSON: true, ReadOnly: true},
		{Name: "watch", Category: "agent", Summary: "Watch Go files and rerun selected launcher-owned test lanes.", Usage: "go run ./tools/gwc watch -root . -lane unit", Examples: []string{"go run ./tools/gwc watch -root . -lane unit", "go run ./tools/gwc watch -root . -lane unit -once -json"}, Flags: []gwcHelpFlag{{Name: "root", Type: "path", Description: "Project root to watch"}, {Name: "lane", Type: "string", Description: "Test lane to run; repeat or comma-separate"}, {Name: "once", Type: "bool", Default: "false", Description: "Run one watched test pass and exit"}, {Name: "debounce", Type: "duration", Default: "500ms", Description: "Polling debounce interval"}, {Name: "json", Type: "bool", Default: "false", Description: "Emit JSON envelope"}}, JSON: true, ReadOnly: true, Experimental: true},
		{Name: "sessions", Category: "agent", Summary: "List live gwc agent bridge sessions.", Usage: "go run ./tools/gwc sessions -hub http://127.0.0.1:8090 -token $GWC_AGENT_TOKEN -json", Examples: []string{"go run ./tools/gwc sessions -hub http://127.0.0.1:8090 -token dev -json"}, Flags: liveBridgeHelpFlags(false), JSON: true, ReadOnly: true, Experimental: true},
		{Name: "snapshot", Category: "agent", Summary: "Read a live agent bridge runtime snapshot.", Usage: "go run ./tools/gwc snapshot -hub http://127.0.0.1:8090 -token $GWC_AGENT_TOKEN -payload '{\"maxDepth\":4}' -json", Examples: []string{"go run ./tools/gwc snapshot -hub http://127.0.0.1:8090 -token dev -json"}, Flags: liveBridgeHelpFlags(false), JSON: true, ReadOnly: true, Experimental: true},
		{Name: "snapshot-diff", Category: "agent", Summary: "Diff two agent bridge snapshots by stable ref.", Usage: "go run ./tools/gwc snapshot-diff -before before.json -after after.json -json", Examples: []string{"go run ./tools/gwc snapshot-diff -before before.json -after after.json -json"}, Flags: []gwcHelpFlag{{Name: "before", Type: "path", Description: "Before snapshot JSON path"}, {Name: "after", Type: "path", Description: "After snapshot JSON path"}, {Name: "json", Type: "bool", Default: "false", Description: "Emit JSON envelope"}}, JSON: true, ReadOnly: true, Experimental: true},
		{Name: "query", Category: "agent", Summary: "Query live agent bridge nodes by semantic selectors.", Usage: "go run ./tools/gwc query -hub http://127.0.0.1:8090 -token $GWC_AGENT_TOKEN -payload '{\"role\":\"button\"}' -json", Examples: []string{"go run ./tools/gwc query -hub http://127.0.0.1:8090 -token dev -payload '{\"text\":\"Save\"}' -json"}, Flags: liveBridgeHelpFlags(false), JSON: true, ReadOnly: true, Experimental: true},
		{Name: "logs", Category: "agent", Summary: "Fetch retained live agent bridge logs and diagnostics.", Usage: "go run ./tools/gwc logs -hub http://127.0.0.1:8090 -token $GWC_AGENT_TOKEN -max 50 -json", Examples: []string{"go run ./tools/gwc logs -hub http://127.0.0.1:8090 -token dev -max 20 -json"}, Flags: liveBridgeHelpFlags(false), JSON: true, ReadOnly: true, Experimental: true},
		{Name: "crash-report", Category: "agent", Summary: "Fetch a live agent bridge socket-death crash report.", Usage: "go run ./tools/gwc crash-report -hub http://127.0.0.1:8090 -token $GWC_AGENT_TOKEN -session sess-1 -json", Examples: []string{"go run ./tools/gwc crash-report -hub http://127.0.0.1:8090 -token dev -json"}, Flags: liveBridgeHelpFlags(false), JSON: true, ReadOnly: true, Experimental: true},
		{Name: "recording", Category: "agent", Summary: "Fetch or clear a live agent bridge command recording.", Usage: "go run ./tools/gwc recording -hub http://127.0.0.1:8090 -token $GWC_AGENT_TOKEN -max 100 -json", Examples: []string{"go run ./tools/gwc recording -hub http://127.0.0.1:8090 -token dev -clear -json"}, Flags: liveBridgeHelpFlags(false), JSON: true, ReadOnly: false, Mutating: true, Experimental: true},
		{Name: "lease", Category: "agent", Summary: "Acquire, release, or explicitly steal a live agent bridge write lease.", Usage: "go run ./tools/gwc lease -hub http://127.0.0.1:8090 -token $GWC_AGENT_TOKEN -holder agent-a -action acquire -json", Examples: []string{"go run ./tools/gwc lease -hub http://127.0.0.1:8090 -token dev -holder agent-a -action steal -steal -json"}, Flags: liveBridgeHelpFlags(true), JSON: true, ReadOnly: false, Mutating: true, Experimental: true},
		{Name: "set-atom", Category: "agent", Summary: "Set a live agent bridge atom value.", Usage: "go run ./tools/gwc set-atom -hub http://127.0.0.1:8090 -token $GWC_AGENT_TOKEN -lease-holder agent-a -payload '{\"id\":\"theme\",\"value\":\"dark\"}' -json", Examples: []string{"go run ./tools/gwc set-atom -hub http://127.0.0.1:8090 -token dev -lease-holder agent-a -payload '{\"id\":\"count\",\"value\":1}' -json"}, Flags: liveBridgeHelpFlags(true), JSON: true, ReadOnly: false, Mutating: true, Experimental: true},
		{Name: "set-state", Category: "agent", Summary: "Set a live agent bridge hook state slot.", Usage: "go run ./tools/gwc set-state -hub http://127.0.0.1:8090 -token $GWC_AGENT_TOKEN -lease-holder agent-a -payload '{\"ref\":\"...\",\"slot\":0,\"value\":true}' -json", Examples: []string{"go run ./tools/gwc set-state -hub http://127.0.0.1:8090 -token dev -lease-holder agent-a -payload '{\"ref\":\"root\",\"slot\":0,\"value\":\"open\"}' -json"}, Flags: liveBridgeHelpFlags(true), JSON: true, ReadOnly: false, Mutating: true, Experimental: true},
		{Name: "mount", Category: "agent", Summary: "Mount a bridge-registered component into a live agent session.", Usage: "go run ./tools/gwc mount -hub http://127.0.0.1:8090 -token $GWC_AGENT_TOKEN -lease-holder agent-a -payload '{\"component\":\"Preview\"}' -json", Examples: []string{"go run ./tools/gwc mount -hub http://127.0.0.1:8090 -token dev -lease-holder agent-a -payload '{\"component\":\"Preview\"}' -json"}, Flags: liveBridgeHelpFlags(true), JSON: true, ReadOnly: false, Mutating: true, Experimental: true},
		{Name: "unmount", Category: "agent", Summary: "Unmount a bridge-mounted component from a live agent session.", Usage: "go run ./tools/gwc unmount -hub http://127.0.0.1:8090 -token $GWC_AGENT_TOKEN -lease-holder agent-a -payload '{\"mountId\":\"preview\"}' -json", Examples: []string{"go run ./tools/gwc unmount -hub http://127.0.0.1:8090 -token dev -lease-holder agent-a -payload '{\"mountId\":\"preview\"}' -json"}, Flags: liveBridgeHelpFlags(true), JSON: true, ReadOnly: false, Mutating: true, Experimental: true},
		{Name: "delete-atom", Category: "agent", Summary: "Delete a live agent bridge atom when safe or explicitly forced.", Usage: "go run ./tools/gwc delete-atom -hub http://127.0.0.1:8090 -token $GWC_AGENT_TOKEN -lease-holder agent-a -payload '{\"id\":\"draft\",\"force\":true}' -json", Examples: []string{"go run ./tools/gwc delete-atom -hub http://127.0.0.1:8090 -token dev -lease-holder agent-a -payload '{\"id\":\"draft\"}' -json"}, Flags: liveBridgeHelpFlags(true), JSON: true, ReadOnly: false, Mutating: true, Experimental: true},
		{Name: "emit", Category: "agent", Summary: "Emit a live agent bridge event handler payload.", Usage: "go run ./tools/gwc emit -hub http://127.0.0.1:8090 -token $GWC_AGENT_TOKEN -lease-holder agent-a -payload '{\"ref\":\"...\",\"event\":\"click\"}' -json", Examples: []string{"go run ./tools/gwc emit -hub http://127.0.0.1:8090 -token dev -lease-holder agent-a -payload '{\"ref\":\"submit\",\"event\":\"click\"}' -json"}, Flags: liveBridgeHelpFlags(true), JSON: true, ReadOnly: false, Mutating: true, Experimental: true},
		{Name: "publish", Category: "agent", Summary: "Publish a live agent bridge topic event.", Usage: "go run ./tools/gwc publish -hub http://127.0.0.1:8090 -token $GWC_AGENT_TOKEN -lease-holder agent-a -payload '{\"topic\":\"saved\"}' -json", Examples: []string{"go run ./tools/gwc publish -hub http://127.0.0.1:8090 -token dev -lease-holder agent-a -payload '{\"topic\":\"saved\"}' -json"}, Flags: liveBridgeHelpFlags(true), JSON: true, ReadOnly: false, Mutating: true, Experimental: true},
		{Name: "navigate", Category: "agent", Summary: "Navigate a live agent bridge router session.", Usage: "go run ./tools/gwc navigate -hub http://127.0.0.1:8090 -token $GWC_AGENT_TOKEN -lease-holder agent-a -payload '{\"href\":\"/settings\"}' -json", Examples: []string{"go run ./tools/gwc navigate -hub http://127.0.0.1:8090 -token dev -lease-holder agent-a -payload '{\"href\":\"/settings\"}' -json"}, Flags: liveBridgeHelpFlags(true), JSON: true, ReadOnly: false, Mutating: true, Experimental: true},
		{Name: "describe", Category: "agent", Summary: "Describe a live agent session's control surface: atoms (with JSON schemas), event topics (with subscriber counts), and available commands.", Usage: "go run ./tools/gwc describe -hub http://127.0.0.1:8090 -token $GWC_AGENT_TOKEN -json", Examples: []string{"go run ./tools/gwc describe -hub http://127.0.0.1:8090 -token dev -json"}, Flags: liveBridgeHelpFlags(false), JSON: true, ReadOnly: true, Experimental: true},
		{Name: "wait-for", Category: "agent", Summary: "Block until a live agent session reaches a condition (scheduler idle, an atom equals a value, or a query yields N matches) or times out.", Usage: "go run ./tools/gwc wait-for -hub http://127.0.0.1:8090 -token $GWC_AGENT_TOKEN -payload '{\"timeoutMs\":2000,\"atom\":{\"id\":\"theme\",\"equals\":\"dark\"}}' -json", Examples: []string{"go run ./tools/gwc wait-for -hub http://127.0.0.1:8090 -token dev -payload '{\"timeoutMs\":2000,\"idle\":true}' -json"}, Flags: liveBridgeHelpFlags(false), JSON: true, ReadOnly: true, Experimental: true},
		{Name: "audit", Category: "agent", Summary: "Read the live agent bridge mutation audit trail (every applied set-atom/delete-atom/publish and undo).", Usage: "go run ./tools/gwc audit -hub http://127.0.0.1:8090 -token $GWC_AGENT_TOKEN -payload '{\"max\":50}' -json", Examples: []string{"go run ./tools/gwc audit -hub http://127.0.0.1:8090 -token dev -json"}, Flags: liveBridgeHelpFlags(false), JSON: true, ReadOnly: true, Experimental: true},
		{Name: "undo", Category: "agent", Summary: "Reverse the most recent reversible live agent bridge mutation (set-atom or delete-atom), restoring the prior value.", Usage: "go run ./tools/gwc undo -hub http://127.0.0.1:8090 -token $GWC_AGENT_TOKEN -json", Examples: []string{"go run ./tools/gwc undo -hub http://127.0.0.1:8090 -token dev -json"}, Flags: liveBridgeHelpFlags(false), JSON: true, ReadOnly: false, Mutating: true, Experimental: true},
		{Name: "replay", Category: "agent", Summary: "Record and deterministically replay a window of runtime scheduling updates to reproduce a bug (action: start/stop/status/replay).", Usage: "go run ./tools/gwc replay -hub http://127.0.0.1:8090 -token $GWC_AGENT_TOKEN -payload '{\"action\":\"start\"}' -json", Examples: []string{"go run ./tools/gwc replay -hub http://127.0.0.1:8090 -token dev -payload '{\"action\":\"stop\"}' -json"}, Flags: liveBridgeHelpFlags(false), JSON: true, ReadOnly: false, Mutating: true, Experimental: true},
		{Name: "render-tree", Category: "agent", Summary: "Inject a structured element tree (JSON: tag/attrs/text/children) into a selector, built live with the real builders in an isolated runtime (no recompile; on* handlers stripped; reversible via undo).", Usage: "go run ./tools/gwc render-tree -hub http://127.0.0.1:8090 -token $GWC_AGENT_TOKEN -payload '{\"selector\":\"#slot\",\"tree\":{\"tag\":\"div\",\"text\":\"hi\"}}' -json", Examples: []string{"go run ./tools/gwc render-tree -hub http://127.0.0.1:8090 -token dev -payload '{\"selector\":\"#slot\",\"tree\":{\"tag\":\"h1\",\"text\":\"Hello\"}}' -json"}, Flags: liveBridgeHelpFlags(false), JSON: true, ReadOnly: false, Mutating: true, Experimental: true},
		{Name: "wasm", Category: "verify", Summary: "Run wasm-focused build experiment helpers such as wasm measure.", Usage: "go run ./tools/gwc wasm measure [flags]", Examples: []string{"go run ./tools/gwc wasm measure -json"}, JSON: true, ReadOnly: true},
	}
	sort.Slice(parseCommands, func(parseI int, parseJ int) bool {
		return parseCommands[parseI].Name < parseCommands[parseJ].Name
	})
	return parseCommands
}

func liveBridgeHelpFlags(parseMutating bool) []gwcHelpFlag {
	parseFlags := []gwcHelpFlag{
		{Name: "hub", Type: "url", Description: "Agent hub base URL"},
		{Name: "token", Type: "string", Description: "Agent hub token"},
		{Name: "session", Type: "id", Description: "Optional live session id; defaults to the newest active session"},
		{Name: "payload", Type: "json", Default: "{}", Description: "Bridge command payload JSON"},
		{Name: "timeout", Type: "duration", Default: "5s", Description: "Bridge command timeout"},
		{Name: "max", Type: "int", Default: "0", Description: "Maximum logs or recording records to return"},
		{Name: "json", Type: "bool", Default: "false", Description: "Emit JSON envelope"},
	}
	if parseMutating {
		parseFlags = append(parseFlags,
			gwcHelpFlag{Name: "lease-holder", Type: "string", Description: "Write lease holder for mutating bridge commands"},
			gwcHelpFlag{Name: "holder", Type: "string", Description: "Lease holder for the lease command"},
			gwcHelpFlag{Name: "action", Type: "string", Default: "acquire", Description: "Lease action: acquire, release, or steal"},
			gwcHelpFlag{Name: "steal", Type: "bool", Default: "false", Description: "Allow explicit write lease stealing"},
			gwcHelpFlag{Name: "clear", Type: "bool", Default: "false", Description: "Clear the session recording after reading it"},
		)
	}
	return parseFlags
}

// stringSliceContains reports whether a slice contains a case-insensitive string.
func stringSliceContains(parseValues []string, parseTarget string) bool {
	for _, parseValue := range parseValues {
		if strings.EqualFold(strings.TrimSpace(parseValue), parseTarget) {
			return true
		}
	}
	return false
}

// suggestGwcHelpCommand returns the nearest command suggestion using prefix and edit distance.
func suggestGwcHelpCommand(parseCommand string, parseCommands []gwcHelpCommand) string {
	for _, parseCandidate := range parseCommands {
		if strings.HasPrefix(parseCandidate.Name, parseCommand) || strings.HasPrefix(parseCommand, parseCandidate.Name) {
			return parseCandidate.Name
		}
	}
	parseBestName := ""
	parseBestDistance := 3
	for _, parseCandidate := range parseCommands {
		parseDistance := levenshteinDistance(parseCommand, parseCandidate.Name)
		if parseDistance < parseBestDistance {
			parseBestDistance = parseDistance
			parseBestName = parseCandidate.Name
		}
	}
	return parseBestName
}

// levenshteinDistance computes a small edit-distance score for command suggestions.
func levenshteinDistance(parseA string, parseB string) int {
	parseARunes := []rune(parseA)
	parseBRunes := []rune(parseB)
	parsePrevious := make([]int, len(parseBRunes)+1)
	for parseIndex := range parsePrevious {
		parsePrevious[parseIndex] = parseIndex
	}
	for parseI, parseRuneA := range parseARunes {
		parseCurrent := make([]int, len(parseBRunes)+1)
		parseCurrent[0] = parseI + 1
		for parseJ, parseRuneB := range parseBRunes {
			parseCost := 0
			if parseRuneA != parseRuneB {
				parseCost = 1
			}
			parseCurrent[parseJ+1] = min(parseCurrent[parseJ]+1, min(parsePrevious[parseJ+1]+1, parsePrevious[parseJ]+parseCost))
		}
		parsePrevious = parseCurrent
	}
	return parsePrevious[len(parseBRunes)]
}

// marshalGwcHelpCommands returns a JSON copy of the registry for tests and MCP.
func marshalGwcHelpCommands(parseCommands []gwcHelpCommand) ([]byte, error) {
	return json.Marshal(parseCommands)
}
