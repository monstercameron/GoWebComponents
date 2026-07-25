package app

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/examples/server/ai-chat-wizard/internal/sqlfiles"
)

// parseInventoryExampleRoot returns the absolute path of the 100-ai-chat-wizard
// example root by resolving two directories above this source file.
func parseInventoryExampleRoot(t *testing.T) string {
	t.Helper()
	_, parseFile, _, parseOk := runtime.Caller(0)
	if !parseOk {
		t.Fatal("runtime.Caller(0) failed — cannot determine test source path")
	}
	// This file lives at <root>/server/app/; two dirs up is <root>.
	return filepath.Clean(filepath.Join(filepath.Dir(parseFile), "..", ".."))
}

// TestSQLInventoryAllQueriesLoad calls parseLoadStoreQueries and confirms that
// every storeQueries field is non-empty, proving every registered SQL path
// exists on disk and is loadable by the production loader.
func TestSQLInventoryAllQueriesLoad(t *testing.T) {
	parseQ, parseErr := parseLoadStoreQueries()
	if parseErr != nil {
		t.Fatalf("parseLoadStoreQueries: %v", parseErr)
	}
	parseV := reflect.ValueOf(parseQ)
	parseT := parseV.Type()
	for parseI := 0; parseI < parseV.NumField(); parseI++ {
		if parseV.Field(parseI).String() == "" {
			t.Errorf("storeQueries.%s is empty after parseLoadStoreQueries", parseT.Field(parseI).Name)
		}
	}
}

// TestSQLInventoryDiskFilesRegistered walks every .sql file under sql/store,
// verifies each is loadable via sqlfiles.ParseLoad, and confirms the disk file
// count matches the storeQueries field count. A new .sql file that lands
// without a corresponding parseLoadStoreQuery registration will cause the
// count check to fail, making the gap immediately visible.
func TestSQLInventoryDiskFilesRegistered(t *testing.T) {
	parseRoot := parseInventoryExampleRoot(t)
	parseSQLStoreDir := filepath.Join(parseRoot, "sql", "store")
	if _, parseStatErr := os.Stat(parseSQLStoreDir); parseStatErr != nil {
		t.Fatalf("sql/store not found at %s: %v — check CHAT_WIZARD_ROOT or cwd", parseSQLStoreDir, parseStatErr)
	}

	parseSQLRootPrefix := filepath.ToSlash(filepath.Join(parseRoot, "sql")) + "/"
	var parseDiskPaths []string
	parseWalkErr := filepath.WalkDir(parseSQLStoreDir, func(parsePath string, parseDe os.DirEntry, parseErrIn error) error {
		if parseErrIn != nil {
			return parseErrIn
		}
		if !parseDe.IsDir() && strings.HasSuffix(parsePath, ".sql") {
			// Convert to forward-slash relative form expected by sqlfiles.ParseLoad,
			// e.g. "store/admin/get_admin_dashboard_summary.sql".
			parseRelPath := strings.TrimPrefix(filepath.ToSlash(parsePath), parseSQLRootPrefix)
			parseDiskPaths = append(parseDiskPaths, parseRelPath)
		}
		return nil
	})
	if parseWalkErr != nil {
		t.Fatalf("walk sql/store: %v", parseWalkErr)
	}
	if len(parseDiskPaths) == 0 {
		t.Fatal("no .sql files found under sql/store — root discovery may be wrong")
	}

	// Each disk file must be loadable via the production loader.
	for _, parseRelPath := range parseDiskPaths {
		t.Run(parseRelPath, func(t *testing.T) {
			parseText, parseLoadErr := sqlfiles.ParseLoad(parseRelPath)
			if parseLoadErr != nil {
				t.Errorf("sqlfiles.ParseLoad(%q): %v", parseRelPath, parseLoadErr)
				return
			}
			if strings.TrimSpace(parseText) == "" {
				t.Errorf("sqlfiles.ParseLoad(%q): returned empty text", parseRelPath)
			}
		})
	}

	// Guard against new .sql files that land on disk without a storeQueries
	// field (and therefore no registration in parseLoadStoreQueries).
	parseStructFieldCount := reflect.TypeFor[storeQueries]().NumField()
	if len(parseDiskPaths) != parseStructFieldCount {
		t.Errorf(
			"disk SQL file count (%d) != storeQueries field count (%d): "+
				"a new .sql file may have been added without a load registration in "+
				"parseLoadStoreQueries, or a registered path has no matching file on disk",
			len(parseDiskPaths), parseStructFieldCount,
		)
	}
	t.Logf("verified %d SQL files under sql/store are loadable", len(parseDiskPaths))
}
