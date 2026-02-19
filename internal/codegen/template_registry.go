package codegen

import "strings"

type (
	// TypeTemplate is executed once per Type node.
	TypeTemplate struct {
		Name   string             // matches a {{ define "name" }} in a .tmpl file
		Cond   func(*Type) bool   // optional: skip if returns false
		Format func(*Type) string // output file name
	}

	// GraphTemplate is executed once for the whole Graph.
	GraphTemplate struct {
		Name   string            // matches a {{ define "name" }} in a .tmpl file
		Skip   func(*Graph) bool // optional: skip if returns true
		Format string            // output file name
	}
)

// TypeTemplates is the list of per-type templates to execute.
var TypeTemplates = []TypeTemplate{
	{
		Name: "store",
		Format: func(t *Type) string {
			return strings.ToLower(t.Name) + "_store.go"
		},
	},
}

// GraphTemplates is the list of graph-wide templates to execute.
var GraphTemplates = []GraphTemplate{}
