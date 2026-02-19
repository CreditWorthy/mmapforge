package codegen

import (
	"fmt"
	"os"

	"github.com/CreditWorthy/mmapforge"
)

type Generator interface {
	Generate(*Graph) error
}

// GenerateFunc adapts an ordinary function to the Generator interface.
type GenerateFunc func(*Graph) error

// Generate calls f(g).
func (f GenerateFunc) Generate(g *Graph) error { return f(g) }

type Hook func(Generator) Generator

type Graph struct {
	*Config

	Nodes []*Type
}

func NewGraph(c *Config, schemas []StructSchema) (*Graph, error) {
	if c.Target == "" {
		return nil, fmt.Errorf("mmapforge: codegen: target directory is required")
	}

	g := &Graph{
		Config: c,
		Nodes:  make([]*Type, len(schemas)),
	}

	for _, s := range schemas {
		layout, err := mmapforge.ComputeLayout(s.Fields)
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

func (g *Graph) Gen() error {
	var gen Generator = GenerateFunc(generate)
	for i := len(g.Hooks) - 1; i >= 0; i-- {
		gen = g.Hooks[i](gen)
	}
	return gen.Generate(g)
}

func generate(g *Graph) error {
	if err := os.MkdirAll(g.Target, os.ModePerm); err != nil {
		return fmt.Errorf("mmapforge: create target dir: %w", err)
	}

	initTemplates()

	//for _, ext := range g.Templates {
	//	templates.Funcs(ext.FuncMap)
	//	for _, tmpl := range ext.Templates() {
	//		if parse.IsEmptyTree(tmpl.Root) {
	//			continue
	//		}
	//		templates = MustParse(templates.AddParseTree(tmpl.Name(), tmpl.Tree))
	//	}
	//}
	//

	return nil
}
