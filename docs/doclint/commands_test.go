package doclint

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanDocGwcCommandsParsesFencedCommands(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "README.md"), "# Doc\n\n```powershell\n"+
		"go run ./tools/gwc build -app \".\\examples\\public\\counter\\main.go\" -profile development\n"+
		"gwc dev -app ./examples/public/counter/main.go -dry-run\n"+
		"go test ./...\n"+
		"```\n\n```go\n"+
		"func main() { println(\"gwc dev is prose here\") }\n"+
		"```\n")

	refs, err := ScanDocGwcCommands(root)
	if err != nil {
		t.Fatalf("scan commands: %v", err)
	}
	if len(refs) != 2 {
		t.Fatalf("expected 2 gwc command refs, got %d: %+v", len(refs), refs)
	}
	if refs[0].Args[0] != "build" || refs[1].Args[0] != "dev" {
		t.Fatalf("unexpected command order: %+v", refs)
	}
	if !strings.Contains(filepath.ToSlash(refs[0].Args[2]), "examples/public/counter/main.go") {
		t.Fatalf("expected windows path to be normalized, got args %+v", refs[0].Args)
	}
}

func TestPlanDocGwcCommandsClassifiesAndSandboxesOutputs(t *testing.T) {
	outRoot := t.TempDir()
	refs := []GwcCommandRef{
		{DocPath: "README.md", Line: 10, Args: []string{"build", "-app", filepath.FromSlash("./examples/public/counter/main.go"), "-out", filepath.FromSlash("./bin/counter.wasm")}},
		{DocPath: "README.md", Line: 11, Args: []string{"build", "-app", filepath.FromSlash("./main.go")}},
		{DocPath: "README.md", Line: 12, Args: []string{"build", "-app", filepath.FromSlash("./examples/public/counter/main.go"), "-profile", "tinygo"}},
		{DocPath: "README.md", Line: 13, Args: []string{"dev", "-app", filepath.FromSlash("./examples/public/counter/main.go")}},
		{DocPath: "README.md", Line: 14, Args: []string{"dev", "-app", filepath.FromSlash("./examples/public/counter/main.go"), "-dry-run"}},
		{DocPath: "README.md", Line: 15, Args: []string{"examples"}},
		{DocPath: "README.md", Line: 16, Args: []string{"examples", "-export-static-catalog", filepath.FromSlash("./bin/catalog.json")}},
		{DocPath: "README.md", Line: 17, Args: []string{"doctor"}},
		{DocPath: "README.md", Line: 18, Args: []string{"doctor", "-h"}},
		{DocPath: "README.md", Line: 19, Args: []string{"release", "-app", filepath.FromSlash("./examples/public/counter/main.go")}},
	}

	plans := PlanDocGwcCommands(refs, outRoot)
	assertPlanExecutable(t, plans[0], "-out", outRoot)
	assertPlanSkipped(t, plans[1], "placeholder")
	assertPlanSkipped(t, plans[2], "tinygo")
	assertPlanSkipped(t, plans[3], "long-running")
	if plans[4].SkipReason != "" {
		t.Fatalf("expected dev -dry-run to be executable, got skip %q", plans[4].SkipReason)
	}
	assertPlanSkipped(t, plans[5], "long-running")
	assertPlanExecutable(t, plans[6], "-export-static-catalog", outRoot)
	assertPlanSkipped(t, plans[7], "prerequisites")
	if plans[8].SkipReason != "" {
		t.Fatalf("expected help command to be executable, got skip %q", plans[8].SkipReason)
	}
	assertPlanSkipped(t, plans[9], "release")
}

func TestRunDocGwcCommandsWithFakeRunnerReportsFailures(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "README.md"), "# Doc\n\n```powershell\n"+
		"go run ./tools/gwc files -root . -ext go\n"+
		"go run ./tools/gwc build -app .\\examples\\public\\counter\\main.go\n"+
		"go run ./tools/gwc dev -app .\\examples\\public\\counter\\main.go\n"+
		"```\n")

	var ran []string
	runner := func(ctx context.Context, dir string, env []string, name string, args []string) (string, error) {
		_ = ctx
		if dir != root {
			t.Fatalf("expected command dir %q, got %q", root, dir)
		}
		if name != "go" {
			t.Fatalf("expected go runner, got %q", name)
		}
		joined := strings.Join(args, " ")
		ran = append(ran, joined)
		if strings.Contains(joined, " build ") {
			return "build failed", errors.New("planted failure")
		}
		return "ok", nil
	}

	summary, err := RunDocGwcCommands(root, GwcCommandRunOptions{
		OutputRoot: t.TempDir(),
		Runner:     runner,
	})
	if err != nil {
		t.Fatalf("run doc commands: %v", err)
	}
	if len(summary.Executed) != 2 {
		t.Fatalf("expected 2 executed commands, got %d: %+v", len(summary.Executed), summary.Executed)
	}
	if len(summary.Skipped) != 1 || !strings.Contains(summary.Skipped[0].Plan.SkipReason, "long-running") {
		t.Fatalf("expected dev command to be skipped as long-running, got %+v", summary.Skipped)
	}
	if summary.Executed[1].Err == nil || !strings.Contains(summary.Executed[1].Output, "build failed") {
		t.Fatalf("expected planted build failure to be reported, got %+v", summary.Executed[1])
	}
	if len(ran) != 2 || !strings.Contains(ran[1], "-out") {
		t.Fatalf("expected build command to be run with sandboxed -out, ran %+v", ran)
	}
}

func assertPlanSkipped(t *testing.T, plan GwcCommandPlan, want string) {
	t.Helper()
	if !strings.Contains(plan.SkipReason, want) {
		t.Fatalf("expected skip reason containing %q for %+v, got %q", want, plan.Ref.Args, plan.SkipReason)
	}
}

func assertPlanExecutable(t *testing.T, plan GwcCommandPlan, outputFlag string, outRoot string) {
	t.Helper()
	if plan.SkipReason != "" {
		t.Fatalf("expected executable plan for %+v, got skip %q", plan.Ref.Args, plan.SkipReason)
	}
	for i, arg := range plan.Args {
		if arg == outputFlag && i+1 < len(plan.Args) {
			if !strings.HasPrefix(plan.Args[i+1], outRoot) {
				t.Fatalf("expected %s output under %q, got args %+v", outputFlag, outRoot, plan.Args)
			}
			return
		}
	}
	t.Fatalf("expected output flag %s in args %+v", outputFlag, plan.Args)
}
