package codegen

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"text/template/parse"

	"github.com/CreditWorthy/mmapforge"
)

// Mockable functions
var computeLayoutFunc = mmapforge.ComputeLayout
var mkdirAllFunc = os.MkdirAll
var writeFileFunc = os.WriteFile
var generateFunc = generate
var writeFormattedFunc = writeFormatted

// Generator is the interface for codegen from a Graph.
type Generator interface {
	Generate(*Graph) error
}

// GenerateFunc adapts an ordinary function to the Generator interface.
type GenerateFunc func(*Graph) error

// Generate calls f(g).
func (f GenerateFunc) Generate(g *Graph) error { return f(g) }

// Hook is "generate middleware" - wraps a Generator to inject logic
type Hook func(Generator) Generator

// Graph holds all Type nodes and derive code generation.
type Graph struct {
	*Config

	Nodes []*Type
}

// NewGraph builds a Graph from parsed schemas and config.
// It computes layouts, builds rich Type/Field objects, and validates.
func NewGraph(c *Config, schemas []StructSchema) (*Graph, error) {
	if c.Target == "" {
		return nil, fmt.Errorf("mmapforge: codegen: target directory is required")
	}

	g := &Graph{
		Config: c,
		Nodes:  make([]*Type, 0, len(schemas)),
	}

	for _, s := range schemas {
		layout, err := computeLayoutFunc(s.Fields)
		if err != nil {
			return nil, fmt.Errorf("mmapforge: compute layout for %s: %w", s.Name, err)
		}
		fields := make([]*Field, len(layout.Fields))
		for i := range layout.Fields {
			fields[i] = &Field{
				FieldLayout: layout.Fields[i],
			}
		}
		pkg := s.Package
		if c.Package != "" {
			pkg = c.Package
		}
		g.Nodes = append(g.Nodes, &Type{
			Config:        c,
			Name:          s.Name,
			Package:       pkg,
			SchemaVersion: s.SchemaVersion,
			Fields:        fields,
			RecordSize:    layout.RecordSize,
		})
	}

	return g, nil
}

// Gen generates all artifacts. Hooks wrap the core generation
func (g *Graph) Gen() error {
	var gen Generator = GenerateFunc(generateFunc)
	for i := len(g.Hooks) - 1; i >= 0; i-- {
		gen = g.Hooks[i](gen)
	}
	return gen.Generate(g)
}

func generate(g *Graph) error {
	if err := mkdirAllFunc(g.Target, os.ModePerm); err != nil {
		return fmt.Errorf("mmapforge: create target dir: %w", err)
	}

	initTemplates()

	for _, ext := range g.Templates {
		templates.Funcs(ext.FuncMap)
		for _, tmpl := range ext.Templates() {
			if parse.IsEmptyTree(tmpl.Tree.Root) {
				continue
			}
			templates = MustParse(templates.AddParseTree(tmpl.Name(), tmpl.Tree))
		}
	}

	for _, node := range g.Nodes {
		for _, tmpl := range TypeTemplates {
			if tmpl.Cond != nil && !tmpl.Cond(node) {
				continue
			}
			b := bytes.NewBuffer(nil)
			if err := templates.ExecuteTemplate(b, tmpl.Name, node); err != nil {
				return fmt.Errorf("mmapforge: execute %q for %s: %w", tmpl.Name, node.Name, err)
			}
			path := filepath.Join(g.Target, tmpl.Format(node))
			if err := writeFormattedFunc(path, b.Bytes()); err != nil {
				return err
			}
		}
	}

	for _, tmpl := range GraphTemplates {
		if tmpl.Skip != nil && tmpl.Skip(g) {
			continue
		}

		b := bytes.NewBuffer(nil)
		if err := templates.ExecuteTemplate(b, tmpl.Name, g); err != nil {
			return fmt.Errorf("mmapforge: execute %q: %w", tmpl.Name, err)
		}
		path := filepath.Join(g.Target, tmpl.Format)
		if err := writeFormattedFunc(path, b.Bytes()); err != nil {
			return err
		}
	}

	return nil
}

// writeFormatted writes Go source to a file, running gofmt first.
func writeFormatted(path string, src []byte) error {
	formatted, err := format.Source(src)
	if err != nil {
		writeErr := writeFileFunc(path, src, 0644)
		return errors.Join(
			fmt.Errorf("mmapforge: format %s: %w", path, err),
			fmt.Errorf("mmapforge: write %s: %w", path, writeErr),
		)
	}
	return writeFileFunc(path, formatted, 0644)
}
