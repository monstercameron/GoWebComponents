package main

type launcherPluginInvocationRequest struct {
	SchemaVersion string                         `json:"schemaVersion"`
	PluginName    string                         `json:"pluginName"`
	Capability    string                         `json:"capability"`
	Command       string                         `json:"command"`
	CommandArgs   []string                       `json:"commandArgs,omitempty"`
	RepoRoot      string                         `json:"repoRoot,omitempty"`
	Sources       launcherEnterpriseConfigSources `json:"sources"`
}

type launcherPluginDiagnostic struct {
	Code     string `json:"code,omitempty"`
	Severity string `json:"severity,omitempty"`
	Summary  string `json:"summary,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

type launcherPluginTestLane struct {
	Name    string   `json:"name"`
	Command []string `json:"command"`
}

type launcherPluginCheck struct {
	Name       string `json:"name"`
	Passed     bool   `json:"passed"`
	Summary    string `json:"summary,omitempty"`
	Remediation string `json:"remediation,omitempty"`
}

type launcherPluginScaffoldSection struct {
	Title    string   `json:"title"`
	Features []string `json:"features,omitempty"`
	Summary  string   `json:"summary,omitempty"`
}

type launcherPluginInvocationResponse struct {
	OK                *bool                         `json:"ok,omitempty"`
	Summary           string                        `json:"summary,omitempty"`
	Diagnostics       []launcherPluginDiagnostic    `json:"diagnostics,omitempty"`
	ContributedLanes  []launcherPluginTestLane      `json:"contributedTestLanes,omitempty"`
	VerifyChecks      []launcherPluginCheck         `json:"verifyChecks,omitempty"`
	ReleaseValidators []launcherPluginCheck         `json:"releaseValidators,omitempty"`
	ScaffoldSections  []launcherPluginScaffoldSection `json:"scaffoldSections,omitempty"`
}
