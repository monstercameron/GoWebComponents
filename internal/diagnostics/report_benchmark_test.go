package diagnostics

import "testing"

func BenchmarkBuild(t *testing.B) {
	t.ReportAllocs()
	options := Options{
		Summary:  "state snapshot decode failed",
		Code:     "snapshot_decode_failed",
		Headline: "Could not decode snapshot payload",
		Path:     "state.UnmarshalSnapshotJSON",
		Runtime:  "decode aborted before restore",
		Next:     "verify payload shape",
		Docs:     "ACTIONABLE_ERRORS.md#snapshot-decode",
	}
	for t.Loop() {
		_ = Build(options)
	}
}

func BenchmarkReportFormatted(t *testing.B) {
	t.ReportAllocs()
	report := Build(Options{
		Summary:  "hook called outside component",
		Code:     "hook_outside_component",
		Headline: "Hook misuse",
		Path:     "ui.UseState",
		Runtime:  "hook invocation blocked",
		Next:     "move call into render path",
		Docs:     "ACTIONABLE_ERRORS.md#gwc-runtime-hook-outside-component",
	})
	for t.Loop() {
		_ = report.Formatted()
	}
}
