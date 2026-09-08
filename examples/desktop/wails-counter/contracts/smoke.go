package contracts

// SmokeReport records checks completed in the actual native WebView.
type SmokeReport struct {
	OK     bool     `json:"ok"`
	Checks []string `json:"checks"`
	Error  string   `json:"error"`
}

// ObserverState reports rendered evidence from the second native smoke window.
type ObserverState struct {
	Ready bool   `json:"ready"`
	Value int    `json:"value"`
	Local int    `json:"local"`
	Error string `json:"error"`
}
