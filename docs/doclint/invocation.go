package doclint

import "strings"

// parseGoGwcArgs extracts launcher arguments from a go run invocation that
// targets tools/gwc, including Go's global -C directory option and build flags.
func parseGoGwcArgs(parseFields []string) ([]string, bool) {
	if len(parseFields) < 2 || !strings.EqualFold(parseFields[0], "go") {
		return nil, false
	}
	parseIndex := 1
	parseSawRun := false
	parseChangedDir := false
	for parseIndex < len(parseFields) {
		parseToken := parseFields[parseIndex]
		if !parseSawRun && strings.EqualFold(parseToken, "run") {
			parseSawRun = true
			parseIndex++
			continue
		}
		if parseToken == "-C" {
			if parseIndex+1 >= len(parseFields) {
				return nil, false
			}
			parseChangedDir = strings.EqualFold(parseFields[parseIndex+1], "tools")
			parseIndex += 2
			continue
		}
		if !parseSawRun {
			if !strings.HasPrefix(parseToken, "-") {
				return nil, false
			}
			if !strings.Contains(parseToken, "=") && goRunFlagTakesValue(parseToken) {
				parseIndex++
				if parseIndex >= len(parseFields) {
					return nil, false
				}
			}
			parseIndex++
			continue
		}
		if strings.HasPrefix(parseToken, "-") {
			if !strings.Contains(parseToken, "=") && goRunFlagTakesValue(parseToken) {
				parseIndex++
				if parseIndex >= len(parseFields) {
					return nil, false
				}
			}
			parseIndex++
			continue
		}
		if isToolsGwcToken(parseToken) || (parseChangedDir && normalizeGwcPackage(parseToken) == "gwc") {
			return normalizeCommandPathArgs(parseFields[parseIndex+1:]), true
		}
		return nil, false
	}
	return nil, false
}

// normalizeGwcPackage returns a slash-normalized package token without its
// relative-path prefix.
func normalizeGwcPackage(parseToken string) string {
	parseNormalized := strings.ReplaceAll(strings.Trim(parseToken, "\"'"), `\`, "/")
	return strings.TrimPrefix(parseNormalized, "./")
}

// goRunFlagTakesValue reports whether a documented go run flag consumes the
// following shell field when it is not written with an equals sign.
func goRunFlagTakesValue(parseToken string) bool {
	switch parseToken {
	case "-tags", "-ldflags", "-gcflags", "-asmflags", "-mod", "-modfile", "-overlay", "-p", "-exec", "-C", "-compiler", "-buildmode", "-toolexec", "-pkgdir", "-installsuffix", "-coverpkg":
		return true
	default:
		return false
	}
}
