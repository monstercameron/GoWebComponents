//go:build js && wasm

package app

import "testing"

func TestShouldApplySelectedModelBootstrap(parseT *testing.T) {
	parseT.Parallel()

	parseBase := appState{
		GRPCReady:        true,
		Authenticated:    true,
		ActiveConvID:     0,
		Messages:         []message{},
		ModelOptions:     []modelOption{{ID: "gpt-5.4-mini"}},
		DefaultModel:     "gpt-5.4-mini",
		SelectedModel:    "gpt-5.4-mini",
		SessionEmail:     "demo@example.com",
		ConversationList: []convSummary{},
	}

	parseTests := []struct {
		name              string
		bootstrapComplete bool
		cacheReady        bool
		mutate            func(appState) appState
		want              bool
	}{
		{
			name:              "applies during empty draft bootstrap",
			bootstrapComplete: false,
			cacheReady:        true,
			mutate:            func(parseState2 appState) appState { return parseState2 },
			want:              true,
		},
		{
			name:              "does not apply after bootstrap completes",
			bootstrapComplete: true,
			cacheReady:        true,
			mutate:            func(parseState3 appState) appState { return parseState3 },
			want:              false,
		},
		{
			name:              "does not apply while conversation is active",
			bootstrapComplete: false,
			cacheReady:        true,
			mutate: func(parseState4 appState) appState {
				parseState4.ActiveConvID = 42
				return parseState4
			},
			want: false,
		},
		{
			name:              "does not apply when draft has messages",
			bootstrapComplete: false,
			cacheReady:        true,
			mutate: func(parseState5 appState) appState {
				parseState5.Messages = []message{{Role: roleUser, Content: "hello"}}
				return parseState5
			},
			want: false,
		},
		{
			name:              "does not apply when cache is not ready",
			bootstrapComplete: false,
			cacheReady:        false,
			mutate:            func(parseState6 appState) appState { return parseState6 },
			want:              false,
		},
	}

	for _, parseTt := range parseTests {
		parseTt2 := parseTt
		parseT.Run(parseTt2.name, func(parseT2 *testing.T) {
			parseT2.Parallel()
			parseState := parseTt2.mutate(parseBase)
			parseGot := shouldApplySelectedModelBootstrap(parseTt2.bootstrapComplete, parseState, parseTt2.cacheReady)
			if parseGot != parseTt2.want {
				parseT2.Fatalf("shouldApplySelectedModelBootstrap() = %v, want %v", parseGot, parseTt2.want)
			}
		})
	}
}

func TestShouldApplySelectedModelRecovery(parseT *testing.T) {
	parseT.Parallel()

	parseBase := appState{
		GRPCReady:        true,
		Authenticated:    true,
		ActiveConvID:     0,
		Messages:         []message{},
		ModelOptions:     []modelOption{{ID: "gpt-5.4-mini"}},
		DefaultModel:     "gpt-5.4-mini",
		SelectedModel:    "gpt-5.4-mini",
		SessionEmail:     "demo@example.com",
		ConversationList: []convSummary{},
	}

	parseTests := []struct {
		name             string
		recoveryComplete bool
		cacheReady       bool
		mutate           func(appState) appState
		want             bool
	}{
		{
			name:             "applies when catalog is loaded and draft is empty",
			recoveryComplete: false,
			cacheReady:       true,
			mutate:           func(parseState2 appState) appState { return parseState2 },
			want:             true,
		},
		{
			name:             "does not apply after recovery completes",
			recoveryComplete: true,
			cacheReady:       true,
			mutate:           func(parseState3 appState) appState { return parseState3 },
			want:             false,
		},
		{
			name:             "does not apply without model options",
			recoveryComplete: false,
			cacheReady:       true,
			mutate: func(parseState4 appState) appState {
				parseState4.ParseModelOptions = nil
				return parseState4
			},
			want: false,
		},
		{
			name:             "does not apply while messages exist",
			recoveryComplete: false,
			cacheReady:       true,
			mutate: func(parseState5 appState) appState {
				parseState5.Messages = []message{{Role: roleUser, Content: "hello"}}
				return parseState5
			},
			want: false,
		},
	}

	for _, parseTt := range parseTests {
		parseTt2 := parseTt
		parseT.Run(parseTt2.name, func(parseT2 *testing.T) {
			parseT2.Parallel()
			parseState := parseTt2.mutate(parseBase)
			parseGot := shouldApplySelectedModelRecovery(parseTt2.recoveryComplete, parseState, parseTt2.cacheReady)
			if parseGot != parseTt2.want {
				parseT2.Fatalf("shouldApplySelectedModelRecovery() = %v, want %v", parseGot, parseTt2.want)
			}
		})
	}
}

func TestShouldApplyThinkingPreferencesBootstrap(parseT *testing.T) {
	parseT.Parallel()

	parseBase := appState{
		GRPCReady:     true,
		Authenticated: true,
	}

	parseTests := []struct {
		name              string
		bootstrapComplete bool
		enabledReady      bool
		effortReady       bool
		mutate            func(appState) appState
		want              bool
	}{
		{
			name:              "applies once both caches are ready",
			bootstrapComplete: false,
			enabledReady:      true,
			effortReady:       true,
			mutate:            func(parseState2 appState) appState { return parseState2 },
			want:              true,
		},
		{
			name:              "does not apply once bootstrapped",
			bootstrapComplete: true,
			enabledReady:      true,
			effortReady:       true,
			mutate:            func(parseState3 appState) appState { return parseState3 },
			want:              false,
		},
		{
			name:              "does not apply until both caches are ready",
			bootstrapComplete: false,
			enabledReady:      true,
			effortReady:       false,
			mutate:            func(parseState4 appState) appState { return parseState4 },
			want:              false,
		},
		{
			name:              "does not apply when unauthenticated",
			bootstrapComplete: false,
			enabledReady:      true,
			effortReady:       true,
			mutate: func(parseState5 appState) appState {
				parseState5.Authenticated = false
				return parseState5
			},
			want: false,
		},
	}

	for _, parseTt := range parseTests {
		parseTt2 := parseTt
		parseT.Run(parseTt2.name, func(parseT2 *testing.T) {
			parseT2.Parallel()
			parseState := parseTt2.mutate(parseBase)
			parseGot := shouldApplyThinkingPreferencesBootstrap(parseTt2.bootstrapComplete, parseState, parseTt2.enabledReady, parseTt2.effortReady)
			if parseGot != parseTt2.want {
				parseT2.Fatalf("shouldApplyThinkingPreferencesBootstrap() = %v, want %v", parseGot, parseTt2.want)
			}
		})
	}
}
