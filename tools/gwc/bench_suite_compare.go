package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
)

// Verdicts assigned to each compared entry.
const (
	benchSuiteVerdictImproved  = "improved"
	benchSuiteVerdictRegressed = "regressed"
	benchSuiteVerdictNoise     = "noise"
)

const (
	// benchSuiteNoiseFloorPct is the smallest delta ever flagged as significant.
	benchSuiteNoiseFloorPct = 2.0
	// benchSuiteMADMultiplier scales the combined relative MAD into the significance threshold.
	benchSuiteMADMultiplier = 3.0
)

// benchSuiteDelta is the comparison result for one entry present in both reports.
type benchSuiteDelta struct {
	ID           string  `json:"id"`
	Layer        string  `json:"layer"`
	Unit         string  `json:"unit"`
	Baseline     float64 `json:"baseline"`
	Candidate    float64 `json:"candidate"`
	DeltaPct     float64 `json:"deltaPct"`
	ThresholdPct float64 `json:"thresholdPct"`
	Verdict      string  `json:"verdict"`
}

// benchSuiteLayerDelta is the geometric-mean change of every matched entry in one layer.
type benchSuiteLayerDelta struct {
	Layer      string  `json:"layer"`
	Entries    int     `json:"entries"`
	GeomeanPct float64 `json:"geomeanPct"`
}

// benchSuiteComparison is the full outcome of comparing two suite reports.
type benchSuiteComparison struct {
	Deltas        []benchSuiteDelta      `json:"deltas"`
	Layers        []benchSuiteLayerDelta `json:"layers"`
	Improved      int                    `json:"improved"`
	Regressed     int                    `json:"regressed"`
	Noise         int                    `json:"noise"`
	OnlyBaseline  []string               `json:"onlyBaseline,omitempty"`
	OnlyCandidate []string               `json:"onlyCandidate,omitempty"`
}

// benchSuiteComparable is an entry flattened for comparison, including size rows.
type benchSuiteComparable struct {
	layer  string
	unit   string
	median float64
	mad    float64
}

// flattenBenchSuiteReport indexes every comparable value by id. Only lanes that ran contribute
// micro entries; size rows are deterministic so they carry a zero MAD.
func flattenBenchSuiteReport(parseReport benchSuiteReport) map[string]benchSuiteComparable {
	parseIndex := map[string]benchSuiteComparable{}
	for _, parseLane := range parseReport.Lanes {
		if parseLane.Status == benchSuiteStatusNotRun {
			continue
		}
		for _, parseEntry := range parseLane.Entries {
			parseIndex[parseEntry.ID] = benchSuiteComparable{layer: parseLane.Layer, unit: parseEntry.Unit, median: parseEntry.Median, mad: parseEntry.MAD}
		}
	}
	for _, parseSize := range parseReport.Size {
		for _, parseRow := range []struct {
			name  string
			bytes int64
		}{{"raw", parseSize.RawBytes}, {"gzip", parseSize.GzipBytes}, {"brotli", parseSize.BrotliBytes}} {
			if parseRow.bytes > 0 {
				parseIndex["size/"+parseSize.Artifact+"/"+parseRow.name] = benchSuiteComparable{layer: "size", unit: "bytes", median: float64(parseRow.bytes)}
			}
		}
	}
	return parseIndex
}

// getBenchSuiteMachineMismatch lists the machine fields that differ; empty means comparable.
func getBenchSuiteMachineMismatch(parseBaseline *benchmarkMachine, parseCandidate *benchmarkMachine) []string {
	if parseBaseline == nil && parseCandidate == nil {
		return nil
	}
	if parseBaseline == nil || parseCandidate == nil {
		return []string{"machine block missing in one report"}
	}
	parseDiffs := []string{}
	for _, parseField := range []struct{ name, left, right string }{
		{"machine.arch", parseBaseline.Arch, parseCandidate.Arch},
		{"machine.hostArch", parseBaseline.HostArch, parseCandidate.HostArch},
		{"machine.cpuModel", parseBaseline.CPUModel, parseCandidate.CPUModel},
	} {
		if parseField.left != parseField.right {
			parseDiffs = append(parseDiffs, fmt.Sprintf("%s: %q vs %q", parseField.name, parseField.left, parseField.right))
		}
	}
	return parseDiffs
}

// compareBenchSuiteReports computes per-entry deltas and per-layer geomeans. Lower is better for
// every unit. An entry is significant only when |delta| exceeds
// max(2%, 3 * sqrt(madBase^2 + madCand^2) / medianBase).
func compareBenchSuiteReports(parseBaseline benchSuiteReport, parseCandidate benchSuiteReport) benchSuiteComparison {
	parseBase := flattenBenchSuiteReport(parseBaseline)
	parseCand := flattenBenchSuiteReport(parseCandidate)
	parseResult := benchSuiteComparison{Deltas: []benchSuiteDelta{}, Layers: []benchSuiteLayerDelta{}}
	parseLogs := map[string][]float64{}
	for parseID, parseLeft := range parseBase {
		parseRight, isFound := parseCand[parseID]
		if !isFound {
			parseResult.OnlyBaseline = append(parseResult.OnlyBaseline, parseID)
			continue
		}
		if parseLeft.median <= 0 || parseRight.median <= 0 {
			continue
		}
		parseDeltaPct := (parseRight.median - parseLeft.median) / parseLeft.median * 100
		parseCombined := math.Hypot(parseLeft.mad, parseRight.mad)
		parseThreshold := math.Max(benchSuiteNoiseFloorPct, benchSuiteMADMultiplier*parseCombined/parseLeft.median*100)
		parseVerdict := benchSuiteVerdictNoise
		if math.Abs(parseDeltaPct) > parseThreshold {
			if parseDeltaPct > 0 {
				parseVerdict = benchSuiteVerdictRegressed
				parseResult.Regressed++
			} else {
				parseVerdict = benchSuiteVerdictImproved
				parseResult.Improved++
			}
		} else {
			parseResult.Noise++
		}
		parseResult.Deltas = append(parseResult.Deltas, benchSuiteDelta{
			ID: parseID, Layer: parseLeft.layer, Unit: parseLeft.unit,
			Baseline: parseLeft.median, Candidate: parseRight.median,
			DeltaPct: parseDeltaPct, ThresholdPct: parseThreshold, Verdict: parseVerdict,
		})
		parseLogs[parseLeft.layer] = append(parseLogs[parseLeft.layer], parseRight.median/parseLeft.median)
	}
	for parseID := range parseCand {
		if _, isFound := parseBase[parseID]; !isFound {
			parseResult.OnlyCandidate = append(parseResult.OnlyCandidate, parseID)
		}
	}
	sort.Strings(parseResult.OnlyBaseline)
	sort.Strings(parseResult.OnlyCandidate)
	sort.Slice(parseResult.Deltas, func(parseI, parseJ int) bool {
		if parseResult.Deltas[parseI].Layer != parseResult.Deltas[parseJ].Layer {
			return parseResult.Deltas[parseI].Layer < parseResult.Deltas[parseJ].Layer
		}
		return parseResult.Deltas[parseI].ID < parseResult.Deltas[parseJ].ID
	})
	for parseLayer, parseRatios := range parseLogs {
		parseResult.Layers = append(parseResult.Layers, benchSuiteLayerDelta{
			Layer: parseLayer, Entries: len(parseRatios), GeomeanPct: (geometricMean(parseRatios) - 1) * 100,
		})
	}
	sort.Slice(parseResult.Layers, func(parseI, parseJ int) bool { return parseResult.Layers[parseI].Layer < parseResult.Layers[parseJ].Layer })
	return parseResult
}

// getBenchSuiteRegressionGate returns an error when a significant, material
// regression exceeds the percent cap. The baseline floor applies only to
// ns/op timings; tiny CI microbenchmarks remain visible in the report.
func getBenchSuiteRegressionGate(parseComparison benchSuiteComparison, parseMaxPct, parseMinBaselineNs float64) error {
	if parseMaxPct <= 0 {
		return nil
	}
	parseOver := []string{}
	for _, parseDelta := range parseComparison.Deltas {
		if parseDelta.Unit == "ns/op" && parseDelta.Baseline < parseMinBaselineNs {
			continue
		}
		if parseDelta.Verdict == benchSuiteVerdictRegressed && parseDelta.DeltaPct > parseMaxPct {
			parseOver = append(parseOver, fmt.Sprintf("%s (+%.1f%%)", parseDelta.ID, parseDelta.DeltaPct))
		}
	}
	if len(parseOver) == 0 {
		return nil
	}
	return fmt.Errorf("bench suite regression gate: %d entr(ies) regressed beyond %.1f%%: %s", len(parseOver), parseMaxPct, strings.Join(parseOver, ", "))
}

// parseBenchSuiteInterleaved parses flags that may appear before, between or after positionals,
// which the standard flag package does not support on its own.
func parseBenchSuiteInterleaved(parseFs *flag.FlagSet, parseArgs []string) ([]string, error) {
	parsePositional := []string{}
	parseRest := parseArgs
	for {
		if parseErr := parseFs.Parse(parseRest); parseErr != nil {
			return nil, parseErr
		}
		if parseFs.NArg() == 0 {
			return parsePositional, nil
		}
		parsePositional = append(parsePositional, parseFs.Arg(0))
		parseRest = parseFs.Args()[1:]
	}
}

// runBenchSuiteCompare implements `gwc bench suite compare`.
func runBenchSuiteCompare(parseArgs []string) error {
	parseFs := flag.NewFlagSet("bench suite compare", flag.ContinueOnError)
	parseFs.SetOutput(os.Stdout)
	parseFs.Usage = func() {
		fmt.Fprintln(parseFs.Output(), "Usage: gwc bench suite compare [flags] <baseline.json> <candidate.json>\n\n"+
			"Prints per-entry median delta, a significance verdict and per-layer geomeans.\n"+
			"An entry is flagged only when |delta| exceeds max(2%, 3*sqrt(madBase^2+madCand^2)/medianBase).\n"+
			"Lower is better for every unit. Not-run lanes are ignored.")
		parseFs.PrintDefaults()
	}
	parseMax := parseFs.Float64("max-regress-pct", 0, "Exit non-zero when any significant regression exceeds this percent; 0 disables the gate")
	parseMinBaselineNs := parseFs.Float64("min-baseline-ns", 0, "Only gate ns/op entries whose baseline is at least this many nanoseconds; all entries remain in the report")
	isAllowMismatch := parseFs.Bool("allow-machine-mismatch", false, "Compare even when machine.arch, hostArch or cpuModel differ")
	parseJSON := parseFs.Bool("json", false, "Emit the comparison as JSON")
	parsePositional, parseErr := parseBenchSuiteInterleaved(parseFs, parseArgs)
	if parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	if len(parsePositional) != 2 {
		return errors.New("bench suite compare needs exactly two arguments: <baseline.json> <candidate.json>")
	}
	if *parseMinBaselineNs < 0 {
		return errors.New("-min-baseline-ns must be non-negative")
	}
	parseBaseline, parseErr2 := loadBenchSuiteReport(parsePositional[0])
	if parseErr2 != nil {
		return parseErr2
	}
	parseCandidate, parseErr3 := loadBenchSuiteReport(parsePositional[1])
	if parseErr3 != nil {
		return parseErr3
	}
	if parseDiffs := getBenchSuiteMachineMismatch(parseBaseline.Machine, parseCandidate.Machine); len(parseDiffs) > 0 && !*isAllowMismatch {
		return fmt.Errorf("refusing to compare reports from different machines (%s); pass -allow-machine-mismatch to override", strings.Join(parseDiffs, "; "))
	}
	parseComparison := compareBenchSuiteReports(parseBaseline, parseCandidate)
	if *parseJSON {
		parseEncoder := json.NewEncoder(os.Stdout)
		parseEncoder.SetIndent("", "  ")
		if parseEncodeErr := parseEncoder.Encode(parseComparison); parseEncodeErr != nil {
			return parseEncodeErr
		}
	} else {
		printBenchSuiteComparison(parseComparison)
	}
	return getBenchSuiteRegressionGate(parseComparison, *parseMax, *parseMinBaselineNs)
}

// printBenchSuiteComparison prints deltas, layer geomeans and unmatched ids.
func printBenchSuiteComparison(parseComparison benchSuiteComparison) {
	fmt.Println("GWC bench suite compare")
	for _, parseDelta := range parseComparison.Deltas {
		parseFlag := " "
		if parseDelta.Verdict != benchSuiteVerdictNoise {
			parseFlag = "*"
		}
		fmt.Printf("%s %-9s %+8.2f%% (thr %.1f%%)  %s\n", parseFlag, parseDelta.Verdict, parseDelta.DeltaPct, parseDelta.ThresholdPct, parseDelta.ID)
	}
	for _, parseLayer := range parseComparison.Layers {
		fmt.Printf("layer %-12s geomean %+.2f%% over %d entries\n", parseLayer.Layer, parseLayer.GeomeanPct, parseLayer.Entries)
	}
	fmt.Printf("%d improved, %d regressed, %d noise; %d only in baseline, %d only in candidate\n",
		parseComparison.Improved, parseComparison.Regressed, parseComparison.Noise, len(parseComparison.OnlyBaseline), len(parseComparison.OnlyCandidate))
}
