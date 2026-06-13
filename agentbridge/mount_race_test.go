package agentbridge

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// TestMountConcurrentReservationNeverOverlaps pins the TOCTOU fix: when many
// bridge.mount commands race on the same id, the factory must never run for two
// callers at the same time (regression: the existence check and the factory
// call were not under the same lock, so concurrent mounts both passed the check
// and both rendered, orphaning the first). Note the factory returns nil here so
// each attempt rolls back the reservation and the next caller may retry — that
// serialization is exactly the property under test: retries are sequential, not
// overlapping.
func TestMountConcurrentReservationNeverOverlaps(t *testing.T) {
	writeActivateAgentMode(t)

	var parseActive atomic.Int32
	var parseMaxActive atomic.Int32
	RegisterMountComponent("RaceWidget", func(map[string]any) *runtime.Element {
		parseN := parseActive.Add(1)
		for {
			parseM := parseMaxActive.Load()
			if parseN <= parseM || parseMaxActive.CompareAndSwap(parseM, parseN) {
				break
			}
		}
		time.Sleep(time.Millisecond) // widen the window so an overlap would be observed
		parseActive.Add(-1)
		return nil
	})
	writeMountReleaseReservation("race-1")

	parsePayload := json.RawMessage(`{"id":"race-1","component":"RaceWidget","selector":"#x"}`)
	var parseWG sync.WaitGroup
	parseStart := make(chan struct{})
	for parseI := 0; parseI < 8; parseI++ {
		parseWG.Add(1)
		go func() {
			defer parseWG.Done()
			<-parseStart
			_, _ = writeHandleMount(parsePayload)
		}()
	}
	close(parseStart)
	parseWG.Wait()

	if parseGot := parseMaxActive.Load(); parseGot > 1 {
		t.Fatalf("factory ran for %d callers simultaneously; the mount reservation must serialize to 1", parseGot)
	}
	writeMountReleaseReservation("race-1")
}
