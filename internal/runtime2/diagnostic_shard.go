package runtime2

import (
	"fmt"
	"strings"
)

// ValidateDiagnosticShardID verifies diagnostic shard reporting uses a non-empty, normalized shard identity.
func ValidateDiagnosticShardID(parseShardID SchedulerShardID) error {
	parseRawShardID := string(parseShardID)
	parseTrimmedShardID := strings.TrimSpace(parseRawShardID)
	if parseTrimmedShardID == "" {
		return fmt.Errorf("runtime2: diagnostic shard ID is required")
	}
	if parseTrimmedShardID != parseRawShardID {
		return fmt.Errorf("runtime2: diagnostic shard ID %q must not contain surrounding whitespace", parseRawShardID)
	}
	return nil
}
