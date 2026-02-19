package codegen

import "github.com/CreditWorthy/mmapforge"

type Type struct {
	*Config
	Name          string
	Package       string
	Fields        []*Field
	SchemaVersion uint32
	RecordSize    uint32
}

type Field struct {
	mmapforge.FieldLayout
}
