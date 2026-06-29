package agentbridge

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// AgentCommandHandler executes one bridge command. It returns the ack payload
// on success, or a structured error (with a stable wire code) on failure.
// Handlers own their scheduler-locking discipline: anything touching the
// fiber tree or hook state must take the runtime's scheduler lock the same
// way Inspect does.
type AgentCommandHandler func(parsePayload json.RawMessage) (json.RawMessage, *EnvelopeError)

// commandRegistryMu guards the process-wide bridge command registry.
var commandRegistryMu sync.RWMutex

// agentModeMu guards the agent-mode flag.
var agentModeMu sync.RWMutex

// isAgentModeActive records whether the bridge client enabled agent mode for
// this app (dev build tag + ?gwc-dev=agent + hub token all present).
var isAgentModeActive bool

// SetAgentModeActive records whether agent mode is enabled. The bridge client
// sets it after its gating checks pass; mutating command handlers refuse with
// the forbidden wire code while it is false.
func SetAgentModeActive(parseActive bool) {
	agentModeMu.Lock()
	defer agentModeMu.Unlock()
	isAgentModeActive = parseActive
}

// IsAgentModeActive reports whether the bridge client enabled agent mode.
func IsAgentModeActive() bool {
	agentModeMu.RLock()
	defer agentModeMu.RUnlock()
	return isAgentModeActive
}

// commandRegistry maps command names (e.g. "bridge.snapshot") to handlers.
var commandRegistry = map[string]AgentCommandHandler{}

// RegisterAgentCommand registers a bridge command handler by wire name.
// Later registrations replace earlier ones (last writer wins) so tests can
// stub commands; production registration happens once at bridge install.
//
// A blank name or nil handler is silently ignored (no registration, no panic), so
// guard against accidentally passing either if you rely on the command being present.
func RegisterAgentCommand(parseName string, parseHandler AgentCommandHandler) {
	parseTrimmed := strings.TrimSpace(parseName)
	if parseTrimmed == "" || parseHandler == nil {
		return
	}
	commandRegistryMu.Lock()
	defer commandRegistryMu.Unlock()
	commandRegistry[parseTrimmed] = parseHandler
}

// ListAgentCommands returns the sorted names of every registered command.
func ListAgentCommands() []string {
	commandRegistryMu.RLock()
	defer commandRegistryMu.RUnlock()
	parseNames := make([]string, 0, len(commandRegistry))
	for parseName := range commandRegistry {
		parseNames = append(parseNames, parseName)
	}
	sort.Strings(parseNames)
	return parseNames
}

// ExecuteAgentCommand runs a registered command by name. Unknown names fail
// with the stable unknown-command wire code so agents can branch on it. A
// handler that panics is contained into a bad-payload error so a single bad
// command can never kill the client's dispatch goroutine (which would strand
// the app in agent mode with no listener).
func ExecuteAgentCommand(parseName string, parsePayload json.RawMessage) (parseResult json.RawMessage, parseErr *EnvelopeError) {
	commandRegistryMu.RLock()
	parseHandler, parseOk := commandRegistry[strings.TrimSpace(parseName)]
	commandRegistryMu.RUnlock()
	if !parseOk {
		return nil, &EnvelopeError{Code: ErrorCodeUnknownCommand, Message: "no handler registered for command " + parseName}
	}
	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			parseResult = nil
			parseErr = &EnvelopeError{
				Code:    ErrorCodeBadPayload,
				Message: fmt.Sprintf("command %q panicked: %v", parseName, parseRecovered),
			}
		}
	}()
	return parseHandler(parsePayload)
}
