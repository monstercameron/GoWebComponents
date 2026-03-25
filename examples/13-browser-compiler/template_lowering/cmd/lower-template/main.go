package main

import (
	"flag"
	"fmt"
	"os"

	templatelowering "github.com/monstercameron/GoWebComponents/examples/13-browser-compiler/template_lowering"
)

func main() {
	inputPath := flag.String("input", "", "path to the constrained HTML-like template")
	outputPath := flag.String("output", "", "path to write the generated Go file; omit to print to stdout")
	packageName := flag.String("package", "templatelowering", "generated package name")
	structName := flag.String("struct", "LandingProps", "generated props struct name")
	funcName := flag.String("func", "RenderLanding", "generated render function name")
	flag.Parse()

	if *inputPath == "" {
		fmt.Fprintln(os.Stderr, "lower-template: -input is required")
		os.Exit(1)
	}

	output, err := templatelowering.GenerateFromFile(*inputPath, templatelowering.TemplateConfig{
		PackageName: *packageName,
		StructName:  *structName,
		FuncName:    *funcName,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if *outputPath == "" {
		fmt.Print(output)
		return
	}
	if err := os.WriteFile(*outputPath, []byte(output), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
