package workbench_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/internal/apidump"
)

// TestPublicAPIBaseline pins this package's exported surface (see internal/apidump).
func TestPublicAPIBaseline(t *testing.T) {
	apidump.Check(t, ".", "api_baseline.txt")
}
