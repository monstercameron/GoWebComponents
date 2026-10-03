//go:build !js

package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// TestMain doubles as a stub "node": when GOWASMEXEC_STUB is set it echoes its arguments.
func TestMain(parseM *testing.M) {
	if os.Getenv("GOWASMEXEC_STUB") == "1" {
		for _, parseArg := range os.Args[1:] {
			os.Stdout.WriteString("ARG[" + parseArg + "]\n")
		}
		if parseCode := os.Getenv("GOWASMEXEC_EXIT"); parseCode == "7" {
			os.Exit(7)
		}
		os.Exit(0)
	}
	os.Exit(parseM.Run())
}

func TestWasmExecForwardsArgsVerbatim(parseT *testing.T) {
	parseT.Setenv("GOWASMEXEC_STUB", "1")
	parseT.Setenv("GOWASMEXEC_EXIT", "7")
	var parseOut, parseErrBuf bytes.Buffer
	parseArgs := []string{"x.wasm", "-test.run=^$", "-test.bench=A|B", "-test.v=true", "a b", `c"d`}
	parseCode := runWasmExec(os.Args[0], "script.js", parseArgs, strings.NewReader(""), &parseOut, &parseErrBuf)
	if parseCode != 7 {
		parseT.Fatalf("exit code = %d, want 7", parseCode)
	}
	parseWant := "ARG[script.js]\n"
	for _, parseArg := range parseArgs {
		parseWant += "ARG[" + parseArg + "]\n"
	}
	if parseOut.String() != parseWant {
		parseT.Fatalf("args not verbatim:\n got %q\nwant %q", parseOut.String(), parseWant)
	}
}

func TestWasmExecReportsStartFailure(parseT *testing.T) {
	var parseOut, parseErrBuf bytes.Buffer
	if parseCode := runWasmExec("definitely-not-a-binary-xyz", "s.js", nil, strings.NewReader(""), &parseOut, &parseErrBuf); parseCode != 1 {
		parseT.Fatalf("exit code = %d, want 1", parseCode)
	}
}
