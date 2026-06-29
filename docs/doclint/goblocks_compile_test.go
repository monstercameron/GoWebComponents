//go:build doccompile

package doclint

import (
	"os"
	"testing"
)

// TestDocsMarkedSamplesCompile is the F1 type-level gate: every `gwc:build`-marked complete-file
// doc sample must COMPILE against the real module (not just parse), so a renamed/removed API used
// by a runnable sample fails CI even though the sample is still syntactically valid Go. Behind
// the `doccompile` build tag because it shells out to `go build` per sample; the dedicated
// doc-samples.yml workflow runs it. There must be at least one marked sample so the gate is real.
func TestDocsMarkedSamplesCompile(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	root, ok := FindRepoRoot(wd)
	if !ok {
		t.Fatalf("module root not found")
	}

	blocks, err := ScanGoBlocks(root)
	if err != nil {
		t.Fatalf("scan go blocks: %v", err)
	}
	marked := 0
	for _, b := range blocks {
		if b.Compile {
			marked++
		}
	}
	if marked == 0 {
		t.Fatal("no gwc:build-marked doc samples found; the compile gate would be a no-op (mark canonical runnable samples)")
	}

	errs, err := CompileMarkedGoBlocks(root)
	if err != nil {
		t.Fatalf("compile marked samples: %v", err)
	}
	if len(errs) > 0 {
		for _, e := range errs {
			t.Errorf("%s:%d sample failed to compile:\n%s", e.DocPath, e.Line, e.Err)
		}
		t.Fatalf("%d gwc:build-marked doc sample(s) no longer compile", len(errs))
	}
	t.Logf("compiled %d gwc:build-marked doc sample(s) clean", marked)
}
