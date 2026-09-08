package escalate_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/escalate"
)

// v5 P3.15b — the escalation assistant.
//
// Criterion: flags every Query/Exec on an escalated handle, classifies
// read/write, proposes a signature. NOT a codemod — bodies are manual.
//
// Two properties carry the whole thing, and they pull against each other:
//
//   - Completeness. A missed call site is one that silently keeps blocking the
//     render thread after a migration everyone believes is finished.
//   - Refusal to guess. A write misclassified as a read becomes a command the
//     caller believes is safe to retry, which is exactly the decision the
//     classification exists to inform.
//
// So the tool over-reports by default and marks what it cannot read.

const sampleSource = `package app

import "context"

type store struct{ db *DB }

func (parseStore *store) load(parseCtx context.Context) {
	parseStore.db.Query(parseCtx, "SELECT id, name FROM users WHERE active = ?", true)
	parseStore.db.QueryRow(parseCtx, "SELECT count(*) FROM orders")
}

func (parseStore *store) save(parseCtx context.Context, parseName string) {
	parseStore.db.Exec(parseCtx, "INSERT INTO users (name) VALUES (?)", parseName)
	parseStore.db.Exec(parseCtx, "UPDATE user_settings SET theme = ? WHERE user_id = ?", "dark", 1)
	parseStore.db.Exec(parseCtx, "DELETE FROM sessions WHERE expires_at < ?", 0)
}

func (parseStore *store) dynamic(parseCtx context.Context, parseSQL string) {
	parseStore.db.Query(parseCtx, parseSQL)
}

func unrelated() {
	somethingElse.Query("not a database")
}
`

func analyze(parseT *testing.T, parseSource string, parseOptions escalate.Options) []escalate.Finding {
	parseT.Helper()
	parseFindings, parseErr := escalate.AnalyzeSource("app.go", parseSource, parseOptions)
	if parseErr != nil {
		parseT.Fatalf("AnalyzeSource: %v", parseErr)
	}
	return parseFindings
}

// ------------------------------------------------------------ completeness

// TestEveryCallSiteIsFlagged is the first half of the criterion.
func TestEveryCallSiteIsFlagged(parseT *testing.T) {
	parseFindings := analyze(parseT, sampleSource, escalate.Options{Receivers: []string{"parseStore.db"}})

	if len(parseFindings) != 6 {
		parseT.Fatalf("found %d call sites, want 6 — a missed site keeps blocking the render thread after a migration everyone believes is done", len(parseFindings))
	}

	parseMethods := map[string]int{}
	for _, parseFinding := range parseFindings {
		parseMethods[parseFinding.Method]++
	}
	if parseMethods["Query"] != 2 || parseMethods["QueryRow"] != 1 || parseMethods["Exec"] != 3 {
		parseT.Errorf("methods = %v, want 2 Query, 1 QueryRow, 3 Exec", parseMethods)
	}
}

// TestReceiverFilterExcludesUnrelatedCalls: Query is a common method name, and
// a report full of calls on unrelated types trains a reviewer to skim.
func TestReceiverFilterExcludesUnrelatedCalls(parseT *testing.T) {
	parseFiltered := analyze(parseT, sampleSource, escalate.Options{Receivers: []string{"parseStore.db"}})
	for _, parseFinding := range parseFiltered {
		if parseFinding.Receiver != "parseStore.db" {
			parseT.Errorf("finding on %q slipped past the receiver filter", parseFinding.Receiver)
		}
	}

	// With no filter the unrelated call IS reported. Over-reporting costs a
	// reviewer an afternoon; under-reporting costs a silent blocking call.
	parseAll := analyze(parseT, sampleSource, escalate.Options{})
	if len(parseAll) <= len(parseFiltered) {
		parseT.Error("the unfiltered pass must report at least as much as the filtered one")
	}
}

func TestFindingsCarryTheirLocation(parseT *testing.T) {
	parseFindings := analyze(parseT, sampleSource, escalate.Options{Receivers: []string{"parseStore.db"}})
	for _, parseFinding := range parseFindings {
		if parseFinding.File == "" || parseFinding.Line <= 0 {
			parseT.Errorf("finding %+v has no location; a reviewer cannot act on it", parseFinding)
		}
	}
	// Sorted by line so a reviewer reads the file top to bottom.
	for parseIndex := 1; parseIndex < len(parseFindings); parseIndex++ {
		if parseFindings[parseIndex].Line < parseFindings[parseIndex-1].Line {
			parseT.Error("findings are not in line order")
		}
	}
}

// --------------------------------------------------------- classification

func TestReadsAndWritesAreClassified(parseT *testing.T) {
	parseFindings := analyze(parseT, sampleSource, escalate.Options{Receivers: []string{"parseStore.db"}})

	parseByLine := map[string]escalate.Access{}
	for _, parseFinding := range parseFindings {
		if parseFinding.SQL != "" {
			parseByLine[parseFinding.SQL] = parseFinding.Access
		}
	}

	for parseSQL, parseWant := range map[string]escalate.Access{
		"SELECT id, name FROM users WHERE active = ?":          escalate.AccessRead,
		"SELECT count(*) FROM orders":                          escalate.AccessRead,
		"INSERT INTO users (name) VALUES (?)":                  escalate.AccessWrite,
		"UPDATE user_settings SET theme = ? WHERE user_id = ?": escalate.AccessWrite,
		"DELETE FROM sessions WHERE expires_at < ?":            escalate.AccessWrite,
	} {
		if parseByLine[parseSQL] != parseWant {
			parseT.Errorf("%q classified %q, want %q", parseSQL, parseByLine[parseSQL], parseWant)
		}
	}
}

// TestClassifySQLCoversTheVerbsThatMatter tests the classifier directly,
// because a write misclassified as a read becomes a command a caller believes is
// safe to retry.
func TestClassifySQLCoversTheVerbsThatMatter(parseT *testing.T) {
	for parseSQL, parseWant := range map[string]escalate.Access{
		"SELECT * FROM t":                 escalate.AccessRead,
		"  select lower-case works":       escalate.AccessRead,
		"PRAGMA journal_mode":             escalate.AccessRead,
		"EXPLAIN QUERY PLAN SELECT 1":     escalate.AccessRead,
		"INSERT INTO t VALUES (1)":        escalate.AccessWrite,
		"UPDATE t SET a = 1":              escalate.AccessWrite,
		"DELETE FROM t":                   escalate.AccessWrite,
		"REPLACE INTO t VALUES (1)":       escalate.AccessWrite,
		"CREATE TABLE t (a INTEGER)":      escalate.AccessWrite,
		"DROP TABLE t":                    escalate.AccessWrite,
		"ALTER TABLE t ADD COLUMN b TEXT": escalate.AccessWrite,
		"VACUUM":                          escalate.AccessWrite,
		"BEGIN":                           escalate.AccessUnknown,
		"":                                escalate.AccessUnknown,
	} {
		if parseGot := escalate.ClassifySQL(parseSQL); parseGot != parseWant {
			parseT.Errorf("ClassifySQL(%q) = %q, want %q", parseSQL, parseGot, parseWant)
		}
	}
}

// TestCTEsAreClassifiedByWhatTheyActuallyDo: a WITH can end in either a SELECT
// or an INSERT, and calling every CTE a read would hand a write to a retry
// policy as safe.
func TestCTEsAreClassifiedByWhatTheyActuallyDo(parseT *testing.T) {
	parseReadCTE := "WITH recent AS (SELECT * FROM orders) SELECT * FROM recent"
	if parseGot := escalate.ClassifySQL(parseReadCTE); parseGot != escalate.AccessRead {
		parseT.Errorf("a read CTE classified %q, want read", parseGot)
	}

	parseWriteCTE := "WITH stale AS (SELECT id FROM sessions) DELETE FROM sessions WHERE id IN (SELECT id FROM stale)"
	if parseGot := escalate.ClassifySQL(parseWriteCTE); parseGot != escalate.AccessWrite {
		parseT.Errorf("a writing CTE classified %q, want write — a write handed to a retry policy as safe", parseGot)
	}
}

func TestLeadingCommentsDoNotHideTheVerb(parseT *testing.T) {
	parseSQL := "-- refresh the cache\n-- see issue 412\nDELETE FROM cache"
	if parseGot := escalate.ClassifySQL(parseSQL); parseGot != escalate.AccessWrite {
		parseT.Errorf("a commented statement classified %q, want write", parseGot)
	}
}

// ------------------------------------------------------- refusal to guess

// TestDynamicSQLIsFlaggedNotGuessed is the property that makes this an
// assistant rather than a codemod.
func TestDynamicSQLIsFlaggedNotGuessed(parseT *testing.T) {
	parseFindings := analyze(parseT, sampleSource, escalate.Options{Receivers: []string{"parseStore.db"}})

	parseFound := false
	for _, parseFinding := range parseFindings {
		if parseFinding.SQL != "" {
			continue
		}
		parseFound = true
		if parseFinding.Access != escalate.AccessUnknown {
			parseT.Errorf("unreadable SQL classified as %q; it must be unknown", parseFinding.Access)
		}
		if !parseFinding.NeedsHuman {
			parseT.Error("a site the tool cannot read must be marked for a human")
		}
		if parseFinding.Note == "" {
			parseT.Error("a needs-human finding must say why")
		}
	}
	if !parseFound {
		parseT.Fatal("the dynamic-SQL call site was not reported at all")
	}
}

func TestUnrecognizedVerbsAreQuestionsNotGuesses(parseT *testing.T) {
	parseSource := `package app
func f() { db.Exec(ctx, "ATTACH DATABASE 'other.db' AS other") }
`
	parseFindings := analyze(parseT, parseSource, escalate.Options{Receivers: []string{"db"}})
	if len(parseFindings) != 1 {
		parseT.Fatalf("found %d sites, want 1", len(parseFindings))
	}
	if parseFindings[0].Access != escalate.AccessUnknown || !parseFindings[0].NeedsHuman {
		parseT.Errorf("finding = %+v, want unknown and needs-human", parseFindings[0])
	}
}

// ------------------------------------------------------------- proposals

// TestSignaturesAreProposedForEverySite is the third part of the criterion.
func TestSignaturesAreProposedForEverySite(parseT *testing.T) {
	parseFindings := analyze(parseT, sampleSource, escalate.Options{Receivers: []string{"parseStore.db"}})

	for _, parseFinding := range parseFindings {
		if parseFinding.ProposedCommand == "" {
			parseT.Errorf("line %d has no proposed command", parseFinding.Line)
		}
		if !strings.Contains(parseFinding.ProposedSignature, "projection.Define") {
			parseT.Errorf("line %d: signature does not declare a command:\n%s",
				parseFinding.Line, parseFinding.ProposedSignature)
		}
		// A write's result carries what a write returns; a read's carries rows.
		// Getting this backwards would propose a signature that cannot express
		// the operation.
		if parseFinding.Access == escalate.AccessWrite &&
			!strings.Contains(parseFinding.ProposedSignature, "Affected") {
			parseT.Errorf("line %d: a write's proposed result has no affected count:\n%s",
				parseFinding.Line, parseFinding.ProposedSignature)
		}
	}
}

func TestProposedNamesUseTheVerbAndTable(parseT *testing.T) {
	parseFindings := analyze(parseT, sampleSource, escalate.Options{Receivers: []string{"parseStore.db"}})

	parseNamesBySQL := map[string]string{}
	for _, parseFinding := range parseFindings {
		parseNamesBySQL[parseFinding.SQL] = parseFinding.ProposedCommand
	}

	for parseSQL, parseWant := range map[string]string{
		"SELECT id, name FROM users WHERE active = ?":          "ListUsers",
		"INSERT INTO users (name) VALUES (?)":                  "CreateUsers",
		"UPDATE user_settings SET theme = ? WHERE user_id = ?": "UpdateUserSettings",
		"DELETE FROM sessions WHERE expires_at < ?":            "DeleteSessions",
	} {
		if parseNamesBySQL[parseSQL] != parseWant {
			parseT.Errorf("%q proposed %q, want %q", parseSQL, parseNamesBySQL[parseSQL], parseWant)
		}
	}
}

func TestBindParametersAreCounted(parseT *testing.T) {
	parseSource := `package app
func f() { db.Exec(ctx, "UPDATE t SET a = ?, b = ? WHERE c = ?", 1, 2, 3) }
`
	parseFindings := analyze(parseT, parseSource, escalate.Options{Receivers: []string{"db"}})
	if parseFindings[0].ArgCount != 3 {
		parseT.Errorf("ArgCount = %d, want 3", parseFindings[0].ArgCount)
	}
	if !strings.Contains(parseFindings[0].ProposedSignature, "Arg3") {
		parseT.Errorf("the proposed args do not cover all three:\n%s", parseFindings[0].ProposedSignature)
	}
}

func TestNoBindParametersProposesAnEmptyArgs(parseT *testing.T) {
	parseSource := `package app
func f() { db.Query(ctx, "SELECT * FROM t") }
`
	parseFindings := analyze(parseT, parseSource, escalate.Options{Receivers: []string{"db"}})
	if parseFindings[0].ArgCount != 0 {
		parseT.Errorf("ArgCount = %d, want 0", parseFindings[0].ArgCount)
	}
	if !strings.Contains(parseFindings[0].ProposedSignature, "no bind parameters") {
		parseT.Errorf("signature should note the absence of parameters:\n%s", parseFindings[0].ProposedSignature)
	}
}

// --------------------------------------------------------------- summary

// TestSummaryCountsWhatAMigrationNeedsToPlan: the needs-human count is the
// number of sites someone has to READ, as opposed to review.
func TestSummaryCountsWhatAMigrationNeedsToPlan(parseT *testing.T) {
	parseFindings := analyze(parseT, sampleSource, escalate.Options{Receivers: []string{"parseStore.db"}})
	parseSummary := escalate.Summarize(parseFindings)

	if parseSummary.Total != 6 {
		parseT.Errorf("total = %d, want 6", parseSummary.Total)
	}
	if parseSummary.Reads != 2 {
		parseT.Errorf("reads = %d, want 2", parseSummary.Reads)
	}
	if parseSummary.Writes != 3 {
		parseT.Errorf("writes = %d, want 3", parseSummary.Writes)
	}
	if parseSummary.NeedsHuman != 1 {
		parseT.Errorf("needsHuman = %d, want 1 (the dynamic-SQL site)", parseSummary.NeedsHuman)
	}
}

// --------------------------------------------------------------- guards

func TestUnparseableSourceIsReported(parseT *testing.T) {
	if _, parseErr := escalate.AnalyzeSource("broken.go", "package app\nfunc f( {", escalate.Options{}); parseErr == nil {
		parseT.Error("source that does not parse must be reported rather than silently yielding no findings")
	}
}

func TestSourceWithNoDatabaseCallsFindsNothing(parseT *testing.T) {
	parseFindings := analyze(parseT, "package app\nfunc f() { println(\"hello\") }\n", escalate.Options{})
	if len(parseFindings) != 0 {
		parseT.Errorf("found %d sites in source with no database calls", len(parseFindings))
	}
	if parseSummary := escalate.Summarize(parseFindings); parseSummary.Total != 0 {
		parseT.Errorf("summary = %+v, want empty", parseSummary)
	}
}

// TestUnusualReceiversDoNotProduceWrongNames: a wrong receiver in a report
// sends a reviewer to the wrong place, which is worse than a placeholder.
func TestUnusualReceiversDoNotProduceWrongNames(parseT *testing.T) {
	parseSource := `package app
func f() {
	handles[0].Query(ctx, "SELECT 1")
	openDB().Exec(ctx, "DELETE FROM t")
}
`
	parseFindings := analyze(parseT, parseSource, escalate.Options{})
	if len(parseFindings) != 2 {
		parseT.Fatalf("found %d sites, want 2", len(parseFindings))
	}
	for _, parseFinding := range parseFindings {
		if parseFinding.Receiver == "" {
			parseT.Error("a receiver must render as something a reviewer can recognize")
		}
	}
}
