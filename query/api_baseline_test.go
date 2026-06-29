package query_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/apidump"
)

// TestPublicAPIBaseline pins this package's exported surface. A change to any exported
// type, function, method, const, or var fails this test; regenerate the golden with
// UPDATE_API_BASELINE=1 after an intentional, reviewed change.
func TestPublicAPIBaseline(t *testing.T) {
	apidump.Check(t, ".", "api_baseline.txt")
}
