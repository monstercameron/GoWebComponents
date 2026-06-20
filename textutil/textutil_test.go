package textutil

import "testing"

func TestHumanize(t *testing.T) {
	cases := map[string]string{
		"credit_card":  "Credit card",
		"creditCard":   "Credit card",
		"credit-card":  "Credit card",
		"USD":          "Usd",
		"":             "",
		"already nice": "Already nice",
	}
	for parseIn, parseWant := range cases {
		if parseGot := Humanize(parseIn); parseGot != parseWant {
			t.Errorf("Humanize(%q) = %q, want %q", parseIn, parseGot, parseWant)
		}
	}
}

func TestTitleCase(t *testing.T) {
	cases := map[string]string{
		"credit_card": "Credit Card",
		"fooBarBaz":   "Foo Bar Baz",
		"a-b-c":       "A B C",
	}
	for parseIn, parseWant := range cases {
		if parseGot := TitleCase(parseIn); parseGot != parseWant {
			t.Errorf("TitleCase(%q) = %q, want %q", parseIn, parseGot, parseWant)
		}
	}
}
