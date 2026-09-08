package projection_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/delta"
	"github.com/monstercameron/GoWebComponents/v6/projection"
)

type scalingRow struct {
	Name string `json:"name"`
}

func buildScalingOps(parseCount int) []projection.Op {
	parseEngine := delta.New()
	parseRows := make([]delta.Row, 0, parseCount)
	for parseIndex := 0; parseIndex < parseCount; parseIndex++ {
		parsePayload, _ := json.Marshal(scalingRow{Name: fmt.Sprintf("row-%d", parseIndex)})
		parseRows = append(parseRows, delta.Row{
			Key:     delta.Key(fmt.Sprintf("k-%06d", parseIndex)),
			Version: uint64(parseIndex + 1),
			Payload: parsePayload,
		})
	}
	parseOps, parseErr := parseEngine.Publish(parseRows)
	if parseErr != nil {
		panic(parseErr)
	}
	return parseOps
}

// Apply used to be O(n²): every structural op invalidated the key index, and the
// next op's lookup rebuilt the whole map. This asserts the shape is now linear.
//
// Ratio rather than absolute time, because absolute timings are meaningless on a
// thermally throttling laptop. 8x the ops on an O(n²) implementation costs ~64x;
// on a linear one it costs ~8x. The threshold sits well between the two so the
// test fails on a genuine regression and not on noise.
func TestApplyScalesLinearlyWithOpCount(parseT *testing.T) {
	parseMeasure := func(parseCount int) float64 {
		parseOps := buildScalingOps(parseCount)
		parseResult := testing.Benchmark(func(parseB *testing.B) {
			for parseIteration := 0; parseIteration < parseB.N; parseIteration++ {
				parseB.StopTimer()
				parseProjection, _ := projection.New(func(parsePayload []byte) (scalingRow, error) {
					var parseRow scalingRow
					return parseRow, json.Unmarshal(parsePayload, &parseRow)
				}, projection.Options{})
				parseB.StartTimer()
				if parseErr := parseProjection.Apply(parseOps); parseErr != nil {
					parseB.Fatalf("apply: %v", parseErr)
				}
			}
		})
		return float64(parseResult.NsPerOp())
	}

	parseSmall := parseMeasure(500)
	parseLarge := parseMeasure(4000) // 8x the ops
	parseRatio := parseLarge / parseSmall
	parseT.Logf("500 ops: %.0f ns | 4000 ops: %.0f ns | ratio %.1fx (linear ≈ 8x, quadratic ≈ 64x)",
		parseSmall, parseLarge, parseRatio)

	if parseRatio > 24 {
		parseT.Fatalf("Apply is scaling super-linearly (%.1fx for 8x the ops); the key index is being "+
			"invalidated per op again — see reindexFrom", parseRatio)
	}
}
