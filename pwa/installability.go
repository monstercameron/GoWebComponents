package pwa

import "context"

type InstallabilityOptions struct {
	Manifest *Manifest
}

type InstallPromptResult struct {
	Outcome  string
	Platform string
}

type InstallabilityState struct {
	ManifestValid   bool
	ManifestError   string
	PromptAvailable bool
	Installed       bool
	Reasons         []string
}

type InstallabilitySubscription struct {
	cancel func()
}

func (s InstallabilitySubscription) Cancel() {
	if s.cancel != nil {
		s.cancel()
	}
}

type InstallabilityManager struct {
	state     func() InstallabilityState
	prompt    func(context.Context) (InstallPromptResult, error)
	subscribe func(func(InstallabilityState)) (InstallabilitySubscription, error)
}

func (m InstallabilityManager) State() InstallabilityState {
	if m.state == nil {
		return InstallabilityState{}
	}
	return m.state()
}

func (m InstallabilityManager) Prompt(ctx context.Context) (InstallPromptResult, error) {
	if m.prompt == nil {
		return InstallPromptResult{}, installabilityUnavailable("InstallabilityManager.Prompt", "")
	}
	return m.prompt(ctx)
}

func (m InstallabilityManager) Subscribe(handler func(InstallabilityState)) (InstallabilitySubscription, error) {
	if m.subscribe == nil {
		return InstallabilitySubscription{}, installabilityUnavailable("InstallabilityManager.Subscribe", "")
	}
	return m.subscribe(handler)
}
