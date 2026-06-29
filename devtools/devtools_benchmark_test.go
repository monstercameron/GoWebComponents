package devtools

import "testing"

func benchmarkBugCaptureBundle() BugCaptureBundle {
	return BugCaptureBundle{
		Version:    1,
		Label:      "bench",
		CapturedAt: "2026-03-25T00:00:00Z",
		Trace: TraceCapture{
			Label:      "trace",
			CapturedAt: "2026-03-25T00:00:00Z",
			Snapshot: Snapshot{
				Route: Route{
					Params: map[string]string{
						"user":  "123",
						"token": "Bearer abc.def.ghi",
					},
					Query: map[string][]string{
						"q": {"authorization=top-secret"},
					},
				},
				Diagnostics: []Diagnostic{
					{Message: "password=super-secret"},
				},
				Logs: []Log{
					{
						Message: "authorization: Bearer abc.def.ghi",
						Fields: map[string]string{
							"cookie": "session=secret",
						},
					},
				},
			},
		},
	}
}

func BenchmarkSanitizeBugCaptureBundleForSupport(parseB *testing.B) {
	parseB.ReportAllocs()
	parseBundle := benchmarkBugCaptureBundle()
	for parseB.Loop() {
		parseSupport := SanitizeBugCaptureBundleForSupport(parseBundle)
		if !parseSupport.Sanitized {
			parseB.Fatal("expected sanitized support bundle")
		}
	}
}

func BenchmarkExportSupportDiagnosticBundleJSON(parseB *testing.B) {
	parseB.ReportAllocs()
	parseBundle := benchmarkBugCaptureBundle()
	for parseB.Loop() {
		parseData, parseErr := ExportSupportDiagnosticBundleJSON(parseBundle)
		if parseErr != nil {
			parseB.Fatalf("ExportSupportDiagnosticBundleJSON: %v", parseErr)
		}
		if len(parseData) == 0 {
			parseB.Fatal("expected non-empty support JSON payload")
		}
	}
}
