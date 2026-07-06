package extract

import "testing"

// TestExtractsNamespaceReceiverTCalls pins that the extractor recognizes the
// Namespace.T(key) call shape produced by rt.NS("ns").T("key"). Previously only
// the Runtime.T(namespace, key) form was handled: a one-arg .T call had its key
// misread as the namespace (and no key), so Namespace-bound keys silently never
// entered the completeness check — a false "complete" report against real code.
func TestExtractsNamespaceReceiverTCalls(parseT *testing.T) {
	parseSource := `package sample

func demo(rt Runtime) {
	// Namespace.T form: namespace bound in the receiver, key is arg0.
	_ = rt.NS("home").T("greeting")
	// Runtime.T form: namespace + key as the first two args (must still work).
	_ = rt.T("account", "welcome")
	// Non-literal key on a Namespace receiver must be surfaced as Dynamic, not dropped.
	key := "dynamic"
	_ = rt.NS("cart").T(key)
}

type Runtime struct{}

func (Runtime) NS(string) Namespace { return Namespace{} }
func (Runtime) T(string, string) string { return "" }

type Namespace struct{}

func (Namespace) T(string) string { return "" }
`

	parseResult, parseErr := ExtractFromSource("sample.go", parseSource)
	if parseErr != nil {
		parseT.Fatalf("extract failed: %v", parseErr)
	}

	parseWant := map[string]string{
		"home\x00greeting":   "",
		"account\x00welcome": "",
	}
	parseGot := map[string]bool{}
	for _, parseMsg := range parseResult.Messages {
		parseGot[parseMsg.Namespace+"\x00"+parseMsg.Key] = true
	}
	for parseKey := range parseWant {
		if !parseGot[parseKey] {
			parseT.Fatalf("expected extracted message %q, got messages %#v", parseKey, parseResult.Messages)
		}
	}
	// The Namespace.T key must NOT have been misread as a namespace.
	for _, parseMsg := range parseResult.Messages {
		if parseMsg.Namespace == "greeting" {
			parseT.Fatalf("key %q was misread as a namespace", parseMsg.Namespace)
		}
	}
	// The non-literal Namespace.T key is surfaced as Dynamic (key is non-literal),
	// not silently dropped.
	parseFoundDynamic := false
	for _, parseDyn := range parseResult.Dynamic {
		if parseDyn.Reason == "key is non-literal" {
			parseFoundDynamic = true
		}
	}
	if !parseFoundDynamic {
		parseT.Fatalf("expected a non-literal-key Dynamic usage, got %#v", parseResult.Dynamic)
	}
}
