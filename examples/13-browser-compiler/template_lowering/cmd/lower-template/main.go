package main

import (
	"flag"
	"fmt"
	"os"

	templatelowering "github.com/monstercameron/GoWebComponents/examples/13-browser-compiler/template_lowering"
)

func main() {
	parseInputPath := flag.String("input", "", "path to the constrained HTML-like template")
	parseOutputPath := flag.String("output", "", "path to write the generated Go file; omit to print to stdout")
	parsePackageName := flag.String("package", "templatelowering", "generated package name")
	parseStructName := flag.String("struct", "LandingProps", "generated props struct name")
	parseFuncName := flag.String("func", "RenderLanding", "generated render function name")
	flag.Parse()

	if *parseInputPath == "" {
		fmt.Fprintln(os.Stderr, "lower-template: -input is required")
		os.Exit(1)
	}

	parseOutput, parseErr := templatelowering.GenerateFromFile(*parseInputPath, templatelowering.TemplateConfig{
		PackageName: *parsePackageName,
		StructName:  *parseStructName,
		FuncName:    *parseFuncName,
	})
	if parseErr != nil {
		fmt.Fprintln(os.Stderr, parseErr)
		os.Exit(1)
	}

	if *parseOutputPath == "" {
		fmt.Print(parseOutput)
		return
	}
	if parseErr2 := os.WriteFile(*parseOutputPath, []byte(parseOutput), 0o644); parseErr2 != nil {
		fmt.Fprintln(os.Stderr, parseErr2)
		os.Exit(1)
	}
}
