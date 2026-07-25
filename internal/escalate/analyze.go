// Package escalate finds database call sites that must become domain commands
// when a handle moves off-thread (plan item P3.15b).
//
// Moving a database to a worker changes what a call site can be. An in-process
// Query returns a cursor over a live connection; an off-thread one cannot,
// because there is no connection on this side. Every call site has to become a
// command — a named operation the worker performs — and that is a design
// decision per site, not a mechanical rewrite.
//
// So this is an ASSISTANT and explicitly NOT A CODEMOD, which the plan states
// outright. It finds the sites, says what each one appears to do, and proposes a
// signature. A human writes the body. The reason is not caution for its own
// sake: a rewrite that mechanically wrapped each Query in a command would
// produce one command per call site, which is a chatty protocol with the same
// round-trip problem the move was meant to solve. Deciding which sites collapse
// into one command is the actual work, and a tool cannot do it.
//
// What the tool CAN do reliably is find every site and refuse to guess about
// the ones it cannot read.
package escalate

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

// Access classifies what a call site does to the database.
type Access string

const (
	// AccessRead only reads.
	AccessRead Access = "read"
	// AccessWrite modifies data or schema.
	AccessWrite Access = "write"
	// AccessUnknown means the SQL could not be read at this call site — it is
	// built at runtime, or comes from a variable or a constant elsewhere.
	//
	// Reported as its own class rather than guessed at. A write misclassified as
	// a read becomes a command the caller believes is safe to retry, and the
	// whole point of classifying is to inform exactly that decision.
	AccessUnknown Access = "unknown"
)

// Finding is one call site that must become a command.
type Finding struct {
	// File and Line locate the call.
	File string
	Line int
	// Method is the method called: Query, QueryRow, or Exec.
	Method string
	// Receiver is the expression the method was called on, as written.
	Receiver string
	// SQL is the literal statement, empty when it could not be read.
	SQL string
	// Access classifies the statement.
	Access Access
	// ArgCount is the number of bind parameters passed.
	ArgCount int
	// ProposedCommand is a suggested command name derived from the statement.
	ProposedCommand string
	// ProposedSignature is a suggested Go declaration for the command.
	ProposedSignature string
	// NeedsHuman marks a site the tool could not classify, so a reviewer knows
	// which findings are information and which are questions.
	NeedsHuman bool
	// Note explains a NeedsHuman finding.
	Note string
}

// escalatedMethods are the calls that cannot survive the move unchanged.
var escalatedMethods = map[string]bool{
	"Query":    true,
	"QueryRow": true,
	"Exec":     true,
}

// Options configure an analysis.
type Options struct {
	// Receivers restricts findings to calls on these receiver expressions, as
	// written in the source — "db", "s.db", "app.store".
	//
	// Empty means every receiver, which is the honest default for a first pass:
	// over-reporting is a reviewer's afternoon, and under-reporting is a call
	// site that silently keeps blocking the render thread.
	Receivers []string
}

// AnalyzeSource finds escalation sites in one Go source file.
//
// Source text rather than a path so the analysis is testable without fixtures on
// disk and usable on unsaved buffers in an editor.
func AnalyzeSource(parseFileName string, parseSource string, parseOptions Options) ([]Finding, error) {
	parseFileSet := token.NewFileSet()
	parseFile, parseErr := parser.ParseFile(parseFileSet, parseFileName, parseSource, parser.AllErrors)
	if parseErr != nil {
		return nil, fmt.Errorf("escalate: parsing %s: %w", parseFileName, parseErr)
	}

	parseWanted := make(map[string]bool, len(parseOptions.Receivers))
	for _, parseReceiver := range parseOptions.Receivers {
		parseWanted[parseReceiver] = true
	}

	var parseFindings []Finding
	ast.Inspect(parseFile, func(parseNode ast.Node) bool {
		parseCall, isCall := parseNode.(*ast.CallExpr)
		if !isCall {
			return true
		}
		parseSelector, isSelector := parseCall.Fun.(*ast.SelectorExpr)
		if !isSelector || !escalatedMethods[parseSelector.Sel.Name] {
			return true
		}

		parseReceiver := renderExpr(parseSelector.X)
		if len(parseWanted) > 0 && !parseWanted[parseReceiver] {
			return true
		}

		parsePosition := parseFileSet.Position(parseCall.Pos())
		parseFindings = append(parseFindings, buildFinding(
			parsePosition.Filename, parsePosition.Line,
			parseSelector.Sel.Name, parseReceiver, parseCall))
		return true
	})

	sort.SliceStable(parseFindings, func(parseLeft int, parseRight int) bool {
		return parseFindings[parseLeft].Line < parseFindings[parseRight].Line
	})
	return parseFindings, nil
}

// buildFinding describes one call site.
func buildFinding(parseFile string, parseLine int, parseMethod string, parseReceiver string, parseCall *ast.CallExpr) Finding {
	parseFinding := Finding{
		File:     parseFile,
		Line:     parseLine,
		Method:   parseMethod,
		Receiver: parseReceiver,
		Access:   AccessUnknown,
	}

	// The SQL is conventionally the argument after a context. Both shapes exist
	// in this codebase, so the statement is found by looking for a string
	// literal rather than by position.
	parseSQLIndex := -1
	for parseIndex, parseArg := range parseCall.Args {
		if parseLiteral, isLiteral := parseArg.(*ast.BasicLit); isLiteral && parseLiteral.Kind == token.STRING {
			parseSQLIndex = parseIndex
			parseUnquoted, parseUnquoteErr := strconv.Unquote(parseLiteral.Value)
			if parseUnquoteErr == nil {
				parseFinding.SQL = parseUnquoted
			}
			break
		}
	}

	if parseSQLIndex >= 0 {
		parseFinding.ArgCount = len(parseCall.Args) - parseSQLIndex - 1
	}

	if parseFinding.SQL == "" {
		parseFinding.NeedsHuman = true
		parseFinding.Note = "the statement is not a literal here, so it cannot be classified; " +
			"read the call site to decide whether this command reads or writes"
		parseFinding.ProposedCommand = "TODO"
		parseFinding.ProposedSignature = proposeSignature("TODO", AccessUnknown, parseFinding.ArgCount)
		return parseFinding
	}

	parseFinding.Access = ClassifySQL(parseFinding.SQL)
	if parseFinding.Access == AccessUnknown {
		parseFinding.NeedsHuman = true
		parseFinding.Note = "the leading keyword is not one this tool classifies; read the statement"
	}
	parseFinding.ProposedCommand = proposeCommandName(parseFinding.SQL, parseFinding.Access)
	parseFinding.ProposedSignature = proposeSignature(parseFinding.ProposedCommand, parseFinding.Access, parseFinding.ArgCount)
	return parseFinding
}

// ClassifySQL reports whether a statement reads or writes.
//
// Exported because it is useful on its own, and because it is the piece most
// worth testing directly: a write misclassified as a read becomes a command a
// caller believes is safe to retry.
func ClassifySQL(parseSQL string) Access {
	parseKeyword := leadingKeyword(parseSQL)
	switch parseKeyword {
	case "SELECT", "PRAGMA", "EXPLAIN":
		return AccessRead
	case "WITH":
		// A CTE can end in either. Reading to the end of the statement to decide
		// would mean parsing SQL properly, and being wrong here is exactly the
		// failure worth avoiding — so it is a question rather than a guess.
		if containsWriteKeyword(parseSQL) {
			return AccessWrite
		}
		return AccessRead
	case "INSERT", "UPDATE", "DELETE", "REPLACE", "CREATE", "DROP", "ALTER", "TRUNCATE", "VACUUM", "REINDEX":
		return AccessWrite
	default:
		return AccessUnknown
	}
}

// leadingKeyword returns the first SQL word, upper-cased, skipping comments and
// leading parentheses.
func leadingKeyword(parseSQL string) string {
	parseTrimmed := strings.TrimSpace(parseSQL)
	for strings.HasPrefix(parseTrimmed, "--") {
		parseNewline := strings.IndexByte(parseTrimmed, '\n')
		if parseNewline < 0 {
			return ""
		}
		parseTrimmed = strings.TrimSpace(parseTrimmed[parseNewline+1:])
	}
	parseTrimmed = strings.TrimLeft(parseTrimmed, "( \t\n\r")

	parseEnd := strings.IndexAny(parseTrimmed, " \t\n\r(")
	if parseEnd < 0 {
		parseEnd = len(parseTrimmed)
	}
	return strings.ToUpper(parseTrimmed[:parseEnd])
}

// containsWriteKeyword reports whether a statement contains a write verb.
func containsWriteKeyword(parseSQL string) bool {
	parseUpper := strings.ToUpper(parseSQL)
	for _, parseVerb := range []string{"INSERT ", "UPDATE ", "DELETE ", "REPLACE "} {
		if strings.Contains(parseUpper, parseVerb) {
			return true
		}
	}
	return false
}

// proposeCommandName derives a command name from a statement.
//
// A proposal, not an answer. It is built from the verb and the table because
// those are the two things almost always in the eventual name, and a reviewer
// renaming "InsertOrders" to "PlaceOrder" has been helped more than one staring
// at a blank line.
func proposeCommandName(parseSQL string, parseAccess Access) string {
	parseVerb := leadingKeyword(parseSQL)
	parseTable := targetTable(parseSQL)

	if parseTable == "" {
		if parseAccess == AccessRead {
			return "Query"
		}
		return "Apply"
	}

	parseSubject := exportedIdentifier(parseTable)
	switch parseVerb {
	case "SELECT", "WITH", "EXPLAIN", "PRAGMA":
		return "List" + parseSubject
	case "INSERT", "REPLACE":
		return "Create" + parseSubject
	case "UPDATE":
		return "Update" + parseSubject
	case "DELETE":
		return "Delete" + parseSubject
	default:
		return exportedIdentifier(strings.ToLower(parseVerb)) + parseSubject
	}
}

// targetTable extracts the table a statement names, best effort.
func targetTable(parseSQL string) string {
	parseFields := strings.Fields(strings.ToUpper(parseSQL))
	parseOriginal := strings.Fields(parseSQL)

	for parseIndex, parseField := range parseFields {
		parseIsAnchor := parseField == "FROM" || parseField == "INTO" || parseField == "UPDATE" ||
			parseField == "TABLE" || parseField == "JOIN"
		if !parseIsAnchor || parseIndex+1 >= len(parseOriginal) {
			continue
		}
		parseCandidate := strings.Trim(parseOriginal[parseIndex+1], "`\"'();,")
		if parseCandidate != "" && !strings.EqualFold(parseCandidate, "IF") {
			return parseCandidate
		}
	}
	return ""
}

// exportedIdentifier turns a table name into a Go identifier.
func exportedIdentifier(parseName string) string {
	parseParts := strings.FieldsFunc(parseName, func(parseRune rune) bool {
		return parseRune == '_' || parseRune == '-' || parseRune == '.'
	})
	var parseBuilder strings.Builder
	for _, parsePart := range parseParts {
		if parsePart == "" {
			continue
		}
		parseBuilder.WriteString(strings.ToUpper(parsePart[:1]))
		if len(parsePart) > 1 {
			parseBuilder.WriteString(strings.ToLower(parsePart[1:]))
		}
	}
	if parseBuilder.Len() == 0 {
		return "Rows"
	}
	return parseBuilder.String()
}

// proposeSignature renders a suggested command declaration.
func proposeSignature(parseCommandName string, parseAccess Access, parseArgCount int) string {
	parseArgs := make([]string, 0, parseArgCount)
	for parseIndex := range parseArgCount {
		parseArgs = append(parseArgs, fmt.Sprintf("Arg%d any", parseIndex+1))
	}
	parseArgList := strings.Join(parseArgs, "; ")
	if parseArgList == "" {
		parseArgList = "// no bind parameters"
	}

	parseResult := "Result struct{ /* rows */ }"
	if parseAccess == AccessWrite {
		parseResult = "Result struct{ Affected int64 }"
	}

	return fmt.Sprintf("var %s = projection.Define[%sArgs, %sResult](%q)\n"+
		"type %sArgs struct { %s }\n"+
		"type %s%s",
		parseCommandName, parseCommandName, parseCommandName, lowerFirst(parseCommandName),
		parseCommandName, parseArgList,
		parseCommandName, parseResult)
}

func lowerFirst(parseText string) string {
	if parseText == "" {
		return parseText
	}
	return strings.ToLower(parseText[:1]) + parseText[1:]
}

// renderExpr renders a receiver expression approximately as written.
//
// Only the shapes a database handle actually appears as: an identifier, a field
// selector, or a pointer dereference. Anything else renders as a placeholder
// rather than a wrong name, since a wrong receiver in a report sends a reviewer
// to the wrong place.
func renderExpr(parseExpr ast.Expr) string {
	switch parseTyped := parseExpr.(type) {
	case *ast.Ident:
		return parseTyped.Name
	case *ast.SelectorExpr:
		return renderExpr(parseTyped.X) + "." + parseTyped.Sel.Name
	case *ast.StarExpr:
		return "*" + renderExpr(parseTyped.X)
	case *ast.CallExpr:
		return renderExpr(parseTyped.Fun) + "(...)"
	case *ast.IndexExpr:
		return renderExpr(parseTyped.X) + "[...]"
	default:
		return "<expr>"
	}
}

// Summary describes an analysis run.
type Summary struct {
	Total      int
	Reads      int
	Writes     int
	NeedsHuman int
}

// Summarize counts findings by class.
//
// The NeedsHuman count is the one that matters for planning a migration: it is
// the number of sites a person has to read, as opposed to review.
func Summarize(parseFindings []Finding) Summary {
	parseSummary := Summary{Total: len(parseFindings)}
	for _, parseFinding := range parseFindings {
		switch parseFinding.Access {
		case AccessRead:
			parseSummary.Reads++
		case AccessWrite:
			parseSummary.Writes++
		}
		if parseFinding.NeedsHuman {
			parseSummary.NeedsHuman++
		}
	}
	return parseSummary
}
