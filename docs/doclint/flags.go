package doclint

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// FlagRef is one `-flag` token found on a gwc command line in a doc, which does
// not correspond to any flag defined in the gwc launcher source.
type FlagRef struct {
	DocPath string
	Line    int
	Flag    string
}

var (
	// gwcDir is the launcher source directory, relative to the repo root.
	gwcDirRel = filepath.FromSlash("tools/gwc")

	// nameFirstFlagPattern matches flag definitions whose name is the first
	// argument: parseFs.String("name", ...), .Bool(...), .Int(...), etc.
	nameFirstFlagPattern = regexp.MustCompile(`\.(?:String|Bool|Int|Int64|Uint|Uint64|Duration|Float64)\("([a-z][a-z0-9-]*)"`)
	// typedVarFlagPattern matches the *Var family — .StringVar(&x, "name", ...),
	// .BoolVar, .IntVar, .DurationVar, etc. — where the name is the SECOND
	// argument. \.Var\( does NOT match `.StringVar(` (the char before `Var(` is a
	// letter, not a dot), so without this the whole *Var family of standard flag
	// definitions is silently missed: a future -foo defined via StringVar would be
	// absent from the known set and every doc referencing it flagged as unknown.
	typedVarFlagPattern = regexp.MustCompile(`\.(?:String|Bool|Int|Int64|Uint|Uint64|Duration|Float64)Var\([^,]+,\s*"([a-z][a-z0-9-]*)"`)
	// varFlagPattern matches flag.Var(&value, "name", ...), where the flag name
	// is the SECOND argument (repeatable/custom flags like -lane, -ext).
	varFlagPattern = regexp.MustCompile(`\.Var\([^,]+,\s*"([a-z][a-z0-9-]*)"`)
	// docFlagPattern matches a `-flag` token (not a value, not `--`).
	docFlagPattern = regexp.MustCompile(`(^|\s)-([a-z][a-z0-9-]*)`)
)

// builtinFlagAllowlist are flags every Go flag set understands implicitly.
var builtinFlagAllowlist = map[string]bool{"h": true, "help": true}

// ExtractKnownGwcFlags returns the set of flag names defined anywhere in the
// gwc launcher source under gwcDir (handling both name-first definitions and
// flag.Var definitions where the name is the second argument).
func ExtractKnownGwcFlags(parseGwcDir string) (map[string]bool, error) {
	parseKnown := map[string]bool{}
	for parseName := range builtinFlagAllowlist {
		parseKnown[parseName] = true
	}
	parseEntries, parseErr := os.ReadDir(parseGwcDir)
	if parseErr != nil {
		return nil, parseErr
	}
	for _, parseEntry := range parseEntries {
		parseName := parseEntry.Name()
		if parseEntry.IsDir() || !strings.HasSuffix(parseName, ".go") || strings.HasSuffix(parseName, "_test.go") {
			continue
		}
		parseData, parseReadErr := os.ReadFile(filepath.Join(parseGwcDir, parseName))
		if parseReadErr != nil {
			return nil, parseReadErr
		}
		parseSource := string(parseData)
		for _, parseMatch := range nameFirstFlagPattern.FindAllStringSubmatch(parseSource, -1) {
			parseKnown[parseMatch[1]] = true
		}
		for _, parseMatch := range typedVarFlagPattern.FindAllStringSubmatch(parseSource, -1) {
			parseKnown[parseMatch[1]] = true
		}
		for _, parseMatch := range varFlagPattern.FindAllStringSubmatch(parseSource, -1) {
			parseKnown[parseMatch[1]] = true
		}
	}
	return parseKnown, nil
}

// ScanDocGwcFlags returns every `-flag` used on a gwc command line in a tracked
// Markdown shell block whose flag name is not in parseKnown. It is the
// flag-existence half (part c) of the doc-drift guard: a renamed or removed
// launcher flag still referenced in docs is reported.
func ScanDocGwcFlags(parseRoot string, parseKnown map[string]bool) ([]FlagRef, error) {
	var parseRefs []FlagRef
	parseErr := filepath.Walk(parseRoot, func(parsePath string, parseInfo os.FileInfo, parseWalkErr error) error {
		if parseWalkErr != nil {
			return parseWalkErr
		}
		if parseInfo.IsDir() {
			if dirsSkipped[parseInfo.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.ToLower(filepath.Ext(parsePath)) != ".md" {
			return nil
		}
		parseRel, parseRelErr := filepath.Rel(parseRoot, parsePath)
		if parseRelErr != nil {
			return parseRelErr
		}
		parseRelSlash := filepath.ToSlash(parseRel)
		if strings.HasPrefix(parseRelSlash, snapshotDocDir) {
			return nil
		}
		parseData, parseReadErr := os.ReadFile(parsePath)
		if parseReadErr != nil {
			return parseReadErr
		}
		parseRefs = append(parseRefs, scanFileFlags(string(parseData), parseRelSlash, parseKnown)...)
		return nil
	})
	if parseErr != nil {
		return nil, parseErr
	}
	return parseRefs, nil
}

// scanFileFlags extracts unknown gwc flags from one Markdown file's shell blocks.
func scanFileFlags(parseContent string, parseDocRel string, parseKnown map[string]bool) []FlagRef {
	var parseRefs []FlagRef
	parseInFence := false
	for parseIndex, parseLine := range strings.Split(parseContent, "\n") {
		if fenceOpen.MatchString(parseLine) {
			parseInFence = !parseInFence
			continue
		}
		if !parseInFence {
			continue
		}
		// Only inspect lines that actually invoke the gwc launcher, so flags for
		// go/node/other tools on neighboring lines are not misread as gwc flags.
		parseArgs, isLauncher := parseDocGwcFlagArgs(parseLine)
		if !isLauncher {
			continue
		}
		for _, parseMatch := range docFlagPattern.FindAllStringSubmatch(strings.Join(parseArgs, " "), -1) {
			parseFlag := parseMatch[2]
			if !parseKnown[parseFlag] {
				parseRefs = append(parseRefs, FlagRef{DocPath: parseDocRel, Line: parseIndex + 1, Flag: parseFlag})
			}
		}
	}
	return parseRefs
}

// parseDocGwcFlagArgs recognizes launcher executables and go-run packages without treating build flags or similarly named tools as launcher arguments.
func parseDocGwcFlagArgs(parseLine string) ([]string, bool) {
	parseFields, parseErr := splitShellFields(stripShellPrompt(parseLine))
	if parseErr != nil || len(parseFields) == 0 {
		return nil, false
	}
	parseExecutable := strings.ReplaceAll(parseFields[0], `\`, "/")
	parseExecutable = strings.ToLower(filepath.Base(parseExecutable))
	if parseExecutable == "gwc" || parseExecutable == "gwc.exe" {
		return parseFields[1:], true
	}
	if parseExecutable != "go" || len(parseFields) < 3 || parseFields[1] != "run" {
		return nil, false
	}
	for parseIndex := 2; parseIndex < len(parseFields); parseIndex++ {
		parseToken := parseFields[parseIndex]
		if strings.HasPrefix(parseToken, "-") {
			// Go's value-bearing build flags precede the package, and belong to Go.
			switch parseToken {
			case "-tags", "-ldflags", "-gcflags", "-asmflags", "-mod", "-modfile", "-overlay", "-p", "-exec", "-C", "-compiler", "-buildmode", "-toolexec", "-pkgdir", "-installsuffix", "-coverpkg":
				parseIndex++
			}
			continue
		}
		parsePackage := strings.TrimPrefix(strings.ReplaceAll(parseToken, `\`, "/"), "./")
		if parsePackage != "tools/gwc" {
			return nil, false
		}
		return parseFields[parseIndex+1:], true
	}
	return nil, false
}
