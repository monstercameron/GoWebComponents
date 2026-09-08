package contracts

// APIResult records a native operation separately from a human's visual verdict.
type APIResult struct {
	ID       string   `json:"id"`
	Outcome  string   `json:"outcome"`
	Detail   string   `json:"detail"`
	WindowID string   `json:"windowID"`
	At       string   `json:"at"`
	Paths    []string `json:"paths"`
}

// APIReport is a bounded, explicitly exported session report, never telemetry.
type APIReport struct {
	Platform     string      `json:"platform"`
	WailsVersion string      `json:"wailsVersion"`
	FixtureDir   string      `json:"fixtureDir"`
	Results      []APIResult `json:"results"`
}
