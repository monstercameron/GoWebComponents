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

func BenchmarkSanitizeBugCaptureBundleForSupport(b *testing.B) {
	b.ReportAllocs()
	bundle := benchmarkBugCaptureBundle()
	for b.Loop() {
		support := SanitizeBugCaptureBundleForSupport(bundle)
		if !support.Sanitized {
			b.Fatal("expected sanitized support bundle")
		}
	}
}

func BenchmarkExportSupportDiagnosticBundleJSON(b *testing.B) {
	b.ReportAllocs()
	bundle := benchmarkBugCaptureBundle()
	for b.Loop() {
		data, err := ExportSupportDiagnosticBundleJSON(bundle)
		if err != nil {
			b.Fatalf("ExportSupportDiagnosticBundleJSON: %v", err)
		}
		if len(data) == 0 {
			b.Fatal("expected non-empty support JSON payload")
		}
	}
}
