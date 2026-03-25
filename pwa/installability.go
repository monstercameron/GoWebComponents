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

func (parseS InstallabilitySubscription) Cancel() {
	if parseS.cancel != nil {
		parseS.cancel()
	}
}

type InstallabilityManager struct {
	state     func() InstallabilityState
	prompt    func(context.Context) (InstallPromptResult, error)
	subscribe func(func(InstallabilityState)) (InstallabilitySubscription, error)
}

func (parseM InstallabilityManager) State() InstallabilityState {
	if parseM.state == nil {
		return InstallabilityState{}
	}
	return parseM.state()
}

func (parseM InstallabilityManager) Prompt(parseCtx context.Context) (InstallPromptResult, error) {
	if parseM.prompt == nil {
		return InstallPromptResult{}, installabilityUnavailable("InstallabilityManager.Prompt", "")
	}
	return parseM.prompt(parseCtx)
}

func (parseM InstallabilityManager) Subscribe(parseHandler func(InstallabilityState)) (InstallabilitySubscription, error) {
	if parseM.subscribe == nil {
		return InstallabilitySubscription{}, installabilityUnavailable("InstallabilityManager.Subscribe", "")
	}
	return parseM.subscribe(parseHandler)
}
