package runtime

import (
	"fmt"
	"testing"
)

// TestReconcilerFuzzDeepSoak is a heavier one-off soak (64 seeds x 300 steps
// per mode) guarded by -short so CI keeps the fast variant.
func TestReconcilerFuzzDeepSoak(parseT *testing.T) {
	if testing.Short() {
		parseT.Skip("deep soak skipped in short mode")
	}
	for parseSeed := int64(1000); parseSeed < 1064; parseSeed++ {
		parseSeed2 := parseSeed
		parseT.Run(fmt.Sprintf("keyed-%d", parseSeed2), func(parseT2 *testing.T) {
			parseT2.Parallel()
			runReconcilerFuzz(parseT2, parseSeed2, 300, true)
		})
		parseT.Run(fmt.Sprintf("unkeyed-%d", parseSeed2), func(parseT3 *testing.T) {
			parseT3.Parallel()
			runReconcilerFuzz(parseT3, parseSeed2, 300, false)
		})
	}
}
