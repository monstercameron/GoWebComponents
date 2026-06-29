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

func TestGetValueFallsBackWhenPresentFlagValueEmpty(parseT *testing.T) {
	parseSet := BuildSet(map[string]Flag{
		"theme": {Enabled: true, Value: "", Reason: "remote-empty"},
	}, nil)

	if parseGot := parseSet.GetValue("theme", "system"); parseGot != "system" {
		parseT.Fatalf("expected empty present flag value to use fallback, got %q", parseGot)
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

func TestGetBucketStability(parseT *testing.T) {
	parseBucket := getBucket("pricing", "v1", "customer-123", 100)
	if parseBucket != 59 {
		parseT.Fatalf("getBucket changed cohort assignment: got %d, want 59", parseBucket)
	}
	if parseBucket2 := getBucket("pricing", "v1", "customer-123", 100); parseBucket2 != parseBucket {
		parseT.Fatalf("getBucket changed between calls: first=%d second=%d", parseBucket, parseBucket2)
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

func TestRegistrySetUpdateAndHandlesUseImmutableSnapshots(parseT *testing.T) {
	parseCurrent := BuildSet(map[string]Flag{
		"theme": {Enabled: true, Value: "dark", Reason: "initial"},
	}, map[string]Experiment{
		"checkout": {
			Enabled:  true,
			Salt:     "v1",
			Variants: []Variant{{Name: "control", Value: "A", Weight: 1}},
		},
	})
	parseRegistry := Registry{
		get: func() Set { return parseCurrent },
		set: func(parseNext Set) { parseCurrent = parseNext },
	}

	parseNext := BuildSet(map[string]Flag{
		"theme": {Enabled: false, Value: "light", Reason: "remote"},
	}, map[string]Experiment{
		"checkout": {
			Enabled:  true,
			Salt:     "v2",
			Variants: []Variant{{Name: "variant", Value: "B", Weight: 1}},
		},
	})
	parseRegistry.Set(parseNext)
	parseNext.Flags["theme"] = Flag{}
	parseNext.Experiments["checkout"] = Experiment{}
	if parseGot := parseRegistry.Get().GetValue("theme", "fallback"); parseGot != "light" {
		parseT.Fatalf("Registry.Set should copy input set, got value %q", parseGot)
	}
	if parseAssignment := parseRegistry.Get().GetAssignment("checkout", "subject"); parseAssignment.Variant != "variant" {
		parseT.Fatalf("Registry.Set should copy experiment variants, got %+v", parseAssignment)
	}

	parseRegistry.Update(func(parsePrevious Set) Set {
		if parsePrevious.GetValue("theme", "") != "light" {
			parseT.Fatalf("Registry.Update received wrong previous set: %+v", parsePrevious)
		}
		return BuildSet(map[string]Flag{"theme": {Enabled: true, Value: "system", Reason: "updated"}}, nil)
	})
	if parseGot := parseRegistry.Get().GetValue("theme", "fallback"); parseGot != "system" {
		parseT.Fatalf("Registry.Update value = %q", parseGot)
	}

	parseNilUpdateRegistry := Registry{set: parseRegistry.set}
	parseNilUpdateRegistry.Update(nil)
	if parseGot := parseRegistry.Get().GetValue("theme", "fallback"); parseGot != "system" {
		parseT.Fatalf("nil update should be a no-op, got %q", parseGot)
	}

	parseFlagHandle := FlagHandle{get: func() Flag { return parseRegistry.Get().GetFlag("theme", Flag{}) }}
	if !parseFlagHandle.Enabled() || parseFlagHandle.Value("fallback") != "system" {
		parseT.Fatalf("FlagHandle read wrong value: %+v", parseFlagHandle.Get())
	}
	parseBlankValueHandle := FlagHandle{get: func() Flag { return Flag{Enabled: true} }}
	if parseGot := parseBlankValueHandle.Value("fallback"); parseGot != "fallback" {
		parseT.Fatalf("FlagHandle.Value blank fallback = %q", parseGot)
	}

	parseExperimentHandle := ExperimentHandle{get: func() Assignment {
		return Assignment{Experiment: "checkout", Variant: "variant", Enabled: true, Reason: "assigned"}
	}}
	if parseGot := parseExperimentHandle.Get(); parseGot.Variant != "variant" || !parseGot.Enabled {
		parseT.Fatalf("ExperimentHandle.Get() = %+v", parseGot)
	}
}
