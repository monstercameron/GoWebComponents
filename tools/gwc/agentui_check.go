package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/monstercameron/GoWebComponents/v5/agentui"
)

// runAgentUICommand routes `gwc agentui check <file.json>` — validating an agent-emitted UI
// schema against the allow-list before it is ever rendered.
var runAgentUICommand = func(parseL launcher, parseArgs []string) error {
	return parseL.runAgentUI(parseArgs)
}

// runAgentUI validates an agent-UI JSON schema file against the default component allow-list,
// the structural safety gate (no non-allow-listed components, no disallowed props, within
// size limits) the FC3 runtime enforces — surfaced as a CLI/CI check so bad agent output is
// caught before it reaches a render.
func (parseL launcher) runAgentUI(parseArgs []string) error {
	parseAction := "check"
	if len(parseArgs) > 0 && !startsWithDash(parseArgs[0]) {
		parseAction = parseArgs[0]
		parseArgs = parseArgs[1:]
	}
	if parseAction == "catalog" {
		return printAgentUICatalog()
	}
	if parseAction != "check" {
		return fmt.Errorf("unknown agentui action %q (use check or catalog)", parseAction)
	}

	parseFlags := flag.NewFlagSet("agentui check", flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parseFile := parseFlags.String("file", "", "Path to an agent-UI JSON schema file to validate")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	parsePath := *parseFile
	if parsePath == "" && parseFlags.NArg() > 0 {
		parsePath = parseFlags.Arg(0)
	}
	if parsePath == "" {
		return errors.New("gwc agentui check requires a JSON schema file (path or -file)")
	}

	parseData, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		return fmt.Errorf("read agentui schema: %w", parseErr)
	}
	parseNode, parseErr := agentui.Parse(parseData)
	if parseErr != nil {
		return fmt.Errorf("%s: %w", parsePath, parseErr)
	}
	if parseErr := agentui.DefaultRegistry().Validate(parseNode); parseErr != nil {
		fmt.Printf("GWC agentui: REJECTED %s — %v\n", parsePath, parseErr)
		return fmt.Errorf("agentui schema %s is not allow-list-valid: %w", parsePath, parseErr)
	}
	fmt.Printf("GWC agentui: OK %s — valid against the component allow-list\n", parsePath)
	return nil
}

// printAgentUICatalog emits the component allow-list as JSON — the up-front guidance an
// agent (or an MCP tool wrapping this command) reads to learn exactly which components and
// props it may emit before generating a tree.
func printAgentUICatalog() error {
	parseCatalog := agentui.DefaultRegistry().Catalog()
	parseOut, parseErr := json.MarshalIndent(parseCatalog, "", "  ")
	if parseErr != nil {
		return fmt.Errorf("encode agentui catalog: %w", parseErr)
	}
	fmt.Println(string(parseOut))
	return nil
}

// startsWithDash reports whether s begins with '-'.
func startsWithDash(parseS string) bool {
	return len(parseS) > 0 && parseS[0] == '-'
}
