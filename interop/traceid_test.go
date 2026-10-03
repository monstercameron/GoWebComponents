package interop

import (
	"strings"
	"testing"
)

func TestNewTraceAndSpanIDShape(parseT *testing.T) {
	parseTrace, parseErr := NewTraceID()
	if parseErr != nil || !IsValidTraceID(parseTrace) {
		parseT.Fatalf("trace id = %q err=%v", parseTrace, parseErr)
	}
	parseSpan, parseErr := NewSpanID()
	if parseErr != nil || !IsValidSpanID(parseSpan) {
		parseT.Fatalf("span id = %q err=%v", parseSpan, parseErr)
	}
	parseOther, _ := NewTraceID()
	if parseOther == parseTrace {
		parseT.Fatal("two trace ids collided")
	}
}

func TestFormatAndParseTraceparentRoundTrip(parseT *testing.T) {
	parseTrace := "0af7651916cd43dd8448eb211c80319c"
	parseSpan := "b7ad6b7169203331"
	parseValue := FormatTraceparent(parseTrace, parseSpan, true)
	if parseValue != "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01" {
		parseT.Fatalf("format = %q", parseValue)
	}
	parseCtx, parseErr := ParseTraceparent(parseValue)
	if parseErr != nil || parseCtx.TraceID != parseTrace || parseCtx.SpanID != parseSpan || !parseCtx.Sampled {
		parseT.Fatalf("parse = %#v err=%v", parseCtx, parseErr)
	}
	if parseCtx, _ = ParseTraceparent(FormatTraceparent(parseTrace, parseSpan, false)); parseCtx.Sampled {
		parseT.Fatal("unsampled parsed as sampled")
	}
}

func TestFormatTraceparentRejectsBadIDs(parseT *testing.T) {
	parseSpan := "b7ad6b7169203331"
	for _, parseTrace := range []string{"", strings.Repeat("0", 32), strings.Repeat("A", 32), strings.Repeat("a", 31), strings.Repeat("g", 32)} {
		if FormatTraceparent(parseTrace, parseSpan, true) != "" {
			parseT.Fatalf("accepted trace id %q", parseTrace)
		}
	}
	if FormatTraceparent(strings.Repeat("a", 32), strings.Repeat("0", 16), true) != "" {
		parseT.Fatal("accepted zero span id")
	}
}

func TestParseTraceparentRejectsMalformed(parseT *testing.T) {
	parseGood := "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01"
	for _, parseValue := range []string{
		"", "garbage", "ff-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01",
		"00-00000000000000000000000000000000-b7ad6b7169203331-01",
		"00-0af7651916cd43dd8448eb211c80319c-0000000000000000-01",
		"00-0AF7651916CD43DD8448EB211C80319C-b7ad6b7169203331-01",
		"00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-zz",
		parseGood + "-extra",
	} {
		if _, parseErr := ParseTraceparent(parseValue); parseErr == nil {
			parseT.Fatalf("accepted %q", parseValue)
		}
	}
	if _, parseErr := ParseTraceparent("01-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01-future"); parseErr != nil {
		parseT.Fatalf("future version with extra fields rejected: %v", parseErr)
	}
}
