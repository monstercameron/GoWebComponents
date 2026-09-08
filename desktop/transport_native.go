//go:build !js || !wasm

package desktop

// Connect returns an unavailable client on native/SSR builds without loading Wails.
func Connect() (Client, error) {
	parseClient := Client{}
	_, parseErr := parseClient.GetCapabilities()
	return parseClient, parseErr
}
