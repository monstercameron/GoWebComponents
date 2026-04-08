package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	templatelowering "github.com/monstercameron/GoWebComponents/examples/public/browser-compiler/template_lowering"
)

func main() {
	if parseErr := runTemplateLowering(os.Args[1:], os.Stdout, os.Stderr); parseErr != nil {
		fmt.Fprintln(os.Stderr, parseErr)
		os.Exit(1)
	}
}

// runTemplateLowering parses CLI flags and generates lowered source to stdout or a file.
func runTemplateLowering(parseArgs []string, parseStdout io.Writer, parseStderr io.Writer) error {
	parseFlags := flag.NewFlagSet("lower-template", flag.ContinueOnError)
	parseFlags.SetOutput(parseStderr)
	parseInputPath := parseFlags.String("input", "", "path to the constrained HTML-like template")
	parseOutputPath := parseFlags.String("output", "", "path to write the generated Go file; omit to print to stdout")
	parsePackageName := parseFlags.String("package", "templatelowering", "generated package name")
	parseStructName := parseFlags.String("struct", "LandingProps", "generated props struct name")
	parseFuncName := parseFlags.String("func", "RenderLanding", "generated render function name")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		return parseErr
	}
	if *parseInputPath == "" {
		return errors.New("lower-template: -input is required")
	}

	parseOutput, parseErr := templatelowering.GenerateFromFile(*parseInputPath, templatelowering.TemplateConfig{
		PackageName: *parsePackageName,
		StructName:  *parseStructName,
		FuncName:    *parseFuncName,
	})
	if parseErr != nil {
		return parseErr
	}

	if *parseOutputPath == "" {
		_, parseErr2 := fmt.Fprint(parseStdout, parseOutput)
		return parseErr2
	}
	if parseErr2 := os.WriteFile(*parseOutputPath, []byte(parseOutput), 0o644); parseErr2 != nil {
		return parseErr2
	}
	return nil
}
