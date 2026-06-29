package css

import "sync"

// DefineUtility registers a named, reusable bundle of rules — the plugin path for
// user-defined utilities that compose with the built-ins. It returns the bundle so
// it can be used inline immediately, and stores it under name for later lookup via
// Utility.
//
//	card := css.DefineUtility("card",
//	    css.Padding(css.Px(16)), css.Rounded(css.Px(8)), css.Bg(css.White))
//	Div(css.Class(card...), …)
func DefineUtility(parseName string, rules ...Rule) []Rule {
	utilityMu.Lock()
	defer utilityMu.Unlock()
	bundle := append([]Rule{}, rules...)
	utilityRegistry[parseName] = bundle
	return append([]Rule{}, bundle...)
}

// Utility looks up a previously defined utility bundle; ok reports existence.
func Utility(parseName string) (rules []Rule, ok bool) {
	utilityMu.RLock()
	defer utilityMu.RUnlock()
	bundle, found := utilityRegistry[parseName]
	if !found {
		return nil, false
	}
	return append([]Rule{}, bundle...), true
}

var (
	utilityMu       sync.RWMutex
	utilityRegistry = map[string][]Rule{}
)
