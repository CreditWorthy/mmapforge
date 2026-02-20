package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/CreditWorthy/mmapforge/internal/codegen"
)

var exitFunc = os.Exit
var stderr io.Writer = os.Stderr

func main() {
	input := flag.String("input", "", "Go source file containing mmapforge-annotated structs")
	output := flag.String("output", "", "Output directory (default: same directory as input)")
	flag.Parse()

	if *input == "" {
		fmt.Fprintln(stderr, "mmapforge: -input flag is required")
		exitFunc(1)
		return
	}

	if err := run(*input, *output); err != nil {
		fmt.Fprintf(stderr, "mmapforge: %v\n", err)
		exitFunc(1)
		return
	}
}

func run(inputPath, outputDir string) error {
	schemas, err := codegen.ParseFile(inputPath)
	if err != nil {
		return err
	}

	if len(schemas) == 0 {
		return fmt.Errorf("no mmapforge:schema directives found in %s", inputPath)
	}

	if outputDir == "" {
		outputDir = filepath.Dir(inputPath)
	}

	g, err := codegen.NewGraph(&codegen.Config{
		Target:  outputDir,
		Package: schemas[0].Package,
	}, schemas)
	if err != nil {
		return err
	}

	if err := g.Gen(); err != nil {
		return err
	}

	for _, n := range g.Nodes {
		fmt.Fprintf(stderr, "mmapforge: %s v%d → %d fields, %d bytes/record\n",
			n.Name, n.SchemaVersion, len(n.Fields), n.RecordSize)
	}

	return nil
}
