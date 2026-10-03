package validate

import (
	"regexp"
	"strings"
	"testing"
)

// The oracle regexps are the exact patterns the hand-written matchers replaced.
// They live only in this test file so regexp stays out of the wasm dependency graph.
var (
	oracleEmail    = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	oracleURL      = regexp.MustCompile(`^https?://[^\s]+$`)
	oracleAlpha    = regexp.MustCompile(`^[A-Za-z]+$`)
	oracleAlphanum = regexp.MustCompile(`^[A-Za-z0-9]+$`)
	oracleNumeric  = regexp.MustCompile(`^[0-9]+$`)
)

// buildMatcherCorpus returns hand-picked edge cases plus deterministic pseudo-random strings.
func buildMatcherCorpus() []string {
	parseCorpus := []string{
		"", " ", "\t", "\n", "\r\n", "\v", "\f", "\x00", "a\x00b", "@", "@@", ".", "..", "a", "A", "0", "9",
		"a@b", "a@b.", "a@.b", "a@b.c", "a@.c", "a@..", "a@...", "a@b..c", "a@@b.c", "a@b@c.d", "@b.c", "a@.",
		"a b@c.d", "a@b c.d", "a@b.c ", " a@b.c", "a@b.c\n", "a@b.c\t", "a\n@b.c", "a@b.c\v", "a@b\v.c",
		"a@b .c", "a@b.c ", "a@b.c\u0085", "a@b.c　",
		"user+tag@example.co.uk", "a.b@c.d.e", "é@ü.ö", "a@b.ĉ", "ａ@ｂ.ｃ", "a@b.́", "́@́.́",
		"أ@ب.ج", "a@ب.ج", "‮a@b.c", "a@b.c‏", "a@b.�", "�@�.�",
		"a@b.\xff", "\xff@\xff.\xff", "a@\xff.b", "\xc3@b.c", "a@b.\xc3", "\xed\xa0\x80@b.c", "a@b.\xed\xa0\x80",
		"\U0001F600@\U0001F600.\U0001F600", "a@b.\U0001F600", "퟿@퟿.퟿", "@.",
		"http://", "https://", "http://a", "https://a", "http://a b", "https://a\n", "https:// a", "http:/a", "http//a",
		"HTTP://a", "Http://a", "ftp://a", "ttp://a", "xhttp://a", "http://\x00", "http://é", "https://　",
		"http:// ", "http://a\u0085", "http://a\v", "http://\xff", "http://�", "https://日本語.jp/パス?q=1#f",
		"http://ａ", "httpss://a", "http://http://", "https://https://a", " http://a", "http://a ", "http://a\t",
		"http://a\f", "http://a\r", "https://example.com/a b",
		"abc", "ABC", "abcXYZ", "abc1", "123", "abc 1", "abc_", "abc-", "ab\ncd", "ab\x00", "é", "ñandú", "٣", "１２３", "ａｂｃ",
		"abć", "á", "́", "שלום", "abc\xff", "\xff", "Z", "z", "@", "[", "`", "{", "/", ":", "0123456789",
		"abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ", "abc​", "\U0001F600",
		"٠١", "٠١٢", "123\n", "\n123", "123 ", "1e5", "-1", "+1", "1.5", "1,5",
	}
	for _, parseN := range []int{1, 2, 3, 63, 64, 255, 1000, 4999, 5000} {
		parseCorpus = append(parseCorpus,
			strings.Repeat("a", parseN),
			strings.Repeat("7", parseN),
			strings.Repeat("a1", parseN/2+1)[:parseN],
			strings.Repeat("é", parseN/2+1),
			"http://"+strings.Repeat("x", parseN),
			"https://"+strings.Repeat("x", parseN)+" ",
			strings.Repeat("a", parseN)+"@"+strings.Repeat("b", parseN)+"."+strings.Repeat("c", parseN),
			strings.Repeat("a", parseN)+"@"+strings.Repeat("b", parseN),
			strings.Repeat("a", parseN)+"@"+strings.Repeat(".", parseN),
			"a@"+strings.Repeat("b.", parseN),
			"a@"+strings.Repeat("\xff", parseN)+".b",
		)
	}
	// Deterministic xorshift so every run sees the same corpus.
	parseState := uint64(0x9E3779B97F4A7C15)
	parseNext := func() uint64 {
		parseState ^= parseState << 13
		parseState ^= parseState >> 7
		parseState ^= parseState << 17
		return parseState
	}
	parseAtoms := []string{
		"a", "Z", "m", "0", "5", "9", "@", "@", ".", ".", ":", "/", "http://", "https://", "http", "s", " ", "\t", "\n",
		"\r", "\f", "\v", "\x00", "-", "_", "é", "٣", "１", "́", "‏", "　", " ", "\U0001F600",
		"\xff", "\xc3", "\xed\xa0\x80", "�", "x.y", "a@b.c",
	}
	for parseI := 0; parseI < 20000; parseI++ {
		parseLen := int(parseNext() % 12)
		var parseB strings.Builder
		for parseJ := 0; parseJ < parseLen; parseJ++ {
			parseB.WriteString(parseAtoms[parseNext()%uint64(len(parseAtoms))])
		}
		parseCorpus = append(parseCorpus, parseB.String())
	}
	// Raw random bytes, including invalid UTF-8.
	for parseI := 0; parseI < 5000; parseI++ {
		parseLen := int(parseNext() % 10)
		parseBytes := make([]byte, parseLen)
		for parseJ := range parseBytes {
			parseBytes[parseJ] = byte(parseNext())
		}
		parseCorpus = append(parseCorpus, string(parseBytes))
	}
	return parseCorpus
}

// TestHandWrittenMatchersMatchRegexpOracle proves each hand-written matcher agrees with
// the regexp it replaced on a large deterministic corpus.
func TestHandWrittenMatchersMatchRegexpOracle(t *testing.T) {
	parseRules := []struct {
		name   string
		fn     func(string) bool
		oracle *regexp.Regexp
	}{
		{"email", isEmailString, oracleEmail},
		{"url", isURLString, oracleURL},
		{"alpha", isAlphaString, oracleAlpha},
		{"alphanum", isAlphanumString, oracleAlphanum},
		{"numeric", isNumericString, oracleNumeric},
	}
	parseCorpus := buildMatcherCorpus()
	for _, parseRule := range parseRules {
		parseTrue := 0
		for _, parseIn := range parseCorpus {
			parseWant := parseRule.oracle.MatchString(parseIn)
			if parseWant {
				parseTrue++
			}
			if parseGot := parseRule.fn(parseIn); parseGot != parseWant {
				t.Fatalf("%s(%q): got %v, regexp oracle says %v", parseRule.name, parseIn, parseGot, parseWant)
			}
		}
		if parseTrue == 0 {
			t.Errorf("%s: corpus never produced a match; the differential test is vacuous", parseRule.name)
		}
	}
}

func TestHandWrittenMatchersDoNotAllocate(t *testing.T) {
	parseIn := "someone@example.com"
	parseAllocs := testing.AllocsPerRun(100, func() {
		isEmailString(parseIn)
		isURLString("https://example.com/x")
		isAlphaString("abc")
		isAlphanumString("abc123")
		isNumericString("123")
	})
	if parseAllocs != 0 {
		t.Fatalf("matchers allocate: %v allocs/run", parseAllocs)
	}
}
