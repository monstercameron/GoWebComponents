package flags

import (
	"hash/fnv"
	"maps"

	"github.com/monstercameron/GoWebComponents/state"
)

const defaultRegistryAtomID = "gwc:flags:registry"

// Flag describes one evaluated browser-visible feature flag.
type Flag struct {
	Enabled bool
	Value   string
	Reason  string
}

// Variant describes one weighted experiment outcome.
type Variant struct {
	Name   string
	Value  string
	Weight int
}

// Experiment describes one deterministic weighted experiment.
type Experiment struct {
	Enabled  bool
	Salt     string
	Variants []Variant
}

// Assignment describes an evaluated experiment outcome.
type Assignment struct {
	Experiment string
	Variant    string
	Value      string
	Enabled    bool
	Bucket     int
	Reason     string
}

// Set stores browser-visible flags and experiments.
type Set struct {
	Flags       map[string]Flag
	Experiments map[string]Experiment
}

// Registry exposes the current shared flag set.
type Registry struct {
	get func() Set
	set func(Set)
}

// FlagHandle exposes one hook-selected flag.
type FlagHandle struct {
	get func() Flag
}

// ExperimentHandle exposes one hook-selected experiment assignment.
type ExperimentHandle struct {
	get func() Assignment
}

// BuildSet builds an immutable copy of flags and experiments.
func BuildSet(parseFlags map[string]Flag, parseExperiments map[string]Experiment) Set {
	parseResult := Set{
		Flags:       make(map[string]Flag, len(parseFlags)),
		Experiments: make(map[string]Experiment, len(parseExperiments)),
	}
	maps.Copy(parseResult.Flags, parseFlags)
	for parseName, parseExperiment := range parseExperiments {
		parseCopiedExperiment := parseExperiment
		if len(parseExperiment.Variants) > 0 {
			parseCopiedExperiment.Variants = append([]Variant(nil), parseExperiment.Variants...)
		}
		parseResult.Experiments[parseName] = parseCopiedExperiment
	}
	return parseResult
}

// GetFlag returns a named flag or the fallback when it is absent.
func (parseSet Set) GetFlag(parseName string, parseFallback Flag) Flag {
	if parseSet.Flags == nil {
		return parseFallback
	}
	parseFlag, parseOk := parseSet.Flags[parseName]
	if !parseOk {
		return parseFallback
	}
	return parseFlag
}

// GetEnabled returns whether a named flag is enabled.
func (parseSet Set) GetEnabled(parseName string, parseFallback bool) bool {
	return parseSet.GetFlag(parseName, Flag{Enabled: parseFallback, Reason: "fallback"}).Enabled
}

// GetValue returns a named flag value or the fallback when absent.
func (parseSet Set) GetValue(parseName string, parseFallback string) string {
	parseFlag := parseSet.GetFlag(parseName, Flag{Value: parseFallback, Reason: "fallback"})
	if parseFlag.Value == "" {
		return parseFallback
	}
	return parseFlag.Value
}

// GetAssignment returns a deterministic weighted experiment assignment.
func (parseSet Set) GetAssignment(parseName string, parseSubject string) Assignment {
	parseExperiment, parseOk := parseSet.Experiments[parseName]
	if !parseOk {
		return Assignment{Experiment: parseName, Reason: "missing"}
	}
	if !parseExperiment.Enabled {
		return Assignment{Experiment: parseName, Reason: "disabled"}
	}

	parseTotalWeight := 0
	for _, parseVariant := range parseExperiment.Variants {
		if parseVariant.Weight > 0 {
			parseTotalWeight += parseVariant.Weight
		}
	}
	if parseTotalWeight <= 0 {
		return Assignment{Experiment: parseName, Enabled: true, Reason: "no_variants"}
	}

	parseBucket := getBucket(parseName, parseExperiment.Salt, parseSubject, parseTotalWeight)
	parseCursor := 0
	for _, parseVariant := range parseExperiment.Variants {
		if parseVariant.Weight <= 0 {
			continue
		}
		parseCursor += parseVariant.Weight
		if parseBucket < parseCursor {
			return Assignment{
				Experiment: parseName,
				Variant:    parseVariant.Name,
				Value:      parseVariant.Value,
				Enabled:    true,
				Bucket:     parseBucket,
				Reason:     "assigned",
			}
		}
	}

	return Assignment{Experiment: parseName, Enabled: true, Bucket: parseBucket, Reason: "unassigned"}
}

// UseRegistry subscribes the current component to the shared flag registry.
func UseRegistry(parseInitial Set) Registry {
	parseAtom := state.UseAtom(defaultRegistryAtomID, BuildSet(parseInitial.Flags, parseInitial.Experiments))
	return Registry{get: parseAtom.Get, set: parseAtom.Set}
}

// Get returns the current registry set.
func (parseRegistry Registry) Get() Set {
	if parseRegistry.get == nil {
		return Set{}
	}
	return parseRegistry.get()
}

// Set replaces the current registry set with an immutable copy.
func (parseRegistry Registry) Set(parseSet Set) {
	if parseRegistry.set == nil {
		return
	}
	parseRegistry.set(BuildSet(parseSet.Flags, parseSet.Experiments))
}

// Update replaces the current registry set using the previous value.
func (parseRegistry Registry) Update(parseUpdate func(Set) Set) {
	if parseRegistry.set == nil || parseUpdate == nil {
		return
	}
	parseNext := parseUpdate(parseRegistry.Get())
	parseRegistry.set(BuildSet(parseNext.Flags, parseNext.Experiments))
}

// UseFlag subscribes the current component to one shared feature flag.
func UseFlag(parseName string, parseFallback bool) FlagHandle {
	parseRegistry := UseRegistry(Set{})
	return FlagHandle{get: func() Flag {
		return parseRegistry.Get().GetFlag(parseName, Flag{Enabled: parseFallback, Reason: "fallback"})
	}}
}

// Get returns the current flag value.
func (parseHandle FlagHandle) Get() Flag {
	if parseHandle.get == nil {
		return Flag{}
	}
	return parseHandle.get()
}

// Enabled returns whether the current flag is enabled.
func (parseHandle FlagHandle) Enabled() bool {
	return parseHandle.Get().Enabled
}

// Value returns the current flag value or the fallback when blank.
func (parseHandle FlagHandle) Value(parseFallback string) string {
	parseValue := parseHandle.Get().Value
	if parseValue == "" {
		return parseFallback
	}
	return parseValue
}

// UseExperiment subscribes the current component to one deterministic experiment assignment.
func UseExperiment(parseName string, parseSubject string) ExperimentHandle {
	parseRegistry := UseRegistry(Set{})
	return ExperimentHandle{get: func() Assignment {
		return parseRegistry.Get().GetAssignment(parseName, parseSubject)
	}}
}

// Get returns the current experiment assignment.
func (parseHandle ExperimentHandle) Get() Assignment {
	if parseHandle.get == nil {
		return Assignment{}
	}
	return parseHandle.get()
}

// getBucket returns a deterministic bucket in the range [0, parseModulo).
func getBucket(parseName string, parseSalt string, parseSubject string, parseModulo int) int {
	if parseModulo <= 0 {
		return 0
	}
	parseHash := fnv.New32a()
	_, _ = parseHash.Write([]byte(parseName))
	_, _ = parseHash.Write([]byte{0})
	_, _ = parseHash.Write([]byte(parseSalt))
	_, _ = parseHash.Write([]byte{0})
	_, _ = parseHash.Write([]byte(parseSubject))
	return int(parseHash.Sum32() % uint32(parseModulo))
}
