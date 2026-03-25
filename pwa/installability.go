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

func (parseSubscription InstallabilitySubscription) Cancel() {
	if parseSubscription.cancel != nil {
		parseSubscription.cancel()
	}
}

type InstallabilityManager struct {
	state     func() InstallabilityState
	prompt    func(context.Context) (InstallPromptResult, error)
	subscribe func(func(InstallabilityState)) (InstallabilitySubscription, error)
}

func (parseManager InstallabilityManager) State() InstallabilityState {
	if parseManager.state == nil {
		return InstallabilityState{}
	}
	return parseManager.state()
}

func (parseManager InstallabilityManager) Prompt(parseInstallCtx context.Context) (InstallPromptResult, error) {
	if parseManager.prompt == nil {
		return InstallPromptResult{}, installabilityUnavailable("InstallabilityManager.Prompt", "")
	}
	return parseManager.prompt(parseInstallCtx)
}

func (parseManager InstallabilityManager) Subscribe(parseInstallHandler func(InstallabilityState)) (InstallabilitySubscription, error) {
	if parseManager.subscribe == nil {
		return InstallabilitySubscription{}, installabilityUnavailable("InstallabilityManager.Subscribe", "")
	}
	return parseManager.subscribe(parseInstallHandler)
}
