package flags

import "testing"

func TestBuildSetCopiesInputs(parseT *testing.T) {
	parseFlags := map[string]Flag{
		"new-nav": {Enabled: true, Value: "compact", Reason: "test"},
	}
	parseExperiments := map[string]Experiment{
		"checkout": {
			Enabled: true,
			Salt:    "v1",
			Variants: []Variant{
				{Name: "control", Weight: 1},
				{Name: "treatment", Weight: 1},
			},
		},
	}

	parseSet := BuildSet(parseFlags, parseExperiments)
	parseFlags["new-nav"] = Flag{}
	parseExperiments["checkout"].Variants[0].Name = "mutated"

	parseFlag := parseSet.GetFlag("new-nav", Flag{})
	if !parseFlag.Enabled || parseFlag.Value != "compact" {
		parseT.Fatalf("BuildSet did not preserve copied flag: %+v", parseFlag)
	}
	parseAssignment := parseSet.GetAssignment("checkout", "user-1")
	if parseAssignment.Variant != "control" && parseAssignment.Variant != "treatment" {
		parseT.Fatalf("BuildSet did not preserve copied variants: %+v", parseAssignment)
	}
}

func TestGetFlagFallsBackWhenMissing(parseT *testing.T) {
	parseSet := BuildSet(nil, nil)
	parseFlag := parseSet.GetFlag("missing", Flag{Enabled: true, Value: "fallback", Reason: "default"})

	if !parseFlag.Enabled || parseFlag.Value != "fallback" || parseFlag.Reason != "default" {
		parseT.Fatalf("unexpected fallback flag: %+v", parseFlag)
	}
	if !parseSet.GetEnabled("missing", true) {
		parseT.Fatal("expected enabled fallback")
	}
	if parseSet.GetValue("missing", "fallback") != "fallback" {
		parseT.Fatal("expected value fallback")
	}
}

func TestGetAssignmentIsDeterministic(parseT *testing.T) {
	parseSet := BuildSet(nil, map[string]Experiment{
		"pricing": {
			Enabled: true,
			Salt:    "2026-06",
			Variants: []Variant{
				{Name: "control", Value: "A", Weight: 50},
				{Name: "variant", Value: "B", Weight: 50},
			},
		},
	})

	parseFirst := parseSet.GetAssignment("pricing", "customer-123")
	parseSecond := parseSet.GetAssignment("pricing", "customer-123")

	if parseFirst != parseSecond {
		parseT.Fatalf("assignment changed between calls: first=%+v second=%+v", parseFirst, parseSecond)
	}
	if !parseFirst.Enabled || parseFirst.Reason != "assigned" {
		parseT.Fatalf("expected assigned experiment: %+v", parseFirst)
	}
	if parseFirst.Bucket < 0 || parseFirst.Bucket >= 100 {
		parseT.Fatalf("bucket out of range: %+v", parseFirst)
	}
}

func TestGetAssignmentReportsInactiveStates(parseT *testing.T) {
	parseSet := BuildSet(nil, map[string]Experiment{
		"disabled": {Enabled: false, Variants: []Variant{{Name: "a", Weight: 1}}},
		"empty":    {Enabled: true},
		"zero":     {Enabled: true, Variants: []Variant{{Name: "a", Weight: 0}}},
	})

	parseCases := map[string]string{
		"missing":  "missing",
		"disabled": "disabled",
		"empty":    "no_variants",
		"zero":     "no_variants",
	}
	for parseName, parseReason := range parseCases {
		parseAssignment := parseSet.GetAssignment(parseName, "subject")
		if parseAssignment.Reason != parseReason {
			parseT.Fatalf("%s: expected reason %q, got %+v", parseName, parseReason, parseAssignment)
		}
	}
}

func TestZeroHandlesAreSafe(parseT *testing.T) {
	if (FlagHandle{}).Enabled() {
		parseT.Fatal("zero flag handle should be disabled")
	}
	if (FlagHandle{}).Value("fallback") != "fallback" {
		parseT.Fatal("zero flag handle should return value fallback")
	}
	if (ExperimentHandle{}).Get() != (Assignment{}) {
		parseT.Fatal("zero experiment handle should return zero assignment")
	}
	if (Registry{}).Get().Flags != nil {
		parseT.Fatal("zero registry should return zero set")
	}
}
