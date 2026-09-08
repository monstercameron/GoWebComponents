package contracts

// WorkState exposes cooperative native work counters for the desktop example.
type WorkState struct {
	Active    int `json:"active"`
	Cancelled int `json:"cancelled"`
}
