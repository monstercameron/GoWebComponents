package contracts

// CounterState is the portable counter service response.
type CounterState struct {
	Value int `json:"value"`
}

// Progress is a native operation progress event.
type Progress struct {
	Percent int    `json:"percent"`
	Stage   string `json:"stage"`
}
