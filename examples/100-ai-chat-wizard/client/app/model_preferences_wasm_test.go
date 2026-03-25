//go:build js && wasm

package app

import "testing"

func TestShouldApplySelectedModelBootstrap(t *testing.T) {
	t.Parallel()

	base := appState{
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

	tests := []struct {
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
			mutate:            func(state appState) appState { return state },
			want:              true,
		},
		{
			name:              "does not apply after bootstrap completes",
			bootstrapComplete: true,
			cacheReady:        true,
			mutate:            func(state appState) appState { return state },
			want:              false,
		},
		{
			name:              "does not apply while conversation is active",
			bootstrapComplete: false,
			cacheReady:        true,
			mutate: func(state appState) appState {
				state.ActiveConvID = 42
				return state
			},
			want: false,
		},
		{
			name:              "does not apply when draft has messages",
			bootstrapComplete: false,
			cacheReady:        true,
			mutate: func(state appState) appState {
				state.Messages = []message{{Role: roleUser, Content: "hello"}}
				return state
			},
			want: false,
		},
		{
			name:              "does not apply when cache is not ready",
			bootstrapComplete: false,
			cacheReady:        false,
			mutate:            func(state appState) appState { return state },
			want:              false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			state := tt.mutate(base)
			got := shouldApplySelectedModelBootstrap(tt.bootstrapComplete, state, tt.cacheReady)
			if got != tt.want {
				t.Fatalf("shouldApplySelectedModelBootstrap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldApplySelectedModelRecovery(t *testing.T) {
	t.Parallel()

	base := appState{
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

	tests := []struct {
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
			mutate:           func(state appState) appState { return state },
			want:             true,
		},
		{
			name:             "does not apply after recovery completes",
			recoveryComplete: true,
			cacheReady:       true,
			mutate:           func(state appState) appState { return state },
			want:             false,
		},
		{
			name:             "does not apply without model options",
			recoveryComplete: false,
			cacheReady:       true,
			mutate: func(state appState) appState {
				state.ModelOptions = nil
				return state
			},
			want: false,
		},
		{
			name:             "does not apply while messages exist",
			recoveryComplete: false,
			cacheReady:       true,
			mutate: func(state appState) appState {
				state.Messages = []message{{Role: roleUser, Content: "hello"}}
				return state
			},
			want: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			state := tt.mutate(base)
			got := shouldApplySelectedModelRecovery(tt.recoveryComplete, state, tt.cacheReady)
			if got != tt.want {
				t.Fatalf("shouldApplySelectedModelRecovery() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldApplyThinkingPreferencesBootstrap(t *testing.T) {
	t.Parallel()

	base := appState{
		GRPCReady:     true,
		Authenticated: true,
	}

	tests := []struct {
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
			mutate:            func(state appState) appState { return state },
			want:              true,
		},
		{
			name:              "does not apply once bootstrapped",
			bootstrapComplete: true,
			enabledReady:      true,
			effortReady:       true,
			mutate:            func(state appState) appState { return state },
			want:              false,
		},
		{
			name:              "does not apply until both caches are ready",
			bootstrapComplete: false,
			enabledReady:      true,
			effortReady:       false,
			mutate:            func(state appState) appState { return state },
			want:              false,
		},
		{
			name:              "does not apply when unauthenticated",
			bootstrapComplete: false,
			enabledReady:      true,
			effortReady:       true,
			mutate: func(state appState) appState {
				state.Authenticated = false
				return state
			},
			want: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			state := tt.mutate(base)
			got := shouldApplyThinkingPreferencesBootstrap(tt.bootstrapComplete, state, tt.enabledReady, tt.effortReady)
			if got != tt.want {
				t.Fatalf("shouldApplyThinkingPreferencesBootstrap() = %v, want %v", got, tt.want)
			}
		})
	}
}
